package connect

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"unsafe"

	"github.com/kitdot/nsh/core"

	"golang.org/x/term"
)

var passwordPromptRe = regexp.MustCompile(`(?i)(password|passphrase)\s*:\s*$`)

const maxOutputAccum = 4096 // 4KB buffer cap

// Exec performs auth-aware SSH connection (replaces current process)
func Exec(host *core.NSHHost) {
	switch host.Auth {
	case "password":
		execWithPassword(host)
	case "key":
		execWithKey(host)
	default:
		execDefault(host.Alias)
	}
}

// execWithPassword uses PTY to auto-fill password
func execWithPassword(host *core.NSHHost) {
	password, ok := core.KeychainGetPassword(host.Alias)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: No password found in Keychain for '%s'.\n", host.Alias)
		fmt.Fprintf(os.Stderr, "Set password with: nsh auth %s\n", host.Alias)
		os.Exit(1)
	}

	sshPath := resolveSSHPath()

	// Create PTY
	pty, tty, err := openPTY()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create PTY: %v\n", err)
		os.Exit(1)
	}

	// Fork child process
	pid, err := syscall.ForkExec(sshPath, []string{"ssh", host.Alias}, &syscall.ProcAttr{
		Files: []uintptr{tty, tty, tty},
		Sys: &syscall.SysProcAttr{
			Setsid:  true,
			Setctty: true,
			Ctty:    0,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to fork: %v\n", err)
		os.Exit(1)
	}
	syscall.Close(int(tty))

	// Save and set raw terminal
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	passwordSent := false
	var outputAccum []byte
	passwordBytes := []byte(password + "\r")

	// Main loop: bridge PTY and stdin using poll-like read
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				syscall.Write(int(pty), buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	buf := make([]byte, 4096)
	for {
		n, err := syscall.Read(int(pty), buf)
		if n <= 0 || err != nil {
			break
		}

		if !passwordSent {
			outputAccum = append(outputAccum, buf[:n]...)
			// Cap buffer to prevent unbounded growth
			if len(outputAccum) > maxOutputAccum {
				outputAccum = outputAccum[len(outputAccum)-maxOutputAccum:]
			}
			text := string(outputAccum)
			if passwordPromptRe.MatchString(text) {
				syscall.Write(int(pty), passwordBytes)
				passwordSent = true
				outputAccum = nil
			}
		}

		os.Stdout.Write(buf[:n])
	}

	// Wait for child
	var ws syscall.WaitStatus
	syscall.Wait4(pid, &ws, 0, nil)
	syscall.Close(int(pty))

	if ws.Exited() {
		os.Exit(ws.ExitStatus())
	}
	os.Exit(1)
}

// execWithKey auto-adds the key to ssh-agent then connects
func execWithKey(host *core.NSHHost) {
	if host.IdentityFile != "" {
		keyPath := core.ExpandPath(host.IdentityFile)
		fixKeyPermissions(keyPath)
		sshAddKey(keyPath)
	}
	execDefault(host.Alias)
}

// execDefault does a plain ssh exec
func execDefault(alias string) {
	sshPath := resolveSSHPath()
	err := syscall.Exec(sshPath, []string{"ssh", alias}, os.Environ())
	if err != nil {
		fmt.Fprintf(os.Stderr, "nsh: failed to exec ssh: %v\n", err)
		os.Exit(1)
	}
}

func resolveSSHPath() string {
	if path, err := exec.LookPath("ssh"); err == nil {
		return path
	}
	return "/usr/bin/ssh"
}

func fixKeyPermissions(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Mode().Perm()&0077 != 0 {
		os.Chmod(path, 0600)
	}
}

func sshAddKey(keyPath string) {
	// Check if key is already loaded
	out, err := exec.Command("/usr/bin/ssh-add", "-l").Output()
	if err == nil {
		output := string(out)
		if strings.Contains(output, keyPath) {
			return
		}
	}

	// Add key with Apple Keychain integration
	cmd := exec.Command("/usr/bin/ssh-add", "--apple-use-keychain", keyPath)
	cmd.Run()
}

// openPTY opens a pseudo-terminal pair
func openPTY() (master uintptr, slave uintptr, err error) {
	// Use posix_openpt approach via /dev/ptmx
	masterFd, err := syscall.Open("/dev/ptmx", syscall.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("open /dev/ptmx: %w", err)
	}

	// grantpt and unlockpt via ioctl
	// On macOS, we need to use TIOCPTYGRANT and TIOCPTYUNLK
	const TIOCPTYGRANT = 0x20007454
	const TIOCPTYUNLK = 0x20007452
	const TIOCPTYGNAME = 0x40807453

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(masterFd), TIOCPTYGRANT, 0); errno != 0 {
		syscall.Close(masterFd)
		return 0, 0, fmt.Errorf("grantpt: %v", errno)
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(masterFd), TIOCPTYUNLK, 0); errno != 0 {
		syscall.Close(masterFd)
		return 0, 0, fmt.Errorf("unlockpt: %v", errno)
	}

	// Get slave name
	nameBuf := make([]byte, 128)
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(masterFd), TIOCPTYGNAME, uintptr(unsafe.Pointer(&nameBuf[0]))); errno != 0 {
		syscall.Close(masterFd)
		return 0, 0, fmt.Errorf("ptsname: %v", errno)
	}

	slaveName := cString(nameBuf)
	slaveFd, err := syscall.Open(slaveName, syscall.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		syscall.Close(masterFd)
		return 0, 0, fmt.Errorf("open slave %s: %w", slaveName, err)
	}

	return uintptr(masterFd), uintptr(slaveFd), nil
}

func cString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

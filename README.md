# nsh (Go CLI)

[English](README.md) | [繁體中文](README.zh-TW.md)

`nsh` is a macOS-only SSH host management CLI built with Go, Cobra, and Bubble Tea.

Managed hosts are stored in `~/.ssh/nsh/config`, and `nsh` automatically injects `Include ~/.ssh/nsh/config` into your main `~/.ssh/config`. You can organize hosts by group, description, auth mode, and ordering tags while preserving existing SSH config formatting.

## Highlights

- Interactive TUI: browse, search, connect, edit, delete, pin
- Full host lifecycle: `new` / `copy` / `edit` / `del` / `auth`
- Import/export: plain config export and encrypted full backups (passwords + keys)
- Security features: Keychain, Touch ID, atomic write, rotating backups
- Main config compatibility: preserves existing `Include`, `Match`, comments, and layout

## Requirements

- macOS (Keychain, Touch ID, `ssh-add --apple-use-keychain`)

## Install

```bash
brew tap kitdot/tap
brew install kitdot/tap/nsh
```

## Uninstall

```bash
brew uninstall nsh
brew untap kitdot/tap
```

## Build from source

Requires Go 1.25+.

```bash
git clone https://github.com/kitdot/nsh.git
cd nsh
CGO_ENABLED=1 go build -ldflags "-X github.com/kitdot/nsh/cmd.nshVersion=0.1.0" -o nsh .
sudo install -m 0755 nsh /usr/local/bin/nsh
```

If version injection is omitted, `nsh` shows `dev`. To use git tags:

```bash
CGO_ENABLED=1 go build -ldflags "-X github.com/kitdot/nsh/cmd.nshVersion=$(git describe --tags --always)" -o nsh .
```

### macOS universal binary (optional)

```bash
CGO_ENABLED=1 GOARCH=arm64 go build -ldflags "-X github.com/kitdot/nsh/cmd.nshVersion=0.1.0" -o /tmp/nsh-arm64 .
CGO_ENABLED=1 GOARCH=amd64 CC="clang -arch x86_64" go build -ldflags "-X github.com/kitdot/nsh/cmd.nshVersion=0.1.0" -o /tmp/nsh-amd64 .
lipo -create /tmp/nsh-arm64 /tmp/nsh-amd64 -output nsh
```

## Quick Start

```bash
nsh                 # open the main TUI (groups + pinned)
nsh c web1          # connect directly
nsh n               # create host
nsh e web1          # edit host
nsh d web1          # delete host
nsh p               # pinned-only view
nsh exp             # export
nsh imp backup.nsh.enc
nsh -v              # version
```

If no alias is given, `conn`, `copy`, `edit`, `del`, `auth`, and `show` open an interactive selector.

## Commands

| Command | Alias | Description |
|---|---|---|
| `conn [alias]` | `c` | Connect via SSH |
| `pin` | `p` | Pinned-host view |
| `new` | `n` | Create host |
| `copy [alias]` | `cp` | Copy host as a new entry |
| `edit [alias]` | `e` | Edit host |
| `del [alias\|group]` | `d` | Delete host or group |
| `order [scope]` | `o` | Reorder groups, hosts in group, or pinned hosts (`scope`: `group` / `host` / `pinned`) |
| `auth [alias]` | `au` | Change auth mode |
| `export` | `exp` | Export hosts |
| `import [file]` | `imp` | Import hosts |
| `list` | `l` | Alias of `nsh` (main TUI) |
| `show [alias]` | `s` | Show host details |
| `config [key] [value]` | `conf` | Read/write settings |
| `completion` | — | Interactive install/remove for shell completions |
| `help` | `h` | Help |

## Common Workflows

### 1) Create / copy / edit

- `nsh n`: step-by-step prompt for alias, HostName, User, Port, auth, group, description
- `nsh cp web1`: create a new host using an existing host as template
- `nsh e web1`: edit an existing host; alias changes also update Keychain password mapping

### 2) Manage authentication

`nsh au web1` supports:

- None (no extra auth handling)
- Password (stored in Keychain and auto-filled on connect)
- Private key (runs `ssh-add --apple-use-keychain` before connect)

### 3) Delete host / group

```bash
nsh d web1
nsh d web1 -y
nsh d Production --is-group
```

When deleting a group, you can choose:

- Remove only the group (hosts move to `Uncategorized`)
- Delete the group and all hosts (with confirmation)

`Host *` (global default) is protected and cannot be deleted.

### 4) Reorder

```bash
nsh o
nsh o group
nsh o host Dev
nsh o pinned
```

Interactive reorder keys: `Space` pick/drop, `↑↓` move, `Enter` save, `Esc` cancel.

### 5) Export / import

`nsh exp`:

- Basic: `.nsh.json` (no passwords or keys)
- Full: `.nsh.enc` (includes passwords + keys, encrypted with AES-256-GCM)
- With multiple groups, export all groups or selected groups
- Full export requires an encryption password and Touch ID

`nsh imp [file]`:

- Supports `.nsh.json` and `.nsh.enc`
- `.nsh.enc` requires Touch ID and the export password
- If plain JSON includes secrets, Touch ID is still required
- Conflict strategies: ask each / skip all / overwrite all / rename all
- Passwords are written to Keychain; keys are written to `~/.ssh/nsh/` (`0600`) and `IdentityFile` is updated

## Main UI Keybindings

### Groups View (`nsh` / `nsh l`)

| Key | Action |
|---|---|
| `Enter` | Connect or expand group |
| `e` | Edit selected host |
| `d` | Delete selected host |
| `n` | Create host |
| `p` | Pin / unpin |
| `Tab` | Switch to pinned view |
| `/` | Search (space means AND) |
| `Esc` | Clear search / collapse / quit |
| `↑↓` or `jk` | Navigate |

### Pinned View (`nsh p` or `Tab` from main view)

| Key | Action |
|---|---|
| `Enter` | Connect |
| `e` | Edit |
| `d` | Delete |
| `p` | Unpin |
| `Space` | Enter reorder mode |
| `Tab` | Back to groups view |
| `/` | Search |
| `Esc` | Clear search / quit |

## Completion

```bash
nsh completion
```

Interactive install/remove supports `zsh`, `bash`, and `fish` (PowerShell script output is also available).

Manual output:

```bash
nsh completion zsh > ~/.zsh/completions/_nsh
nsh completion bash > ~/.local/share/bash-completion/completions/nsh
nsh completion fish > ~/.config/fish/completions/nsh.fish
```

## Settings (`nsh config`)

Supported keys:

- `mode`: `auto` (default) / `fzf` / `list`

```bash
nsh conf
nsh conf mode
nsh conf mode fzf
```

## Global Flags

```bash
nsh -v, --version
nsh --ssh-config <path>
```

`--ssh-config` changes the main SSH config path. `nsh` then uses `<dir>/nsh/config` as its managed config file.

## Tag Protocol

`nsh` stores metadata tags in `~/.ssh/nsh/config` using `# nsh:`:

```ssh
# nsh: group=Production, desc=Main web server, auth=password, order=1
Host web1
    HostName 192.168.1.1
    User deploy
    Port 22
    IdentityFile ~/.ssh/prod_key
```

- `group`: group name
- `desc`: description
- `auth`: `password` or `key`
- `order`: order within group

Additional metadata tags:

- `# nsh-groups:` group display order
- `# nsh-pinned:` pinned display order

## Security and Data Integrity

- Lossless parser: preserves original formatting, comments, `Include`, and `Match`
- Atomic write: tmp -> rename
- Backup rotation: creates `.nsh.bak` before write
- Permission control: config and key files are kept at `0600`
- Keychain: passwords are not written into SSH config
- Touch ID: required for sensitive import/export

## Project Structure (summary)

```text
nsh-go/
├── main.go
├── cmd/       # Cobra commands
├── bridge/    # Bubble Tea TUI components
├── core/      # parser, config manager, keychain, crypto
└── connect/   # ssh execution / PTY integration
```

## License

MIT

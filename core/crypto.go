package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// File format magic bytes
var encMagicV1 = []byte{'N', 'S', 'H', 0x01}
var encMagicV2 = []byte{'N', 'S', 'H', 0x02}

const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32 // AES-256
	saltLen      = 16

	// Safe upper bounds for KDF parameters (prevent DoS via malicious files)
	maxKdfTime    = 10
	maxKdfMemory  = 256 * 1024 // 256 MB
	maxKdfThreads = 8
)

// EncryptExport encrypts JSON data with AES-256-GCM using Argon2id key derivation
func EncryptExport(plaintext []byte, password string) ([]byte, error) {
	// Generate random salt
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key with Argon2id
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Create AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Build binary output: magic(v2) + argon_time(4) + argon_memory(4) + argon_threads(1) + salt_len(4) + salt + nonce_len(4) + nonce + ciphertext
	var out []byte
	out = append(out, encMagicV2...)

	// KDF parameters
	timeBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(timeBuf, argonTime)
	out = append(out, timeBuf...)

	memBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(memBuf, argonMemory)
	out = append(out, memBuf...)

	out = append(out, argonThreads)

	saltLenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(saltLenBuf, uint32(len(salt)))
	out = append(out, saltLenBuf...)
	out = append(out, salt...)

	nonceLenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(nonceLenBuf, uint32(len(nonce)))
	out = append(out, nonceLenBuf...)
	out = append(out, nonce...)

	out = append(out, ciphertext...)

	return out, nil
}

// DecryptExport decrypts an encrypted export file (supports v1 and v2)
func DecryptExport(data []byte, password string) ([]byte, error) {
	if len(data) < 4 {
		return nil, errors.New("file too small")
	}

	// Check magic prefix
	if data[0] != 'N' || data[1] != 'S' || data[2] != 'H' {
		return nil, errors.New("invalid file format (bad magic bytes)")
	}

	version := data[3]
	offset := 4

	// Read KDF parameters
	var kdfTime, kdfMemory uint32
	var kdfThreads uint8

	switch version {
	case 0x01:
		// v1: hardcoded params
		kdfTime = argonTime
		kdfMemory = argonMemory
		kdfThreads = argonThreads
	case 0x02:
		// v2: params stored in header
		if offset+9 > len(data) {
			return nil, errors.New("invalid file format (truncated KDF params)")
		}
		kdfTime = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
		kdfMemory = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
		kdfThreads = data[offset]
		offset++

		// Validate KDF parameters to prevent resource exhaustion
		if kdfTime == 0 || kdfTime > maxKdfTime {
			return nil, fmt.Errorf("KDF time parameter out of safe range: %d (max %d)", kdfTime, maxKdfTime)
		}
		if kdfMemory == 0 || kdfMemory > maxKdfMemory {
			return nil, fmt.Errorf("KDF memory parameter out of safe range: %d KB (max %d KB)", kdfMemory, maxKdfMemory)
		}
		if kdfThreads == 0 || kdfThreads > maxKdfThreads {
			return nil, fmt.Errorf("KDF threads parameter out of safe range: %d (max %d)", kdfThreads, maxKdfThreads)
		}
	default:
		return nil, fmt.Errorf("unsupported format version: %d", version)
	}

	// Read salt
	if offset+4 > len(data) {
		return nil, errors.New("invalid file format (truncated salt length)")
	}
	sLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
	offset += 4

	if offset+sLen > len(data) {
		return nil, errors.New("invalid file format (truncated salt)")
	}
	salt := data[offset : offset+sLen]
	offset += sLen

	// Read nonce
	if offset+4 > len(data) {
		return nil, errors.New("invalid file format (truncated nonce length)")
	}
	nLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
	offset += 4

	if offset+nLen > len(data) {
		return nil, errors.New("invalid file format (truncated nonce)")
	}
	nonce := data[offset : offset+nLen]
	offset += nLen

	// Ciphertext
	ciphertext := data[offset:]
	if len(ciphertext) == 0 {
		return nil, errors.New("invalid file format (no ciphertext)")
	}

	// Derive key
	key := argon2.IDKey([]byte(password), salt, kdfTime, kdfMemory, kdfThreads, argonKeyLen)

	// Decrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed (wrong password or corrupted file)")
	}

	return plaintext, nil
}

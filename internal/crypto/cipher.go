package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// Encrypt encrypts plaintext with AES-256-GCM using a randomly generated nonce.
// Output layout: nonce (12 bytes) || ciphertext+tag.
// The key must be exactly 32 bytes.
func Encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: generate nonce: %w", err)
	}
	// Seal appends ciphertext+tag to nonce, yielding nonce||ciphertext+tag.
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts an AES-256-GCM ciphertext produced by Encrypt.
// It returns an error if the key is wrong or the ciphertext has been tampered with.
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new GCM: %w", err)
	}
	ns := gcm.NonceSize()
	if len(ciphertext) < ns {
		return nil, fmt.Errorf("crypto: ciphertext too short (got %d bytes, need at least %d)", len(ciphertext), ns)
	}
	nonce, ct := ciphertext[:ns], ciphertext[ns:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decryption failed: %w", err)
	}
	return plaintext, nil
}

// zeroBytes overwrites b with zeros to limit the lifetime of sensitive data in
// memory. This is best-effort: the Go runtime may have already copied the data
// to other heap locations during GC.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

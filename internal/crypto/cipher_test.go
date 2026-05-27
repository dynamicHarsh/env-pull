package crypto_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/harsh-sonkar/env-pull/internal/crypto"
)

func randomKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	return key
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := randomKey(t)

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"empty plaintext", []byte{}},
		{"simple ASCII", []byte("hello, world")},
		{"env file format", []byte("DB_PASSWORD=s3cr3t\nAPI_KEY=abc123\n")},
		{"binary data", []byte{0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF}},
		{"unicode content", []byte("password=pässwörd🔑")},
		{"large payload 64 KiB", bytes.Repeat([]byte("A"), 64*1024)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := crypto.Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}
			// Ciphertext must differ from plaintext (for non-empty inputs).
			if len(tt.plaintext) > 0 && bytes.Equal(encrypted, tt.plaintext) {
				t.Error("Encrypt() returned plaintext unchanged")
			}

			decrypted, err := crypto.Decrypt(encrypted, key)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypt() = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := randomKey(t)
	key2 := randomKey(t)

	encrypted, err := crypto.Encrypt([]byte("secret value"), key1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := crypto.Decrypt(encrypted, key2); err == nil {
		t.Error("Decrypt() with wrong key should return an error")
	}
}

func TestEncryptProducesUniqueCiphertexts(t *testing.T) {
	key := randomKey(t)
	plaintext := []byte("same plaintext encrypted twice")

	enc1, err := crypto.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() first call: %v", err)
	}
	enc2, err := crypto.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() second call: %v", err)
	}
	if bytes.Equal(enc1, enc2) {
		t.Error("Encrypt() produced identical ciphertexts: nonce is not random")
	}
}

func TestDecryptTruncatedCiphertext(t *testing.T) {
	key := randomKey(t)
	// 2 bytes is shorter than the 12-byte GCM nonce minimum.
	if _, err := crypto.Decrypt([]byte{0x01, 0x02}, key); err == nil {
		t.Error("Decrypt() with truncated ciphertext should return an error")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := randomKey(t)
	encrypted, err := crypto.Encrypt([]byte("authentic message"), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Flip bits in the ciphertext body (bytes after the 12-byte nonce).
	tampered := make([]byte, len(encrypted))
	copy(tampered, encrypted)
	tampered[12] ^= 0xFF

	if _, err := crypto.Decrypt(tampered, key); err == nil {
		t.Error("Decrypt() with tampered ciphertext should return an error (GCM tag check failed)")
	}
}

// Package crypto handles encryption and decryption of the local env-edit
// vault. It manages AES-256-GCM encrypted secret stores, derives keys via
// scrypt, and aggressively zeroes all plaintext key material from memory
// immediately after use — no decrypted bytes are ever written to disk or
// passed to a logger.
package crypto

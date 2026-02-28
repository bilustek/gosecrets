// Package krypto provides AES-256-GCM encryption and decryption
// for gosecrets credential files.
package krypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

const keySize = 32 // AES-256

// GenerateKey creates a new random 32-byte key and returns it as a hex string.
func GenerateKey() (string, error) {
	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("gosecrets: failed to generate key: %w", err)
	}
	return hex.EncodeToString(key), nil
}

// Encrypt encrypts plaintext using AES-256-GCM with the given hex-encoded key.
// Returns hex-encoded ciphertext (nonce prepended).
func Encrypt(plaintext []byte, hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("gosecrets: invalid key format: %w", err)
	}
	if len(key) != keySize {
		return nil, fmt.Errorf("gosecrets: key must be %d bytes, got %d", keySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("gosecrets: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gosecrets: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("gosecrets: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	encoded := make([]byte, hex.EncodedLen(len(ciphertext)))
	hex.Encode(encoded, ciphertext)
	return encoded, nil
}

// Decrypt decrypts hex-encoded ciphertext using AES-256-GCM with the given hex-encoded key.
func Decrypt(hexCiphertext []byte, hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("gosecrets: invalid key format: %w", err)
	}
	if len(key) != keySize {
		return nil, fmt.Errorf("gosecrets: key must be %d bytes, got %d", keySize, len(key))
	}

	ciphertext := make([]byte, hex.DecodedLen(len(hexCiphertext)))
	if _, err = hex.Decode(ciphertext, hexCiphertext); err != nil {
		return nil, fmt.Errorf("gosecrets: invalid ciphertext format: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("gosecrets: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gosecrets: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("gosecrets: ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("gosecrets: decryption failed (wrong key?): %w", err)
	}

	return plaintext, nil
}

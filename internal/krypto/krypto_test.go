package krypto_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/bilustek/gosecrets/internal/krypto"
)

func TestGenerateKeyReturnsValidHexString(t *testing.T) {
	t.Parallel()

	key, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	// Hex-encoded 32 bytes = 64 chars
	if len(key) != 64 {
		t.Fatalf("expected key length 64, got %d", len(key))
	}

	// must be valid hex
	if _, err = hex.DecodeString(key); err != nil {
		t.Fatalf("key is not valid hex: %v", err)
	}
}

func TestGenerateKeyProducesUniqueKeys(t *testing.T) {
	t.Parallel()

	key1, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	key2, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	if key1 == key2 {
		t.Fatal("two generated keys should not be identical")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	key, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("database:\n  password: supersecret\napi_key: sk-123")

	ciphertext, err := krypto.Encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	if string(ciphertext) == string(plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := krypto.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptDecryptEmptyPlaintext(t *testing.T) {
	t.Parallel()

	key, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	ciphertext, err := krypto.Encrypt([]byte(""), key)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := krypto.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != "" {
		t.Fatalf("expected empty string, got %q", decrypted)
	}
}

func TestEncryptDecryptLargePlaintext(t *testing.T) {
	t.Parallel()

	key, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte(strings.Repeat("A", 1<<16)) // 64KB

	ciphertext, err := krypto.Encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := krypto.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatal("large plaintext round-trip failed")
	}
}

func TestEncryptProducesDifferentCiphertextEachTime(t *testing.T) {
	t.Parallel()

	key, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("same content")

	c1, err := krypto.Encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	c2, err := krypto.Encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	if string(c1) == string(c2) {
		t.Fatal("encrypting same content twice should produce different ciphertext")
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	t.Parallel()

	key1, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	key2, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	ciphertext, err := krypto.Encrypt([]byte("secret"), key1)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = krypto.Decrypt(ciphertext, key2); err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	} else if !strings.Contains(err.Error(), "wrong key") {
		t.Fatalf("expected 'wrong key' in error message, got: %v", err)
	}
}

func TestEncryptRejectsInvalidHexKey(t *testing.T) {
	t.Parallel()

	if _, err := krypto.Encrypt([]byte("test"), "not-hex!"); err == nil {
		t.Fatal("expected error for non-hex key")
	} else if !strings.Contains(err.Error(), "invalid key format") {
		t.Fatalf("expected 'invalid key format' in error, got: %v", err)
	}
}

func TestEncryptRejectsWrongSizeKey(t *testing.T) {
	t.Parallel()

	// valid hex but only 16 bytes (32 hex chars) instead of 32 bytes (64 hex chars)
	shortKey := strings.Repeat("ab", 16)

	if _, err := krypto.Encrypt([]byte("test"), shortKey); err == nil {
		t.Fatal("expected error for wrong-size key")
	} else if !strings.Contains(err.Error(), "key must be 32 bytes") {
		t.Fatalf("expected 'key must be 32 bytes' in error, got: %v", err)
	}
}

func TestDecryptRejectsInvalidHexKey(t *testing.T) {
	t.Parallel()

	if _, err := krypto.Decrypt([]byte("aabbccdd"), "not-hex!"); err == nil {
		t.Fatal("expected error for non-hex key")
	} else if !strings.Contains(err.Error(), "invalid key format") {
		t.Fatalf("expected 'invalid key format' in error, got: %v", err)
	}
}

func TestDecryptRejectsWrongSizeKey(t *testing.T) {
	t.Parallel()

	shortKey := strings.Repeat("ab", 16)

	if _, err := krypto.Decrypt([]byte("aabbccdd"), shortKey); err == nil {
		t.Fatal("expected error for wrong-size key")
	} else if !strings.Contains(err.Error(), "key must be 32 bytes") {
		t.Fatalf("expected 'key must be 32 bytes' in error, got: %v", err)
	}
}

func TestDecryptRejectsInvalidHexCiphertext(t *testing.T) {
	t.Parallel()

	key, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	if _, err = krypto.Decrypt([]byte("zzzz-not-hex"), key); err == nil {
		t.Fatal("expected error for invalid hex ciphertext")
	} else if !strings.Contains(err.Error(), "invalid ciphertext format") {
		t.Fatalf("expected 'invalid ciphertext format' in error, got: %v", err)
	}
}

func TestDecryptRejectsTooShortCiphertext(t *testing.T) {
	t.Parallel()

	key, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	// valid hex but too short to contain nonce + ciphertext
	shortCiphertext := hex.EncodeToString([]byte("tiny"))

	if _, err = krypto.Decrypt([]byte(shortCiphertext), key); err == nil {
		t.Fatal("expected error for too-short ciphertext")
	} else if !strings.Contains(err.Error(), "ciphertext too short") {
		t.Fatalf("expected 'ciphertext too short' in error, got: %v", err)
	}
}

func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	t.Parallel()

	key, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	ciphertext, err := krypto.Encrypt([]byte("sensitive data"), key)
	if err != nil {
		t.Fatal(err)
	}

	// flip a byte in the middle of the ciphertext
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	mid := len(tampered) / 2
	if tampered[mid] == '0' {
		tampered[mid] = '1'
	} else {
		tampered[mid] = '0'
	}

	if _, err = krypto.Decrypt(tampered, key); err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

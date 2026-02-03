// Package store provides tests for token encryption/decryption.
package store

import (
	"strings"
	"testing"
)

// valid32ByteKey is a valid 32-byte key for testing.
const valid32ByteKey = "0123456789abcdefghijklmnopqrstuv"

// TestEncryptDecrypt tests that encryption and decryption are reversible.
func TestEncryptDecrypt(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "simple text",
			input: "hello world",
		},
		{
			name:  "bot token",
			input: "1234567890:ABCDefGHIjklMNOpqrsTUVwxyz",
		},
		{
			name:  "dingtalk app secret",
			input: "SEC" + "retValue1234567890123456789012345678901234", // 40+ chars
		},
		{
			name:  "special characters",
			input: "test@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
		{
			name:  "unicode",
			input: "测试中文🎉🔥",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := EncryptToken(tc.input, valid32ByteKey)
			if err != nil {
				t.Fatalf("EncryptToken failed: %v", err)
			}

			// Encrypted should be different from original
			if encrypted == tc.input {
				t.Error("encrypted text should differ from plaintext")
			}

			// Decrypt
			decrypted, err := DecryptToken(encrypted, valid32ByteKey)
			if err != nil {
				t.Fatalf("DecryptToken failed: %v", err)
			}

			// Decrypted should match original
			if decrypted != tc.input {
				t.Errorf("decrypted text mismatch: got %q, want %q", decrypted, tc.input)
			}
		})
	}
}

// TestEncryptWithDifferentKeys tests that different keys produce different ciphertext.
func TestEncryptWithDifferentKeys(t *testing.T) {
	plaintext := "sensitive_token_123"
	key1 := "0123456789abcdefghijklmnopqrstuv"
	key2 := "fedcba0987654321zyxwvutsrqponmlk"

	encrypted1, err1 := EncryptToken(plaintext, key1)
	if err1 != nil {
		t.Fatalf("first encryption failed: %v", err1)
	}

	encrypted2, err2 := EncryptToken(plaintext, key2)
	if err2 != nil {
		t.Fatalf("second encryption failed: %v", err2)
	}

	// Different keys should produce different ciphertext (due to random nonce)
	if encrypted1 == encrypted2 {
		t.Error("different keys should produce different ciphertext")
	}
}

// TestDecryptWithWrongKey tests that decryption fails with wrong key.
func TestDecryptWithWrongKey(t *testing.T) {
	plaintext := "secret_data"
	correctKey := "0123456789abcdefghijklmnopqrstuv"
	wrongKey := "fedcba0987654321zyxwvutsrqponmlk"

	encrypted, err := EncryptToken(plaintext, correctKey)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Try to decrypt with wrong key
	_, err = DecryptToken(encrypted, wrongKey)
	if err == nil {
		t.Error("expected decryption to fail with wrong key, but it succeeded")
	}
}

// TestDecryptInvalidBase64 tests that invalid base64 returns error.
func TestDecryptInvalidBase64(t *testing.T) {
	invalidCiphertext := "not-valid-base64!!!"

	_, err := DecryptToken(invalidCiphertext, valid32ByteKey)
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}

// TestDecryptMalformedCiphertext tests that malformed ciphertext returns error.
func TestDecryptMalformedCiphertext(t *testing.T) {
	// Valid base64 but not a valid ciphertext
	malformed := "dGVzdA==" // "test" in base64, but not encrypted format

	_, err := DecryptToken(malformed, valid32ByteKey)
	if err == nil {
		t.Error("expected error for malformed ciphertext, got nil")
	}
}

// TestEmptyKey tests that empty key returns error.
func TestEmptyKey(t *testing.T) {
	plaintext := "test"
	emptyKey := ""

	_, err := EncryptToken(plaintext, emptyKey)
	if err == nil {
		t.Error("expected error for empty key, got nil")
	}
}

// TestEncryptEmptyString tests encrypting empty string.
func TestEncryptEmptyString(t *testing.T) {
	encrypted, err := EncryptToken("", valid32ByteKey)
	if err != nil {
		t.Fatalf("encryption of empty string failed: %v", err)
	}

	decrypted, err := DecryptToken(encrypted, valid32ByteKey)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != "" {
		t.Errorf("expected empty string, got %q", decrypted)
	}
}

// TestEncryptLongString tests encrypting long strings.
func TestEncryptLongString(t *testing.T) {
	longString := strings.Repeat("A", 10000) // 10KB

	encrypted, err := EncryptToken(longString, valid32ByteKey)
	if err != nil {
		t.Fatalf("encryption of long string failed: %v", err)
	}

	decrypted, err := DecryptToken(encrypted, valid32ByteKey)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != longString {
		t.Errorf("long string round-trip failed: got len=%d, want len=%d", len(decrypted), len(longString))
	}
}

// TestCiphertextFormat tests that ciphertext has expected format.
func TestCiphertextFormat(t *testing.T) {
	plaintext := "test"

	encrypted, err := EncryptToken(plaintext, valid32ByteKey)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Ciphertext should be base64-encoded (no non-base64 chars except padding)
	validBase64 := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	for _, c := range encrypted {
		if !strings.ContainsRune(validBase64, c) {
			t.Errorf("ciphertext contains invalid character: %c", c)
		}
	}

	// Ciphertext should be longer than plaintext (nonce + ciphertext + auth tag)
	if len(encrypted) <= len(plaintext) {
		t.Error("ciphertext should be longer than plaintext")
	}
}

// BenchmarkEncryptToken benchmarks the encryption operation.
func BenchmarkEncryptToken(b *testing.B) {
	plaintext := "this is a test message that needs to be encrypted securely"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncryptToken(plaintext, valid32ByteKey)
	}
}

// BenchmarkDecryptToken benchmarks the decryption operation.
func BenchmarkDecryptToken(b *testing.B) {
	plaintext := "this is a test message that needs to be encrypted securely"
	encrypted, _ := EncryptToken(plaintext, valid32ByteKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecryptToken(encrypted, valid32ByteKey)
	}
}

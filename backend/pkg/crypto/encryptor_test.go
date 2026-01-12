package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestEncryptor_EnvelopeEncryption(t *testing.T) {
	// Create encryptor with test key
	testKey, _ := hex.DecodeString("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	enc, err := NewEncryptorWithKey(testKey)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	// Test data
	plaintext := []byte(`{"api_key": "sk-test-12345", "secret": "my-secret-value"}`)

	// Encrypt
	encrypted, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Verify encrypted data is different from plaintext
	if bytes.Equal(encrypted.Ciphertext, plaintext) {
		t.Error("ciphertext should not equal plaintext")
	}

	// Decrypt
	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	// Verify decrypted data matches original
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted data does not match original: got %s, want %s", decrypted, plaintext)
	}
}

func TestEncryptor_DifferentEncryptionsProduceDifferentResults(t *testing.T) {
	testKey, _ := hex.DecodeString("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	enc, err := NewEncryptorWithKey(testKey)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plaintext := []byte("test data")

	// Encrypt twice
	encrypted1, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("first encryption failed: %v", err)
	}

	encrypted2, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("second encryption failed: %v", err)
	}

	// Ciphertexts should be different (due to random nonce)
	if bytes.Equal(encrypted1.Ciphertext, encrypted2.Ciphertext) {
		t.Error("encrypting same data twice should produce different ciphertexts")
	}

	// But both should decrypt to same plaintext
	decrypted1, _ := enc.Decrypt(encrypted1)
	decrypted2, _ := enc.Decrypt(encrypted2)

	if !bytes.Equal(decrypted1, decrypted2) {
		t.Error("decrypted data should match for both encryptions")
	}
}

func TestEncryptor_WrongKeyFails(t *testing.T) {
	key1, _ := hex.DecodeString("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	key2, _ := hex.DecodeString("fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210")

	enc1, _ := NewEncryptorWithKey(key1)
	enc2, _ := NewEncryptorWithKey(key2)

	plaintext := []byte("secret data")

	// Encrypt with key1
	encrypted, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Try to decrypt with key2 (should fail)
	_, err = enc2.Decrypt(encrypted)
	if err == nil {
		t.Error("decryption with wrong key should fail")
	}
}

func TestEncryptor_InvalidKeySize(t *testing.T) {
	// Too short
	shortKey := []byte("tooshort")
	_, err := NewEncryptorWithKey(shortKey)
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey for short key, got %v", err)
	}

	// Too long
	longKey := make([]byte, 64)
	_, err = NewEncryptorWithKey(longKey)
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey for long key, got %v", err)
	}
}

func TestEncryptor_EmptyData(t *testing.T) {
	testKey, _ := hex.DecodeString("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	enc, _ := NewEncryptorWithKey(testKey)

	// Empty data should still work
	encrypted, err := enc.Encrypt([]byte{})
	if err != nil {
		t.Fatalf("encrypting empty data failed: %v", err)
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypting empty data failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Error("decrypted empty data should be empty")
	}
}

func TestEncryptor_LargeData(t *testing.T) {
	testKey, _ := hex.DecodeString("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	enc, _ := NewEncryptorWithKey(testKey)

	// Create large data (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	encrypted, err := enc.Encrypt(largeData)
	if err != nil {
		t.Fatalf("encrypting large data failed: %v", err)
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypting large data failed: %v", err)
	}

	if !bytes.Equal(decrypted, largeData) {
		t.Error("decrypted large data does not match original")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	testKey, _ := hex.DecodeString("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	enc, _ := NewEncryptorWithKey(testKey)
	data := []byte(`{"api_key": "sk-test-12345", "secret": "my-secret-value"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Encrypt(data)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	testKey, _ := hex.DecodeString("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	enc, _ := NewEncryptorWithKey(testKey)
	data := []byte(`{"api_key": "sk-test-12345", "secret": "my-secret-value"}`)
	encrypted, _ := enc.Encrypt(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Decrypt(encrypted)
	}
}

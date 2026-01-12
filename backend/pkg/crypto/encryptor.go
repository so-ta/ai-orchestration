package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	// KeySize is the size of AES-256 key in bytes
	KeySize = 32

	// NonceSize is the size of GCM nonce in bytes
	NonceSize = 12
)

var (
	// ErrInvalidKey is returned when the key is invalid
	ErrInvalidKey = errors.New("invalid encryption key: must be 32 bytes (64 hex characters)")

	// ErrInvalidNonce is returned when the nonce is invalid
	ErrInvalidNonce = errors.New("invalid nonce: must be 12 bytes")

	// ErrDecryptionFailed is returned when decryption fails
	ErrDecryptionFailed = errors.New("decryption failed: invalid ciphertext or key")

	// ErrNoMasterKey is returned when master key is not set
	ErrNoMasterKey = errors.New("master encryption key not configured")
)

// Encryptor provides AES-256-GCM encryption/decryption with envelope encryption
type Encryptor struct {
	masterKey []byte // Key Encryption Key (KEK)
}

// NewEncryptor creates a new encryptor with the master key from environment
func NewEncryptor() (*Encryptor, error) {
	keyHex := os.Getenv("ENCRYPTION_KEY")
	if keyHex == "" {
		// For development, use a default key (NOT for production!)
		keyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key hex: %w", err)
	}

	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	return &Encryptor{masterKey: key}, nil
}

// NewEncryptorWithKey creates a new encryptor with a specific key
func NewEncryptorWithKey(key []byte) (*Encryptor, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}
	return &Encryptor{masterKey: key}, nil
}

// GenerateDEK generates a new Data Encryption Key
func (e *Encryptor) GenerateDEK() ([]byte, error) {
	dek := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}
	return dek, nil
}

// GenerateNonce generates a new nonce for AES-GCM
func (e *Encryptor) GenerateNonce() ([]byte, error) {
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}

// EncryptDEK encrypts the Data Encryption Key with the master key
func (e *Encryptor) EncryptDEK(dek []byte) (encryptedDEK, nonce []byte, err error) {
	if e.masterKey == nil {
		return nil, nil, ErrNoMasterKey
	}

	block, err := aes.NewCipher(e.masterKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err = e.GenerateNonce()
	if err != nil {
		return nil, nil, err
	}

	encryptedDEK = gcm.Seal(nil, nonce, dek, nil)
	return encryptedDEK, nonce, nil
}

// DecryptDEK decrypts the Data Encryption Key with the master key
func (e *Encryptor) DecryptDEK(encryptedDEK, nonce []byte) ([]byte, error) {
	if e.masterKey == nil {
		return nil, ErrNoMasterKey
	}

	if len(nonce) != NonceSize {
		return nil, ErrInvalidNonce
	}

	block, err := aes.NewCipher(e.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	dek, err := gcm.Open(nil, nonce, encryptedDEK, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return dek, nil
}

// EncryptData encrypts data with a DEK
func (e *Encryptor) EncryptData(data, dek []byte) (ciphertext, nonce []byte, err error) {
	if len(dek) != KeySize {
		return nil, nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err = e.GenerateNonce()
	if err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, data, nil)
	return ciphertext, nonce, nil
}

// DecryptData decrypts data with a DEK
func (e *Encryptor) DecryptData(ciphertext, dek, nonce []byte) ([]byte, error) {
	if len(dek) != KeySize {
		return nil, ErrInvalidKey
	}

	if len(nonce) != NonceSize {
		return nil, ErrInvalidNonce
	}

	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptedData holds the result of envelope encryption
type EncryptedData struct {
	Ciphertext   []byte `json:"ciphertext"`
	EncryptedDEK []byte `json:"encrypted_dek"`
	DataNonce    []byte `json:"data_nonce"`
	DEKNonce     []byte `json:"dek_nonce"`
}

// Encrypt performs envelope encryption: generates DEK, encrypts data with DEK, encrypts DEK with master key
func (e *Encryptor) Encrypt(data []byte) (*EncryptedData, error) {
	// Generate DEK
	dek, err := e.GenerateDEK()
	if err != nil {
		return nil, err
	}

	// Encrypt data with DEK
	ciphertext, dataNonce, err := e.EncryptData(data, dek)
	if err != nil {
		return nil, err
	}

	// Encrypt DEK with master key
	encryptedDEK, dekNonce, err := e.EncryptDEK(dek)
	if err != nil {
		return nil, err
	}

	// Clear DEK from memory
	for i := range dek {
		dek[i] = 0
	}

	return &EncryptedData{
		Ciphertext:   ciphertext,
		EncryptedDEK: encryptedDEK,
		DataNonce:    dataNonce,
		DEKNonce:     dekNonce,
	}, nil
}

// Decrypt performs envelope decryption: decrypts DEK with master key, decrypts data with DEK
func (e *Encryptor) Decrypt(ed *EncryptedData) ([]byte, error) {
	// Decrypt DEK with master key
	dek, err := e.DecryptDEK(ed.EncryptedDEK, ed.DEKNonce)
	if err != nil {
		return nil, err
	}

	// Decrypt data with DEK
	plaintext, err := e.DecryptData(ed.Ciphertext, dek, ed.DataNonce)

	// Clear DEK from memory
	for i := range dek {
		dek[i] = 0
	}

	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

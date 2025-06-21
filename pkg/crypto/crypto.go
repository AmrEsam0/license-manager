package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// CryptoManager handles all cryptographic operations with security-focused key derivation.
// All cryptographic keys are derived from a single master key using PBKDF2 with unique salts.
// This approach provides:
// - No hardcoded secrets in the codebase
// - Deterministic key generation (same master key = same derived keys)
// - Strong isolation between different key types
// - Resistance to rainbow table attacks via unique salts
type CryptoManager struct {
	masterKey []byte
}

// NewCryptoManager creates a new crypto manager with a master key
func NewCryptoManager() (*CryptoManager, error) {
	key, err := getMasterKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get master key: %v", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes")
	}
	return &CryptoManager{masterKey: key}, nil
}

// NewCryptoManagerWithKey creates a new crypto manager with a provided master key
func NewCryptoManagerWithKey(masterKey string) (*CryptoManager, error) {
	if masterKey == "" {
		return nil, fmt.Errorf("master key cannot be empty")
	}

	// Ensure key is exactly 32 bytes by hashing it
	hash := sha256.Sum256([]byte(masterKey))
	key := hash[:]

	return &CryptoManager{masterKey: key}, nil
}

// getMasterKey retrieves the master key from environment variable
func getMasterKey() ([]byte, error) {
	// Get key from environment variable (required)
	envKey := os.Getenv("LICENSE_MASTER_KEY")
	if envKey == "" {
		return nil, fmt.Errorf("LICENSE_MASTER_KEY environment variable is required")
	}

	// Ensure key is exactly 32 bytes by hashing it
	hash := sha256.Sum256([]byte(envKey))
	return hash[:], nil
}

// DeriveSerialKey derives a key for serial generation from the master key.
// Uses PBKDF2 with 10,000 iterations for computational cost against brute force attacks.
// The salt is deterministically derived from the master key to ensure consistency.
func (cm *CryptoManager) DeriveSerialKey() []byte {
	// Use PBKDF2 with a unique salt for serial key derivation
	salt := cm.deriveSalt("SERIAL_KEY_DERIVATION")
	return pbkdf2.Key(cm.masterKey, salt, 10000, 32, sha256.New)
}

// DeriveEncryptionKey derives a key for encryption from the master key.
// Uses PBKDF2 with 10,000 iterations and a unique salt for AES-GCM encryption.
// This ensures the encryption key is completely different from the serial key.
func (cm *CryptoManager) DeriveEncryptionKey() []byte {
	// Use PBKDF2 with a unique salt for encryption key derivation
	salt := cm.deriveSalt("ENCRYPTION_KEY_DERIVATION")
	return pbkdf2.Key(cm.masterKey, salt, 10000, 32, sha256.New)
}

// GenerateSerial creates a serial number using the same logic as before
// but with derived key instead of hardcoded secret
func (cm *CryptoManager) GenerateSerial(pcId, productName string, maxDays int) string {
	serialKey := cm.DeriveSerialKey()
	serialKeyHex := hex.EncodeToString(serialKey)

	// Keep the exact same logic as before to maintain compatibility
	data := fmt.Sprintf("%s|%s|%d|%s", pcId, productName, maxDays, serialKeyHex)
	hash := md5.Sum([]byte(data))
	hashStr := hex.EncodeToString(hash[:])

	serial := ""
	for i, char := range hashStr {
		if i > 0 && i%5 == 0 {
			serial += "-"
		}
		serial += string(char)
	}
	return strings.ToUpper(serial[:23])
}

// Encrypt encrypts data using AES-GCM with derived encryption key
func (cm *CryptoManager) Encrypt(data []byte) ([]byte, error) {
	key := cm.DeriveEncryptionKey()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM with derived encryption key
func (cm *CryptoManager) Decrypt(data []byte) ([]byte, error) {
	key := cm.DeriveEncryptionKey()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// HashData creates a SHA256 hash of the input data
func (cm *CryptoManager) HashData(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// deriveSalt generates a deterministic salt from the master key and purpose.
// This provides several security benefits:
// - Each key type gets a unique salt (prevents key reuse)
// - Salt is derived from master key (no hardcoded values)
// - Deterministic generation (same master key = same salt = same derived keys)
// - Domain separation (different purposes produce different salts)
func (cm *CryptoManager) deriveSalt(purpose string) []byte {
	// Create a unique salt by hashing the master key with the purpose and a constant
	// This ensures the salt is unique per purpose but deterministic per master key
	data := append([]byte("LICENSE_MANAGER_2025_"), []byte(purpose)...)
	data = append(data, cm.masterKey...)
	hash := sha256.Sum256(data)
	return hash[:]
}

// GenerateRandomBytes generates cryptographically secure random bytes
func (cm *CryptoManager) GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

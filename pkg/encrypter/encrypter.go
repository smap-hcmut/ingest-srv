package encrypter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

func validateKey(key []byte) error {
	n := len(key)
	if n != AESKeyLen128 && n != AESKeyLen192 && n != AESKeyLen256 {
		return fmt.Errorf("%w: got %d bytes", ErrInvalidKeyLength, n)
	}
	return nil
}

func (e *implEncrypter) createByteKey() ([]byte, error) {
	key := []byte(e.key)
	if err := validateKey(key); err != nil {
		return nil, err
	}
	return key, nil
}

func (e *implEncrypter) getGCM() (cipher.AEAD, error) {
	key, err := e.createByteKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	return gcm, nil
}

func (e *implEncrypter) Encrypt(plaintext string) (string, error) {
	return e.EncryptBytesToString([]byte(plaintext))
}

func (e *implEncrypter) Decrypt(ciphertextStr string) (string, error) {
	plaintext, err := e.DecryptStringToBytes(ciphertextStr)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (e *implEncrypter) EncryptBytesToString(data []byte) (string, error) {
	gcm, err := e.getGCM()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *implEncrypter) DecryptStringToBytes(ciphertext string) ([]byte, error) {
	gcm, err := e.getGCM()
	if err != nil {
		return nil, err
	}
	ciphertextByte, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertextByte) < nonceSize {
		return nil, ErrCiphertextTooShort
	}
	nonce, ciphertextByte := ciphertextByte[:nonceSize], ciphertextByte[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextByte, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	return plaintext, nil
}

func (e *implEncrypter) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (e *implEncrypter) CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

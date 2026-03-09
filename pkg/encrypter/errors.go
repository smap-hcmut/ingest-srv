package encrypter

import "errors"

var (
	ErrInvalidKeyLength  = errors.New("encryption key must be 16, 24, or 32 bytes long")
	ErrCiphertextTooShort = errors.New("ciphertext is too short")
	ErrDecryptionFailed  = errors.New("decryption failed: invalid ciphertext or key")
)

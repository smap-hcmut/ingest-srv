package encrypter

// Encrypter provides encryption and password hashing.
// Implementations are safe for concurrent use.
type Encrypter interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
	EncryptBytesToString(data []byte) (string, error)
	DecryptStringToBytes(ciphertext string) ([]byte, error)
	HashPassword(password string) (string, error)
	CheckPasswordHash(password, hash string) bool
}

// New creates a new Encrypter with the provided key (16, 24, or 32 bytes for AES).
func New(key string) Encrypter {
	return &implEncrypter{key: key}
}

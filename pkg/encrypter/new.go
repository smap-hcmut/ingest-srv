package encrypter

type Encrypter interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
	EncryptBytesToString(data []byte) (string, error)
	DecryptStringToBytes(ciphertext string) ([]byte, error)

	// GenarateIntegrationKey genarate integration key
	GenarateIntegrationKey(shopID string, suffix string) (string, error)
}

type implEncrypter struct {
	key string
}

func NewEncrypter(key string) Encrypter {
	return &implEncrypter{
		key: key,
	}
}

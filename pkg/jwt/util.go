package jwt

import "fmt"

func validateConfig(cfg Config) error {
	if len(cfg.SecretKey) < MinSecretKeyLen {
		return fmt.Errorf("jwt: secret key must be at least %d characters long, got %d", MinSecretKeyLen, len(cfg.SecretKey))
	}
	return nil
}

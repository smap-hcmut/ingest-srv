package encrypter

import (
	"encoding/json"
	"time"
)

type integrationKey struct {
	ShopID string `json:"shop_id"`
	Suffix string `json:"prefix"`
	TimeSt string `json:"ts"`
}

func (m implEncrypter) GenarateIntegrationKey(shopID string, suffix string) (string, error) {
	data, err := json.Marshal(integrationKey{
		ShopID: shopID,
		Suffix: suffix,
		TimeSt: time.Now().String(),
	})
	if err != nil {
		return "", err
	}
	key, err := m.EncryptBytesToString(data)
	if err != nil {
		return "", err
	}

	return key, nil
}

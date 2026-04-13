package utils

import (
	"crypto/aes"
	"crypto/cipher"
)

const (
	TelegramAESKey = "m3X9pL2qR8tN4vW6cY1eF3hJ5kU7bA0z"
)

var TelegramFixedIV = []byte{0x42, 0x6f, 0x62, 0x20, 0x4c, 0x69, 0x6b, 0x65, 0x73, 0x20, 0x43, 0x61}

func EncryptTelegramPayload(plaintext string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(TelegramAESKey))
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nil, TelegramFixedIV, []byte(plaintext), nil)
	return ciphertext, nil
}

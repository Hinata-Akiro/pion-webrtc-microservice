package chat

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

type MessageStatus string

const (
	StatusSent      MessageStatus = "sent"
	StatusDelivered MessageStatus = "delivered"
	StatusRead      MessageStatus = "read"
	StatusFailed    MessageStatus = "failed"
	StatusQueued    MessageStatus = "queued"
)

type EncryptedMessage struct {
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
}

var encryptionKey = []byte("your-32-byte-encryption-key-here") // In production, use secure key management

func encryptMessage(message []byte) (*EncryptedMessage, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, message, nil)
	return &EncryptedMessage{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
	}, nil
}

func decryptMessage(em *EncryptedMessage) ([]byte, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(em.Ciphertext)
	if err != nil {
		return nil, err
	}

	nonce, err := base64.StdEncoding.DecodeString(em.Nonce)
	if err != nil {
		return nil, err
	}

	return aesgcm.Open(nil, nonce, ciphertext, nil)
} 
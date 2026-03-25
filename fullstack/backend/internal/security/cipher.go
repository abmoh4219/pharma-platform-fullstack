package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

type FieldCipher struct {
	gcm cipher.AEAD
}

func NewFieldCipher(rawKey string) (*FieldCipher, error) {
	key := normalizeKey(rawKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	return &FieldCipher{gcm: gcm}, nil
}

func normalizeKey(rawKey string) []byte {
	trimmed := strings.TrimSpace(rawKey)
	if len(trimmed) == 32 {
		return []byte(trimmed)
	}
	sum := sha256.Sum256([]byte(trimmed))
	return sum[:]
}

func (fc *FieldCipher) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	nonce := make([]byte, fc.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}
	sealed := fc.gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (fc *FieldCipher) Decrypt(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}
	nonceSize := fc.gcm.NonceSize()
	if len(decoded) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := decoded[:nonceSize], decoded[nonceSize:]
	plain, err := fc.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plain), nil
}

func MaskPhone(phone string) string {
	r := []rune(phone)
	if len(r) <= 4 {
		return "****"
	}
	return string(r[:3]) + strings.Repeat("*", len(r)-5) + string(r[len(r)-2:])
}

func MaskID(idNumber string) string {
	r := []rune(idNumber)
	if len(r) <= 4 {
		return "****"
	}
	return string(r[:2]) + strings.Repeat("*", len(r)-4) + string(r[len(r)-2:])
}

func MaskText(input string) string {
	r := []rune(input)
	if len(r) <= 6 {
		return "***"
	}
	return string(r[:3]) + strings.Repeat("*", len(r)-6) + string(r[len(r)-3:])
}

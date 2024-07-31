package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func GenerateRandomString(length int) (string, error) {
	bytes, err := GenerateRandomBytes(length)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bytes), nil
}

func GenerateKeyPair() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate Ed25519 key pair: %v", err)
	}
	return privateKey, publicKey, nil
}

func ExportPrivateKey(privateKey ed25519.PrivateKey) []byte {
	return privateKey
}

func ImportPrivateKey(privateKeyBytes []byte) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return nil, nil, fmt.Errorf("invalid private key size: expected %d bytes, got %d bytes", ed25519.PrivateKeySize, len(privateKeyBytes))
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return privateKey, publicKey, nil
}

func GenerateSymmetricKeyFromText(aesKeyStr string) ([]byte, error) {
	// TODO: maybe change to pbkdf2 later
	hashedKey := sha256.Sum256([]byte(aesKeyStr))

	return hashedKey[:32], nil
}

func EncryptSymmetric(plain []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := GenerateRandomBytes(gcm.NonceSize())
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plain, nil)
	return ciphertext, nil
}

func DecryptSymmetric(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func CheapHash(input string) string {
	// Create a new SHA-256 hash
	hash := sha256.New()

	// Write the input data to the hash
	hash.Write([]byte(input))

	// Calculate the SHA-256 hash and get the result as a byte slice
	hashedData := hash.Sum(nil)

	// Convert the byte slice to a hexadecimal string
	return fmt.Sprintf("%x", hashedData)
}

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
	"math/big"
)

func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func GenerateRandomNumber(max int64) int {
	i, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		panic(err)
	}
	return int(i.Int64())
}

func GenerateRandomStringBase64(length int) string {
	bytes, err := GenerateRandomBytes(length)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(bytes)
}

func GenerateKeyPair() (ed25519.PrivateKey, ed25519.PublicKey) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(fmt.Errorf("failed to generate Ed25519 key pair: %v", err))
	}
	return privateKey, publicKey
}

func ExportPrivateKey(privateKey ed25519.PrivateKey) []byte {
	return privateKey
}

func ImportPrivateKey(privateKeyBytes []byte) (ed25519.PrivateKey, ed25519.PublicKey) {
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		panic(fmt.Errorf("invalid private key size: expected %d bytes, got %d bytes", ed25519.PrivateKeySize, len(privateKeyBytes)))
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return privateKey, publicKey
}

func GenerateSymmetricKeyFromText(aesKeyStr string) []byte {
	hashedKey := sha256.Sum256([]byte(aesKeyStr))
	return hashedKey[:32]
}

func EncryptSymmetric(plain []byte, key []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	nonce, err := GenerateRandomBytes(gcm.NonceSize())
	if err != nil {
		panic(err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plain, nil)
	return ciphertext
}

func DecryptSymmetric(ciphertext []byte, key []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		panic(errors.New("ciphertext too short"))
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	open, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err)
	}

	return open
}

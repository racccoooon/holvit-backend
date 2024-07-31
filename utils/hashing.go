package utils

import (
	"fmt"
	"github.com/go-crypt/crypt"
	"golang.org/x/crypto/bcrypt"
	"holvit/httpErrors"
)

type HashAlgorithm interface {
	Hash(plain string) (string, error)
}

type BCryptHashAlgorithm struct {
	Cost int
}

type UnsupportedHashAlgorithmError struct {
	Algorithm string
}

func (e *UnsupportedHashAlgorithmError) Error() string {
	return fmt.Sprintf("Unsupported hash algorithm: %s", e.Algorithm)
}

func CompareHash(plain string, hash string) error {
	valid, err := crypt.CheckPassword(plain, hash)
	if err != nil {
		return err
	}
	if !valid {
		return httpErrors.Unauthorized()
	}
	return nil
}

func (b *BCryptHashAlgorithm) Hash(plain string) (string, error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plain), b.Cost)
	if err != nil {
		return "", err
	}
	return string(hashBytes), nil
}

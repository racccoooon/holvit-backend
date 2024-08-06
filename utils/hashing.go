package utils

import (
	"crypto/sha512"
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

func CompareHash(plain string, hash string) error {
	sha := CheapHash(plain)
	valid, err := crypt.CheckPassword(sha, hash)
	if err != nil {
		return err
	}
	if !valid {
		return httpErrors.Unauthorized().WithMessage("wrong hash")
	}
	return nil
}

func (b *BCryptHashAlgorithm) Hash(plain string) (string, error) {
	sha := CheapHash(plain)
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(sha), b.Cost)
	if err != nil {
		return "", err
	}
	return string(hashBytes), nil
}

func CheapHash(input string) string {
	hash := sha512.New()
	hash.Write([]byte(input))
	hashedData := hash.Sum(nil)
	return fmt.Sprintf("%x", hashedData)
}

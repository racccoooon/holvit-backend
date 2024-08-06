package utils

import (
	"crypto/sha512"
	"fmt"
	"github.com/go-crypt/crypt"
	"golang.org/x/crypto/bcrypt"
)

type HashAlgorithm interface {
	Hash(plain string) string
}

type BCryptHashAlgorithm struct {
	Cost int
}

func CompareHash(plain string, hash string) bool {
	sha := CheapHash(plain)
	valid, err := crypt.CheckPassword(sha, hash)
	if err != nil {
		panic(err)
	}
	return valid
}

func (b *BCryptHashAlgorithm) Hash(plain string) string {
	sha := CheapHash(plain)
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(sha), b.Cost)
	if err != nil {
		panic(err)
	}
	return string(hashBytes)
}

func CheapHash(input string) string {
	hash := sha512.New()
	hash.Write([]byte(input))
	hashedData := hash.Sum(nil)
	return fmt.Sprintf("%x", hashedData)
}

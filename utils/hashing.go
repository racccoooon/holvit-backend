package utils

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"holvit/constants"
)

type HashAlgorithm interface {
	Hash(plain []byte) ([]byte, error)
	Compare(plain []byte, hash []byte) error
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

func CompareHash(algorithm string, plain []byte, hash []byte) error {
	var algorithmImpl HashAlgorithm
	switch algorithm {
	case constants.HashAlgorithmBCrypt:
		algorithmImpl = &BCryptHashAlgorithm{}
	default:
		return &UnsupportedHashAlgorithmError{
			Algorithm: algorithm,
		}
	}

	return algorithmImpl.Compare(plain, hash)
}

func (b *BCryptHashAlgorithm) Hash(plain []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(plain, b.Cost)
}

func (b *BCryptHashAlgorithm) Compare(plain []byte, hash []byte) error {
	return bcrypt.CompareHashAndPassword(hash, plain)
}

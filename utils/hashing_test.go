package utils

import (
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func Fuzz_CheapHash(f *testing.F) {
	// set corpus
	f.Add("asdasfger")
	f.Add("1294jon1 jjq09r qi3r2i93u  (=()QW=Ü§U )§UJOö")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		// act
		hashed := CheapHash(input)
		hashedAgain := CheapHash(input)

		// assert
		assert.NotEqual(t, input, hashed)
		assert.Equal(t, hashedAgain, hashed)
	})
}

func Fuzz_BCryptAlgorithm(f *testing.F) {
	// set corpus
	f.Add("asdasfger")
	f.Add("fohetwuuw3pmq94cq2m w5uw093 u90w3u09u 098(=)UU(=)§=)q3äö")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		// arrange
		settings := BcryptHashSettings{
			Cost: bcrypt.DefaultCost,
		}
		hasher := settings.MakeHasher()

		// act
		hashed := hasher.Hash(input)
		hashedAgain := hasher.Hash(input)

		res1 := ValidateHash(input, hashed, hasher)
		res2 := ValidateHash(input, hashedAgain, hasher)

		// assert
		assert.NotEqual(t, hashed, input)
		assert.NotEqual(t, hashed, hashedAgain)
		assert.True(t, res1.IsValid)
		assert.True(t, res2.IsValid)
	})
}

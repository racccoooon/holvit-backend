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
		algorithm := BCryptHashAlgorithm{
			Cost: bcrypt.DefaultCost,
		}

		// act
		hashed, err1 := algorithm.Hash(input)
		hashedAgain, err2 := algorithm.Hash(input)

		err3 := CompareHash(input, hashed)
		err4 := CompareHash(input, hashedAgain)

		// assert
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)
		assert.NoError(t, err4)

		assert.NotEqual(t, hashed, input)
		assert.NotEqual(t, hashed, hashedAgain)
	})
}

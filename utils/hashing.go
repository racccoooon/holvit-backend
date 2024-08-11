package utils

import (
	"crypto/sha256"
	"fmt"
	"github.com/gwenya/go-crypt"
	"github.com/gwenya/go-crypt/algorithm"
	"github.com/gwenya/go-crypt/algorithm/argon2"
	"github.com/gwenya/go-crypt/algorithm/bcrypt"
	"github.com/gwenya/go-crypt/algorithm/scrypt"
	"strings"
)

type Hasher interface {
	Hash(plain string) string
	CompareSettings(settings HashSettings) bool
}

type HashSettings interface {
	MakeHasher() Hasher
}

type BcryptHashSettings struct {
	Cost int
}

type BcryptHasher struct {
	hasher   *bcrypt.Hasher
	settings *BcryptHashSettings
}

type ScryptHashSettings struct {
	R            int
	Parallelism  int
	LogN         int
	SaltLength   int
	OutputLength int
}

type ScryptHasher struct {
	hasher   *scrypt.Hasher
	settings *ScryptHashSettings
}

type Argon2idHashSettings struct {
	MemoryCost   uint32
	OpsCost      uint32
	Parallelism  uint32
	OutputLength int
	SaltLength   int
}

type Argon2idHasher struct {
	hasher   *argon2.Hasher
	settings *Argon2idHashSettings
}

type HashValidationResult struct {
	IsValid     bool
	NeedsRehash bool
}

func (s *BcryptHashSettings) MakeHasher() Hasher {
	hasher, err := bcrypt.New(
		bcrypt.WithCost(s.Cost),
	)
	if err != nil {
		panic(err)
	}
	return &BcryptHasher{
		hasher:   hasher,
		settings: s,
	}
}

func (s *ScryptHashSettings) MakeHasher() Hasher {
	hasher, err := scrypt.New(
		scrypt.WithR(s.R),
		scrypt.WithP(s.Parallelism),
		scrypt.WithLN(s.LogN),
		scrypt.WithSaltLength(s.SaltLength),
		scrypt.WithKeyLength(s.OutputLength),
	)
	if err != nil {
		panic(err)
	}
	return &ScryptHasher{
		hasher:   hasher,
		settings: s,
	}
}

func (s *Argon2idHashSettings) MakeHasher() Hasher {
	hasher, err := argon2.New(
		argon2.WithT(int(s.OpsCost)),
		argon2.WithP(int(s.Parallelism)),
		argon2.WithM(s.MemoryCost),
		argon2.WithK(s.OutputLength),
		argon2.WithS(s.SaltLength),
	)
	if err != nil {
		panic(err)
	}
	return &Argon2idHasher{
		hasher:   hasher,
		settings: s,
	}
}

func (h *BcryptHasher) Hash(plain string) string {
	return h.hasher.MustHash(plain).String()
}

func (h *BcryptHasher) CompareSettings(settings HashSettings) bool {
	s, ok := settings.(*BcryptHashSettings)
	if !ok {
		panic(fmt.Errorf("expected settings to be *BcryptHashSettings, got %T", settings))
	}
	return *s == *h.settings
}

func (h *ScryptHasher) Hash(plain string) string {
	return h.hasher.MustHash(plain).String()
}

func (h *ScryptHasher) CompareSettings(settings HashSettings) bool {
	s, ok := settings.(*ScryptHashSettings)
	if !ok {
		panic(fmt.Errorf("expected settings to be *ScryptHashSettings, got %T", settings))
	}
	return *s == *h.settings
}

func (h *Argon2idHasher) Hash(plain string) string {
	return h.hasher.MustHash(plain).String()
}

func (h *Argon2idHasher) CompareSettings(settings HashSettings) bool {
	s, ok := settings.(*Argon2idHashSettings)
	if !ok {
		panic(fmt.Errorf("expected settings to be *Argon2idHashSettings, got %T", settings))
	}
	return *s == *h.settings // TODO: try out what happens if we rename s to settings
}

func checkIfRehashNeeded(digest algorithm.Digest, hasher Hasher) bool {
	var settings HashSettings
	if d, ok := digest.(*bcrypt.Digest); ok {
		settings = &BcryptHashSettings{
			Cost: d.Iterations(),
		}
	} else if d, ok := digest.(*scrypt.Digest); ok {
		settings = &ScryptHashSettings{
			R:            d.R(),
			Parallelism:  d.P(),
			LogN:         d.LN(),
			SaltLength:   len(d.Salt()),
			OutputLength: len(d.Key()),
		}
	} else if d, ok := digest.(*argon2.Digest); ok {
		settings = &Argon2idHashSettings{
			MemoryCost:   d.M(),
			OpsCost:      d.T(),
			Parallelism:  d.P(),
			OutputLength: len(d.Key()),
			SaltLength:   len(d.Salt()),
		}
	} else {
		panic(fmt.Errorf("unsupported digest type: %T", d))
	}
	return hasher.CompareSettings(settings)
}

var decoder *crypt.Decoder

func init() {
	decoder = crypt.NewDecoder()

	err := bcrypt.RegisterDecoder(decoder)
	if err != nil {
		panic(err)
	}

	err = scrypt.RegisterDecoder(decoder)
	if err != nil {
		panic(err)
	}
	err = argon2.RegisterDecoderArgon2id(decoder)
	if err != nil {
		panic(err)
	}
}

func ValidateHash(plain string, hash string, hasher Hasher) HashValidationResult {
	digest, err := decoder.Decode(hash)
	if err != nil {
		panic(err)
	}
	valid, err := digest.MatchAdvanced(plain)
	if err != nil {
		panic(err)
	}

	needsRehash := false
	if valid {
		needsRehash = checkIfRehashNeeded(digest, hasher)
	}

	return HashValidationResult{
		IsValid:     valid,
		NeedsRehash: needsRehash,
	}
}

//////////////////////////

func CheapHash(input string) string {
	return fmt.Sprintf("%x", Sha256(input))
}

func Sha256(input string) []byte {
	hash := sha256.Sum256([]byte(input))
	return hash[:]
}

func Sha256Compare(hash1, hash2 string) bool {
	// don't need constant time compare because we're comparing hashes
	return strings.TrimRight(hash1, "=") == strings.TrimRight(hash2, "=")
}

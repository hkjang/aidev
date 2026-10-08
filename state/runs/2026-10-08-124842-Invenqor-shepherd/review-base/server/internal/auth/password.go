package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

type PasswordPolicy struct {
	MinimumLength int
	MaximumLength int
	RequireUpper  bool
	RequireLower  bool
	RequireNumber bool
	RequireSymbol bool
}

func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinimumLength: 12,
		MaximumLength: 1024,
		RequireUpper:  true,
		RequireLower:  true,
		RequireNumber: true,
		RequireSymbol: true,
	}
}

func (policy PasswordPolicy) Validate(password string) error {
	length := len([]rune(password))
	if length < policy.MinimumLength {
		return fmt.Errorf("password must contain at least %d characters", policy.MinimumLength)
	}
	if policy.MaximumLength > 0 && length > policy.MaximumLength {
		return fmt.Errorf("password must contain no more than %d characters", policy.MaximumLength)
	}
	var upper, lower, number, symbol bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			upper = true
		case unicode.IsLower(char):
			lower = true
		case unicode.IsNumber(char):
			number = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			symbol = true
		}
	}
	if policy.RequireUpper && !upper {
		return errors.New("password must contain an uppercase character")
	}
	if policy.RequireLower && !lower {
		return errors.New("password must contain a lowercase character")
	}
	if policy.RequireNumber && !number {
		return errors.New("password must contain a number")
	}
	if policy.RequireSymbol && !symbol {
		return errors.New("password must contain a symbol")
	}
	return nil
}

type argon2Parameters struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var defaultArgon2Parameters = argon2Parameters{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 4,
	SaltLength:  16,
	KeyLength:   32,
}

func HashPassword(password string) (string, error) {
	parameters := defaultArgon2Parameters
	salt := make([]byte, parameters.SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		parameters.Iterations,
		parameters.Memory,
		parameters.Parallelism,
		parameters.KeyLength,
	)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		parameters.Memory,
		parameters.Iterations,
		parameters.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	parameters, salt, expected, err := parsePasswordHash(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey(
		[]byte(password),
		salt,
		parameters.Iterations,
		parameters.Memory,
		parameters.Parallelism,
		uint32(len(expected)),
	)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func parsePasswordHash(encoded string) (argon2Parameters, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argon2Parameters{}, nil, nil, errors.New("invalid Argon2id hash")
	}
	version, found := strings.CutPrefix(parts[2], "v=")
	if !found {
		return argon2Parameters{}, nil, nil, errors.New("invalid Argon2id version")
	}
	versionNumber, err := strconv.Atoi(version)
	if err != nil || versionNumber != argon2.Version {
		return argon2Parameters{}, nil, nil, errors.New("unsupported Argon2id version")
	}
	var parameters argon2Parameters
	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&parameters.Memory,
		&parameters.Iterations,
		&parameters.Parallelism,
	); err != nil {
		return argon2Parameters{}, nil, nil, errors.New("invalid Argon2id parameters")
	}
	if parameters.Memory < 8*1024 ||
		parameters.Iterations == 0 ||
		parameters.Parallelism == 0 {
		return argon2Parameters{}, nil, nil, errors.New("unsafe Argon2id parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 16 {
		return argon2Parameters{}, nil, nil, errors.New("invalid Argon2id salt")
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) < 16 {
		return argon2Parameters{}, nil, nil, errors.New("invalid Argon2id hash payload")
	}
	parameters.SaltLength = uint32(len(salt))
	parameters.KeyLength = uint32(len(hash))
	return parameters, salt, hash, nil
}

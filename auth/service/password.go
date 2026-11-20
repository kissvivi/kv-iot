package service

import (
	"crypto/subtle"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	passwordHashCost  = bcrypt.DefaultCost
	maxBcryptPassword = 72
)

var (
	ErrPasswordRequired = errors.New("密码不能为空")
	ErrPasswordTooLong  = errors.New("密码长度不能超过72字节")
)

// HashPassword creates a bcrypt hash for newly stored credentials.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrPasswordRequired
	}
	if len([]byte(password)) > maxBcryptPassword {
		return "", ErrPasswordTooLong
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), passwordHashCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword accepts bcrypt hashes and, for one-time migration, historical
// plaintext passwords. needsRehash reports that a successful plaintext or
// weaker bcrypt verification should be persisted as a new bcrypt hash.
func VerifyPassword(encoded, password string) (valid bool, needsRehash bool, err error) {
	if encoded == "" || password == "" {
		return false, false, nil
	}

	if isBcryptHash(encoded) {
		normalized := normalizeBcryptPrefix(encoded)
		compareErr := bcrypt.CompareHashAndPassword([]byte(normalized), []byte(password))
		if compareErr != nil {
			if errors.Is(compareErr, bcrypt.ErrMismatchedHashAndPassword) {
				return false, false, nil
			}
			return false, false, compareErr
		}
		cost, costErr := bcrypt.Cost([]byte(normalized))
		if costErr != nil {
			return true, false, costErr
		}
		return true, cost < passwordHashCost || normalized != encoded, nil
	}

	legacyValid := subtle.ConstantTimeCompare([]byte(encoded), []byte(password)) == 1
	return legacyValid, legacyValid, nil
}

func isBcryptHash(value string) bool {
	return strings.HasPrefix(value, "$2a$") ||
		strings.HasPrefix(value, "$2b$") ||
		strings.HasPrefix(value, "$2x$") ||
		strings.HasPrefix(value, "$2y$")
}

// bcrypt hashes produced by some older systems use the $2y$ prefix. The
// underlying verification format is compatible with $2a$ for this migration.
func normalizeBcryptPrefix(value string) string {
	if strings.HasPrefix(value, "$2y$") || strings.HasPrefix(value, "$2x$") {
		return "$2a$" + value[4:]
	}
	return value
}

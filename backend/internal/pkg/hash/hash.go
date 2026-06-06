package hash

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword returns a bcrypt hash of the given plaintext password.
// Cost is fixed at 12 to balance security and latency.
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hashed), nil
}

// CheckPassword compares a plaintext password against a bcrypt hash.
// Returns nil on match, bcrypt.ErrMismatchedHashAndPassword (or another
// error) otherwise.
func CheckPassword(password, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("check password: %w", err)
	}
	return nil
}

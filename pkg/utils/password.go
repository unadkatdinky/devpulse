package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword takes a plain password and returns a bcrypt hash
// Store the hash — never the original password
// Cost 12 means bcrypt does 4096 rounds of work — slow enough to deter attackers
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPassword compares a plain password against a stored hash
// Returns true if they match
// bcrypt extracts the salt from the hash automatically — you don't manage it
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
// Package auth
package auth

import "github.com/alexedwards/argon2id"

func HashPassword(password string) (string, error) {
	hashParams := argon2id.DefaultParams

	hash, err := argon2id.CreateHash(password, hashParams)
	if err != nil {
		return "", err
	}

	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	passed, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}

	return passed, nil
}

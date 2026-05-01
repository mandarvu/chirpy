package auth

import "testing"

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "common password",
			password: "unset",
		},
		{
			name:     "empty password",
			password: "",
		},
		{
			name:     "password with special characters",
			password: "password!@#$%^&*()./<>?",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			passed := checkAndVerifyPassword(test.password)
			if !passed {
				t.Errorf("CheckPasswordHash() = %v, want %v", passed, true)
			}
		})
	}
}

func checkAndVerifyPassword(password string) bool {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return false
	}

	passed, err := CheckPasswordHash(password, hashedPassword)
	if err != nil {
		return false
	}
	return passed
}


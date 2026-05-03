package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T) {
	tests := []struct {
		name        string
		userID      uuid.UUID
		tokenSecret string
		expiresIn   time.Duration
		expectError bool
	}{
		{
			name:        "normal token",
			userID:      uuid.New(),
			tokenSecret: "mysecret",
			expiresIn:   time.Second * 1,
			expectError: false,
		},
		{
			name:        "empty token secret",
			userID:      uuid.New(),
			tokenSecret: "",
			expiresIn:   time.Second * 1,
			expectError: false,
		},
		{
			name:        "expired token",
			userID:      uuid.New(),
			tokenSecret: "mysecret",
			expiresIn:   -time.Hour,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jwt, err := MakeJWT(test.userID, test.tokenSecret, test.expiresIn)
			if err != nil {
				t.Fatalf("failed to create JWT: %v", err)
			}

			token, err := ValidateJWT(jwt, test.tokenSecret)
			if test.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("failed to validate JWT: %v", err)
			}

			if token != test.userID {
				t.Errorf("expected userID %v, got %v", test.userID, token)
			}
		})
	}
}

func TestGetBearerToken(t *testing.T) {
	header1 := http.Header{}
	header1.Add("Authorization", "Bearer token1")

	header2 := http.Header{}

	tests := []struct {
		name     string
		input    http.Header
		expected string
		isErr    bool
	}{
		{
			name:     "Non empty token",
			input:    header1,
			expected: "token1",
			isErr:    false,
		},
		{
			name:     "Empty token",
			input:    header2,
			expected: "",
			isErr:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := GetBearerToken(test.input)
			if test.isErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("failed to get bearer token: %v", err)
			}

			if token != test.expected {
				t.Errorf("expected token %v, got %v", test.expected, token)
			}
		})
	}	

}

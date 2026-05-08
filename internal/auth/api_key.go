package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	authToken, ok := headers["Authorization"]

	if !ok {
		return "", fmt.Errorf("authorization header not found")
	}

	authToken = strings.Split(authToken[0], " ")

	if len(authToken) != 2 {
		return "", fmt.Errorf("authorization header is malformed")
	}

	if authToken[0] != "ApiKey" {
		return "", fmt.Errorf("authorization header is not api key")
	}

	return authToken[1], nil
}

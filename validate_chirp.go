package main

import (
	"strings"
)

func validateChirpLen(chirp string) bool {
	return len(chirp) <= 140
}

func cleanChirp(chirp string) string {
	profane := map[string]string{
		"kerfuffle": "",
		"sharbert":  "",
		"fornax":    "",
	}

	cleanedBody := []string{}

	for word := range strings.SplitSeq(chirp, " ") {
		if _, ok := profane[strings.ToLower(word)]; ok {
			word = "****"
		}
		cleanedBody = append(cleanedBody, word)
	}
	return strings.Join(cleanedBody, " ")
}

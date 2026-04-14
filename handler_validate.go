package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func handlerChirpsValidate(w http.ResponseWriter, r *http.Request) {
	profane := map[string]string{
		"kerfuffle": "",
		"sharbert":  "",
		"fornax":    "",
	}

	type parameters struct {
		Body string `json:"body"`
	}

	type returnVals struct {
		Valid bool   `json:"valid"`
		Body  string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	cleanedBody := []string{}

	for word := range strings.SplitSeq(params.Body, " ") {
		if _, ok := profane[strings.ToLower(word)]; ok {
			word = "****"
		}
		cleanedBody = append(cleanedBody, word)
	}

	respondWithJSON(w, http.StatusOK, returnVals{
		Valid: true,
		Body:  strings.Join(cleanedBody, " "),
	})
}

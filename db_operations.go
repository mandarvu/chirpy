package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mandarvu/chirpy/internal/database"
)

type userData struct {
	UUID      uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type chirpData struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdateAt  time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *APIConfig) createUser() http.Handler {
	return http.HandlerFunc(func(r http.ResponseWriter, req *http.Request) {
		decoder := json.NewDecoder(req.Body)

		p := userData{}
		err := decoder.Decode(&p)
		if err != nil {
			respondWithError(r, 402, "Could not parse reques", err)
			return
		}

		user, err := cfg.db.CreateUser(req.Context(), p.Email)
		if err != nil {
			respondWithError(r, 400, "could not create user", err)
			return
		} else {
			jsonToReturn := userData{
				UUID:      user.ID,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
				Email:     user.Email,
			}
			respondWithJSON(r, 201, jsonToReturn)
		}
	})
}

func (cfg *APIConfig) dbReset() http.Handler {
	cfg.fileServerHits.Swap(0)
	return http.HandlerFunc(func(r http.ResponseWriter, req *http.Request) {
		// output := fmt.Sprintf("Hits: %d\nResetting counter\nDeleting users from DB", cfg.fileServerHits.Load())
		if cfg.platform == "dev" {
			err := cfg.db.DeleteAllUsers(req.Context())
			if err != nil {
				respondWithError(r, 400, "Could not empty db", err)
				return
			}
			respondWithJSON(r, 200, `{}`)
		} else {
			respondWithError(r, 403, "403 Forbidden", fmt.Errorf("Error"))
		}
	})
}

func (cfg *APIConfig) createChirp() http.Handler {
	return http.HandlerFunc(
		func(r http.ResponseWriter, req *http.Request) {
			decoder := json.NewDecoder(req.Body)

			requestParams := chirpData{}
			err := decoder.Decode(&requestParams)
			if err != nil {
				respondWithError(r, 400, "Invalid parameters", err)
				return
			}

			if validateChirpLen(requestParams.Body) {
				requestParams.Body = cleanChirp(requestParams.Body)
			} else {
				respondWithError(r, 400, "Chirp length too much", fmt.Errorf("chirp too large"))
				return
			}

			chirp, err := cfg.db.CreateChirp(req.Context(), database.CreateChirpParams{
				Body:   requestParams.Body,
				UserID: requestParams.UserID,
			})
			if err != nil {
				respondWithError(r, 400, "Could not create chirp", err)
				return
			}

			respondWithJSON(r, 201, chirpData{
				ID: chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdateAt: chirp.UpdatedAt,
				Body:   chirp.Body,
				UserID: chirp.UserID,
			})
		},
	)
}

func (cfg *APIConfig) getChirps() http.Handler {
	return http.HandlerFunc(
		func(r http.ResponseWriter, req *http.Request) {
			chirpID := req.PathValue("ChirpID")

			if chirpID != "" {
				chirpUUID, err := uuid.Parse(chirpID)
				if err != nil {
					respondWithError(r, 400, "could not parse uuid for chirp", err)
					return
				}

				chirp, err := cfg.db.GetChirpFromID(req.Context(), chirpUUID)
				if err != nil {
					respondWithError(r, 404, "Chirp not found", err)
					return
				}

				respondWithJSON(r, 200, chirpData{
					ID:        chirp.ID,
					CreatedAt: chirp.CreatedAt,
					UpdateAt:  chirp.CreatedAt,
					Body:      chirp.Body,
					UserID:    chirp.UserID,
				})
				return
			} else {
				chirps, err := cfg.db.GetAllChirps(req.Context())
				if err != nil {
					respondWithError(r, 400, "could not get all chirps", err)
					return
				}

				output := []chirpData{}

				for _, c := range chirps {
					output = append(output, chirpData{
						ID:        c.ID,
						CreatedAt: c.CreatedAt,
						UpdateAt:  c.UpdatedAt,
						Body:      c.Body,
						UserID:    c.UserID,
					})
				}

				respondWithJSON(r, 200, output)
			}
		},
	)
}

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

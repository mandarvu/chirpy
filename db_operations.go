package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type userData struct {
	UUID      uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type chirpData struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

func (cfg *APIConfig) createUser() http.Handler {
	return http.HandlerFunc(func(r http.ResponseWriter, req *http.Request) {
		decoder := json.NewDecoder(req.Body)

		p := userData{}
		err := decoder.Decode(&p)
		if err != nil {
			respondWithError(r, 402, "Could not parse reques", err)
		}

		user, err := cfg.db.CreateUser(req.Context(), p.Email)
		if err != nil {
			respondWithError(r, 400, "could not create user", err)
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
			err := decoder.Buffered(&requestParams)
		},
	)
}

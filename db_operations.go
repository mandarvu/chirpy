package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mandarvu/chirpy/internal/auth"
	"github.com/mandarvu/chirpy/internal/database"
)

type userData struct {
	UUID      uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	JWT       string    `json:"token,omitempty"`
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

		p := struct {
			Email   string `json:"email"`
			Pasword string `json:"password"`
		}{}

		err := decoder.Decode(&p)
		if err != nil {
			respondWithError(r, 402, "Could not parse reques", err)
			return
		}

		if p.Pasword == "" {
			respondWithError(r, 400, "Password not provided", fmt.Errorf("no password provided in request"))
			return
		}

		hashedPassword, err := auth.HashPassword(p.Pasword)
		if err != nil {
			respondWithError(r, 400, "Could not hash password", err)
			return
		}

		user, err := cfg.db.CreateUser(
			req.Context(),
			database.CreateUserParams{
				Email:          p.Email,
				HashedPassword: hashedPassword,
			},
		)

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

			bearerToken, err := auth.GetBearerToken(req.Header)
			if err != nil {
				respondWithError(r, 400, "bearer token not found", err)
				return
			}

			log.Printf("jwt secret loaded: %q", cfg.jwtSecret)
			userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)
			if err != nil {
				respondWithError(r, 401, "user could not be authenticated", err)
				return
			}

			requestParams.UserID = userID

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
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdateAt:  chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
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

func (cfg *APIConfig) loginUser() http.Handler {
	return http.HandlerFunc(
		func(r http.ResponseWriter, req *http.Request) {
			const defaultExpirationTime = 3600
			d := json.NewDecoder(req.Body)

			p := struct {
				Email     string `json:"email"`
				Password  string `json:"password"`
				ExpiresIn int    `json:"expires_in_seconds"`
			}{}

			err := d.Decode(&p)
			if err != nil {
				respondWithError(r, 400, "could not parse request", err)
				return
			}

			user, userErr := cfg.db.GetUserFromEmail(req.Context(), p.Email)

			if userErr != nil {
				respondWithError(r, 401, "Unauthorized", userErr)
				return
			}

			passCheck, passErr := auth.CheckPasswordHash(p.Password, user.HashedPassword)

			if passErr != nil || !passCheck {
				respondWithError(r, 401, "Unauthorized", passErr)
				return
			}

			if p.ExpiresIn == 0 || p.ExpiresIn >= 3600 {
				p.ExpiresIn = defaultExpirationTime
			}

			log.Printf("jwt secret loaded: %q", cfg.jwtSecret)
			log.Printf("expires in: %d seconds", p.ExpiresIn)
			jwtToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Second * time.Duration(p.ExpiresIn))
			if err != nil {
				respondWithError(r, 400, "Could not create jwt", err)
				return
			}

			respondWithJSON(r, 200, userData{
				UUID:      user.ID,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
				Email:     user.Email,
				JWT:       jwtToken,
			})
		},
	)
}

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/mandarvu/chirpy/internal/auth"
	"github.com/mandarvu/chirpy/internal/database"
)

const defaultExpirationTime = 3600

type userData struct {
	UUID         uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
	JWT          string    `json:"token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
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
				UUID:        user.ID,
				CreatedAt:   user.CreatedAt,
				UpdatedAt:   user.UpdatedAt,
				Email:       user.Email,
				IsChirpyRed: user.IsChirpyRed,
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
				reqAuthorID := req.URL.Query().Get("author_id")
				sortOrder := req.URL.Query().Get("sort")

				reqUUID := uuid.NullUUID{}

				if reqAuthorID == "" {
					reqUUID.Valid = false
				} else {
					if tmpUUID, err := uuid.Parse(reqAuthorID); err != nil {
						respondWithError(r, 400, "uuid invalid", err)
						return
					} else {
						reqUUID.UUID = tmpUUID
						reqUUID.Valid = true
					}
				}

				chirps, err := cfg.db.GetAllChirps(req.Context(), reqUUID)
				if err != nil {
					respondWithError(r, 400, "could not get all chirps", err)
					return
				}

				output := []chirpData{}

				if sortOrder == "desc" {
					sort.Slice(chirps, func(i, j int) bool {
						return chirps[i].CreatedAt.Compare(chirps[j].CreatedAt) == 1
					})
				}

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
			d := json.NewDecoder(req.Body)

			p := struct {
				Email    string `json:"email"`
				Password string `json:"password"`
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

			refreshToken := auth.MakeRefreshToken()

			refToken, err := cfg.db.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{
				Token:  refreshToken,
				UserID: user.ID,
				ExpiresAt: sql.NullTime{
					Time:  time.Now().Add(time.Hour * 60 * 24), // 60 days = 60 x 24 hours
					Valid: true,
				},
			})
			if err != nil {
				respondWithError(r, 400, "could not add refresh token", err)
			}

			jwtToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Second*time.Duration(defaultExpirationTime))
			if err != nil {
				respondWithError(r, 400, "Could not create jwt", err)
				return
			}

			respondWithJSON(r, 200, userData{
				UUID:         user.ID,
				CreatedAt:    user.CreatedAt,
				UpdatedAt:    user.UpdatedAt,
				Email:        user.Email,
				IsChirpyRed:  user.IsChirpyRed,
				JWT:          jwtToken,
				RefreshToken: refToken.Token,
			})
		},
	)
}

func (cfg *APIConfig) refreshJWTFromRefreshToken() http.Handler {
	return http.HandlerFunc(
		func(r http.ResponseWriter, req *http.Request) {
			bearerToken, err := auth.GetBearerToken(req.Header)
			if err != nil {
				respondWithError(r, 400, "bearer token not found", err)
				return
			}

			tokenData, err := cfg.db.GetDataFromRefreshToken(req.Context(), bearerToken)
			if err != nil {
				respondWithError(r, 401, "refresh token not found", err)
				return
			}

			// HACK: The timestampm comparison below returns -1 if the input time to
			// the method is after the time being operated on. otherwise 1 and 0 if same.
			if tokenData.ExpiresAt.Time.Compare(time.Now()) == -1 || tokenData.RevokedAt.Valid {
				respondWithError(r, 401, "refresh token expired or revoked", fmt.Errorf("refresh token expired or revoked"))
				return
			}

			jwt, err := auth.MakeJWT(tokenData.UserID, cfg.jwtSecret, time.Duration(time.Hour))
			if err != nil {
				respondWithError(r, 400, "could not create jwt", err)
				return
			}

			respondWithJSON(r, 200, struct {
				Token string `json:"token"`
			}{
				Token: jwt,
			})
		},
	)
}

func (cfg *APIConfig) revokeRefreshToken() http.Handler {
	return http.HandlerFunc(
		func(r http.ResponseWriter, req *http.Request) {
			bearerToken, err := auth.GetBearerToken(req.Header)
			if err != nil {
				respondWithError(r, 400, "bearer token not found", err)
				return
			}

			err = cfg.db.RevokeRefreshToken(req.Context(), database.RevokeRefreshTokenParams{
				Token:     bearerToken,
				UpdatedAt: time.Now(),
				RevokedAt: sql.NullTime{
					Time:  time.Now(),
					Valid: true,
				},
			})
			log.Printf("Revoke query executed")
			if err != nil {
				respondWithError(r, 400, "Could not revoke", err)
				return
			}

			respondWithJSON(r, 204, struct{}{})
		},
	)
}

func (cfg *APIConfig) updateUserRecords() http.Handler {
	return http.HandlerFunc(
		func(r http.ResponseWriter, req *http.Request) {
			bearerToken, err := auth.GetBearerToken(req.Header)
			if err != nil {
				respondWithError(r, 401, "could not get bearer token", err)
				return
			}

			userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)
			if err != nil {
				respondWithError(r, 401, "user unauthorized", err)
				return
			}

			p := struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}{}

			decoder := json.NewDecoder(req.Body)
			err = decoder.Decode(&p)
			if err != nil {
				respondWithError(r, 401, "could not parse body", err)
				return
			}

			hashPass, err := auth.HashPassword(p.Password)
			if err != nil {
				respondWithError(r, 401, "could not hash password", err)
				return
			}

			userDetails, err := cfg.db.UpdateUserRecords(req.Context(), database.UpdateUserRecordsParams{
				ID:             userID,
				Email:          p.Email,
				HashedPassword: hashPass,
				UpdatedAt:      time.Now(),
			})
			if err != nil {
				respondWithError(r, 401, "could not update data", err)
				return
			}

			respondWithJSON(r, 200, userData{
				UUID:      userDetails.ID,
				CreatedAt: userDetails.CreatedAt,
				UpdatedAt: userDetails.UpdatedAt,
				Email:     userDetails.Email,
			})
		},
	)
}

func (cfg *APIConfig) deleteChirpFromID() http.Handler {
	return http.HandlerFunc(
		func(r http.ResponseWriter, req *http.Request) {
			bearerToken, err := auth.GetBearerToken(req.Header)
			if err != nil {
				respondWithError(r, 401, "could not get bearer token", err)
				return
			}

			userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)
			if err != nil {
				respondWithError(r, 400, "could not authenticate jwt", err)
				return
			}

			chirpID := req.PathValue("ChirpID")

			if chirpID == "" {
				respondWithError(r, 400, "chirp ID not provided", fmt.Errorf("chirp ID required for deletion"))
				return
			}

			chirpUUID, err := uuid.Parse(chirpID)
			if err != nil {
				respondWithError(r, 400, "could not parse provided chirp id", err)
				return
			}

			chirp, err := cfg.db.GetChirpFromID(req.Context(), chirpUUID)
			if err != nil {
				respondWithError(r, 404, "chirp not found", err)
				return
			}

			if chirp.UserID != userID {
				respondWithError(r, 403, "only author of chirp can delete chirp", fmt.Errorf("%s is not author of chirp %v", userID, chirpID))
				return
			}

			err = cfg.db.DeleteChirpFromID(req.Context(), database.DeleteChirpFromIDParams{
				ID:     chirp.ID,
				UserID: chirp.UserID,
			})
			if err != nil {
				respondWithError(r, 400, "could not delete chirp", err)
				return
			}

			respondWithJSON(r, 204, struct{}{})
		},
	)
}

func (cfg *APIConfig) upgradeUserToChirpyRed() http.Handler {
	return http.HandlerFunc(
		func(r http.ResponseWriter, req *http.Request) {
			apikey, err := auth.GetAPIKey(req.Header)
			if err != nil {
				respondWithError(r, 401, "api key not found", err)
				return
			}

			if apikey != cfg.polkaKey {
				respondWithError(r, 401, "api key mismatch", err)
				return
			}

			p := struct {
				Event string `json:"event"`
				Data  struct {
					UserID string `json:"user_id"`
				} `json:"data"`
			}{}

			decoder := json.NewDecoder(req.Body)

			err = decoder.Decode(&p)
			if err != nil {
				respondWithError(r, 403, "request malformed", err)
				return
			}

			if p.Event != "user.upgraded" {
				respondWithJSON(r, 204, struct{}{})
				return
			}

			userUUID, err := uuid.Parse(p.Data.UserID)
			if err != nil {
				respondWithError(r, 400, "could not parse given UUID", err)
				return
			}

			user, err := cfg.db.UpgradeUserChirpyRedStatus(req.Context(), userUUID)
			if err != nil {
				respondWithError(r, 404, "user cannot be found", err)
				return
			}

			respondWithJSON(r, 204, userData{
				UUID:        user.ID,
				CreatedAt:   user.CreatedAt,
				UpdatedAt:   user.UpdatedAt,
				Email:       user.Email,
				IsChirpyRed: user.IsChirpyRed,
			})
		},
	)
}

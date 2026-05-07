package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mandarvu/chirpy/internal/database"
)

type APIConfig struct {
	fileServerHits atomic.Int32
	db             *database.Queries
	platform       string
	jwtSecret      string
}

func dirHandler(dir string) http.Handler {
	return http.StripPrefix("/"+dir, http.FileServer(http.Dir("./"+dir)))
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	jwtSecret := os.Getenv("JWT_SECRET")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Could not establish connection with db: %v\n", err)
	}
	log.Printf("jwt secret loaded: %q", jwtSecret)
	mux := http.NewServeMux()
	conf := APIConfig{}

	conf.db = database.New(db)
	conf.platform = platform
	conf.jwtSecret = jwtSecret

	mux.Handle("/app/", conf.middlewareMetricsInc(dirHandler("app")))
	mux.Handle("/assets/", dirHandler("assets"))
	mux.HandleFunc("GET /api/healthz", statusHandler)
	mux.Handle("GET /admin/metrics", conf.metricHandler())
	mux.Handle("POST /admin/reset", conf.dbReset())
	mux.Handle("POST /api/users", conf.createUser())
	mux.Handle("POST /api/chirps", conf.createChirp())
	mux.Handle("GET /api/chirps/{ChirpID}", conf.getChirps())
	mux.Handle("POST /api/login", conf.loginUser())
	mux.Handle("POST /api/refresh", conf.refreshJWTFromRefreshToken())
	mux.Handle("POST /api/revoke", conf.revokeRefreshToken())
	mux.Handle("PUT /api/users", conf.updateUserRecords())

	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	server.ListenAndServe()
}

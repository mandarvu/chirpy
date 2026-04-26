package main

import (
	"database/sql"
	"fmt"
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
}

func dirHandler(dir string) http.Handler {
	return http.StripPrefix("/"+dir, http.FileServer(http.Dir("./"+dir)))
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Could not establish connection with db: %v\n", err)
	}

	mux := http.NewServeMux()
	conf := APIConfig{}

	conf.db = database.New(db)
	conf.platform = platform

	mux.Handle("/app/", conf.middlewareMetricsInc(dirHandler("app")))
	mux.Handle("/assets/", dirHandler("assets"))
	mux.HandleFunc("GET /api/healthz", statusHandler)
	mux.Handle("GET /admin/metrics", conf.metricHandler())
	mux.Handle("POST /admin/reset", conf.dbReset())
	mux.HandleFunc("POST /api/validate_chirp", handlerChirpsValidate)
	mux.Handle("POST /api/users", conf.createUser())

	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	server.ListenAndServe()
}

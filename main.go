package main

import (
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileServerHits atomic.Int32
}

func main() {
	mux := http.NewServeMux()
	conf := apiConfig{}

	dirHandler := func(dir string) http.Handler {
		return http.StripPrefix("/"+dir, http.FileServer(http.Dir("./"+dir)))
	}

	mux.Handle("/app/", conf.middlewareMetricsInc(dirHandler("app")))
	mux.Handle("/assets/", dirHandler("assets"))
	mux.HandleFunc("GET /api/healthz", statusHandler)
	mux.Handle("GET /admin/metrics", conf.metricHandler())
	mux.Handle("POST /admin/reset", conf.metricReset())
	mux.HandleFunc("POST /api/validate_chirp", handlerChirpsValidate)

	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	server.ListenAndServe()
}

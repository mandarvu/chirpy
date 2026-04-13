package main

import (
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(r http.ResponseWriter, req *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(r, req)
	})
}

func main() {
	mux := http.NewServeMux()

	conf := apiConfig{
		fileServerHits: atomic.Int32{},
	}

	dirHandler := func(dir string) http.Handler {
		return http.StripPrefix("/"+dir, http.FileServer(http.Dir("./"+dir)))
	}

	mux.Handle("/app/", conf.middlewareMetricsInc(dirHandler("app")))

	mux.Handle("/assets/", dirHandler("assets"))

	statusHandler := func(r http.ResponseWriter, req *http.Request) {
		r.Header().Add("Content-type", "text/plain; charset=utf-8")
		r.WriteHeader(200)
		r.Write([]byte("OK"))
	}

	mux.HandleFunc("/healthz", statusHandler)

	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	server.ListenAndServe()
}

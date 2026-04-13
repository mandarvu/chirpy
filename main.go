package main

import (
	"fmt"
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

func (cfg *apiConfig) metricHandler() http.Handler {
	return http.HandlerFunc(func(r http.ResponseWriter, req *http.Request) {
		output := fmt.Sprintf("Hits: %d", cfg.fileServerHits.Load())

		r.Header().Add("Content-type", "text/plain; charset=utf-8")
		r.WriteHeader(200)
		r.Write([]byte(output))
	})
}

func (cfg *apiConfig) metricReset() http.Handler {
	cfg.fileServerHits.Swap(0)
	return http.HandlerFunc(func(r http.ResponseWriter, req *http.Request) {
		output := fmt.Sprintf("Hits: %d\nResetting counter", cfg.fileServerHits.Load())
		r.Header().Add("Content-type", "text/plain; charset=utf-8")
		r.WriteHeader(200)
		r.Write([]byte(output))
	})
}

func main() {
	mux := http.NewServeMux()

	conf := apiConfig{}

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

	mux.HandleFunc("GET /healthz", statusHandler)

	mux.Handle("GET /metrics", conf.metricHandler())

	mux.Handle("POST /reset", conf.metricReset())

	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	server.ListenAndServe()
}

package main

import (
	"fmt"
	"net/http"
)

func (cfg *APIConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(r http.ResponseWriter, req *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(r, req)
	})
}

func (cfg *APIConfig) metricHandler() http.Handler {
	return http.HandlerFunc(func(r http.ResponseWriter, req *http.Request) {
		html := `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
        </html>`
		output := fmt.Sprintf(html, cfg.fileServerHits.Load())

		r.Header().Add("Content-type", "text/html; charset=utf-8")
		r.WriteHeader(200)
		r.Write([]byte(output))
	})
}



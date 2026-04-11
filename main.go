package main

import (
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("./app"))))

	mux.Handle("/assets/", http.StripPrefix("/app", http.FileServer(http.Dir("./app/assets"))))

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

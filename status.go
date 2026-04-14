package main

import "net/http"

func statusHandler(r http.ResponseWriter, req *http.Request) {
	r.Header().Add("Content-type", "text/plain; charset=utf-8")
	r.WriteHeader(200)
	r.Write([]byte("OK"))
}

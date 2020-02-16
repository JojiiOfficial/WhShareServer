package main

import (
	"log"
	"net/http"
	"time"
)

//Logger logs stuff
func Logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method + " " + r.RequestURI + " " + name)
		start := time.Now()
		inner.ServeHTTP(w, r)
		dur := time.Since(start)
		if dur > 1500*time.Millisecond {
			log.Println("Duration: " + dur.String())
		}
	})
}

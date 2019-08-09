package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"

	log "github.com/sirupsen/logrus"
)


func init() {
	// Setting default level to debug
	log.SetLevel(log.DebugLevel)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		target := r.URL
		target.Scheme = "https"
		target.Host = "docker.io"

		http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
	})

	port := os.Getenv("PORT")
	log.Fatal(http.ListenAndServe(":" + port, nil))
}
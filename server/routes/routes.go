package routes

import (
	"net/http"

	"trackseek/fingerprint"
)

func Register(mux *http.ServeMux, staticDir string, index *fingerprint.InMemoryIndex) {
	mux.Handle("/match", http.HandlerFunc(matchSample(index)))
	mux.Handle("/", staticHandler(staticDir))
}

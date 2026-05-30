package routes

import "net/http"

func Register(mux *http.ServeMux, staticDir string) {
	mux.Handle("/match", http.HandlerFunc(matchSample))
	mux.Handle("/", staticHandler(staticDir))
}

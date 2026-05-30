package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"trackseek/server/routes"
)

func Run(addr string) error {
	staticDir := filepath.Clean("static")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		return err
	}

	mux := http.NewServeMux()
	routes.Register(mux, staticDir)

	log.Printf("trackseek server listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

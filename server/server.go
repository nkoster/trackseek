package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"trackseek/db"
	"trackseek/fingerprint"
	"trackseek/server/routes"
)

func Run(addr string, preload bool) error {
	staticDir := filepath.Clean("static")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		return err
	}

	var index *fingerprint.InMemoryIndex
	if preload {
		log.Printf("preloading fingerprint index into memory")

		loadedIndex, err := fingerprint.BuildInMemoryIndex(db.DB)
		if err != nil {
			return err
		}

		index = loadedIndex
		log.Printf("loaded in-memory fingerprint index with %d hashes and %d tracks", len(index.HitsByHash), len(index.Tracks))
	}

	mux := http.NewServeMux()
	routes.Register(mux, staticDir, index)

	log.Printf("trackseek server listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

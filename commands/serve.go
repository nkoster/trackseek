package commands

import (
	"flag"
	"fmt"
	"io"

	"trackseek/server"
)

func RunServe(args []string) error {
	var addr string
	var preload bool

	serveFlags := flag.NewFlagSet("serve", flag.ContinueOnError)
	serveFlags.SetOutput(io.Discard)
	serveFlags.StringVar(&addr, "addr", ":8080", "http listen address")
	serveFlags.BoolVar(&preload, "preload", false, "preload fingerprints into an in-memory index for faster HTTP matching")
	if err := serveFlags.Parse(args); err != nil {
		return fmt.Errorf("%w\nusage: serve [--addr :8080] [--preload]", err)
	}

	return server.Run(addr, preload)
}

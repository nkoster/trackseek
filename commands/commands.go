package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"trackseek/db"
)

type command struct {
	name        string
	description string
	run         func(args []string) error
}

var commandRegistry = map[string]command{
	"index": {
		name:        "index",
		description: "add or update a track in the fingerprint database",
		run:         RunIndex,
	},
	"match": {
		name:        "match",
		description: "match an audio sample against indexed tracks",
		run:         RunMatch,
	},
	"list": {
		name:        "list",
		description: "list indexed tracks",
		run:         RunList,
	},
	"serve": {
		name:        "serve",
		description: "start the HTTP server",
		run:         RunServe,
	},
}

func Run(args []string) error {
	program := "trackseek"
	if len(args) > 0 {
		program = args[0]
	}

	if len(args) < 2 {
		return fmt.Errorf("no command provided\n\n%s", usageText(program))
	}

	selectedCommand, ok := commandRegistry[args[1]]
	if !ok {
		return fmt.Errorf("unknown command %q\n\n%s", args[1], usageText(program))
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	dotEnvPath := filepath.Join(filepath.Dir(execPath), ".env")
	if err := loadDotEnv(dotEnvPath); err != nil {
		return err
	}

	log.Printf("using sqlite database: %s", db.CurrentDBPath())

	if err := db.InitDB(); err != nil {
		return err
	}
	defer db.Close()

	return selectedCommand.run(args[2:])
}

func usageText(program string) string {
	keys := make([]string, 0, len(commandRegistry))
	for name := range commandRegistry {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	var b strings.Builder
	fmt.Fprintf(&b, "usage: %s <command> [args]\n\n", program)
	b.WriteString("commands:\n")
	for _, name := range keys {
		cmd := commandRegistry[name]
		fmt.Fprintf(&b, "  %-6s %s\n", cmd.name, cmd.description)
	}

	return strings.TrimRight(b.String(), "\n")
}

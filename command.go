package main

import (
	"fmt"
	"sort"
	"strings"

	"trackseek/commands"
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
		run:         commands.RunIndex,
	},
	"match": {
		name:        "match",
		description: "match an audio sample against indexed tracks",
		run:         commands.RunMatch,
	},
	"list": {
		name:        "list",
		description: "list indexed tracks",
		run:         commands.RunList,
	},
	"serve": {
		name:        "serve",
		description: "start the HTTP server",
		run:         commands.RunServe,
	},
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

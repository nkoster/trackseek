package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"trackseek/db"
)

func main() {
	log.SetFlags(0)

	if err := run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
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

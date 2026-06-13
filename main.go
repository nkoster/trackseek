package main

import (
	"log"
	"os"

	"trackseek/commands"
)

func main() {
	log.SetFlags(0)

	if err := commands.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

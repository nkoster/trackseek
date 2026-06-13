package commands

import (
	"flag"
	"fmt"
	"io"

	"trackseek/models"
)

func RunList(args []string) error {
	listFlags := flag.NewFlagSet("list", flag.ContinueOnError)
	listFlags.SetOutput(io.Discard)
	if err := listFlags.Parse(args); err != nil {
		return fmt.Errorf("%w\nusage: list", err)
	}

	if listFlags.NArg() > 0 {
		return fmt.Errorf("usage: list")
	}

	tracks, err := models.ListTracks()
	if err != nil {
		return err
	}

	for _, track := range tracks {
		fmt.Printf("%d  %s - %s [%s]\n", track.ID, track.Artist.Name, track.Title, track.Path)
	}

	return nil
}

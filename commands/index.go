package commands

import (
	"flag"
	"fmt"
	"io"

	"trackseek/audio"
	"trackseek/db"
	"trackseek/fingerprint"
	"trackseek/models"
)

func RunIndex(args []string) error {
	var title string
	var artist string

	indexFlags := flag.NewFlagSet("index", flag.ContinueOnError)
	indexFlags.SetOutput(io.Discard)
	indexFlags.StringVar(&title, "title", "", "track title")
	indexFlags.StringVar(&artist, "artist", "", "artist name")
	if err := indexFlags.Parse(args); err != nil {
		return fmt.Errorf("%w\nusage: index file.wav --title \"Title\" --artist \"Artist\"", err)
	}

	if indexFlags.NArg() < 1 {
		return fmt.Errorf("usage: index file.wav --title \"Title\" --artist \"Artist\"")
	}

	audioPath := indexFlags.Arg(0)
	if title == "" {
		return fmt.Errorf("index requires --title")
	}

	samples, sampleRate, err := readAudio(audioPath)
	if err != nil {
		return err
	}

	if artist == "" {
		artist = "Unknown Artist"
	}

	peaks := fingerprint.ExtractPeaks(samples)
	fmt.Printf("found %d peaks\n", len(peaks))

	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	trackID, err := (models.Track{Path: audioPath, Title: title, Artist: &models.Artist{Name: artist}}).UpsertByPathTx(tx)
	if err != nil {
		return err
	}

	count, err := fingerprint.ReplaceTrackFingerprintsTx(tx, trackID, sampleRate, peaks)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("stored %d hashes for track_id=%d\n", count, trackID)
	return nil
}

func readAudio(audioPath string) ([]float64, int, error) {
	samples, sampleRate, err := audio.ReadMono(audioPath)
	if err != nil {
		return nil, 0, err
	}

	fmt.Printf("loaded %d samples, sample rate %d Hz\n", len(samples), sampleRate)
	return samples, sampleRate, nil
}

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"trackseek/audio"
	"trackseek/db"
	"trackseek/fingerprint"
	"trackseek/models"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <index|match> ...", os.Args[0])
	}

	command := os.Args[1]

	var audioPath string
	var title string
	var artist string
	var minScore int
	var threshold int

	switch command {
	case "index":
		indexFlags := flag.NewFlagSet("index", flag.ExitOnError)
		indexFlags.StringVar(&title, "title", "", "track title")
		indexFlags.StringVar(&artist, "artist", "", "artist name")
		if err := indexFlags.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}

		if indexFlags.NArg() < 1 {
			log.Fatalf("usage: %s index file.wav --title \"Title\" --artist \"Artist\"", os.Args[0])
		}

		audioPath = indexFlags.Arg(0)
		if title == "" || artist == "" {
			log.Fatal("index requires both --title and --artist")
		}
	case "match":
		matchFlags := flag.NewFlagSet("match", flag.ExitOnError)
		matchFlags.IntVar(&minScore, "min-score", 0, "minimum score required to accept a match")
		matchFlags.IntVar(&threshold, "threshold", 0, "score at which matching stops early and accepts the current best candidate")
		if err := matchFlags.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}

		if matchFlags.NArg() < 1 {
			log.Fatalf("usage: %s match [--min-score N] [--threshold N] sample.wav", os.Args[0])
		}

		audioPath = matchFlags.Arg(0)
	default:
		log.Fatalf("unknown command %q, expected index or match", command)
	}

	if err := db.InitDB(); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	samples, sampleRate, err := audio.ReadMono(audioPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("loaded %d samples, sample rate %d Hz\n", len(samples), sampleRate)

	peaks := fingerprint.ExtractPeaks(samples)
	fmt.Printf("found %d peaks\n", len(peaks))

	switch command {
	case "index":
		trackID, err := (models.Track{Path: audioPath, Title: title, Artist: &models.Artist{Name: artist}}).Save()
		if err != nil {
			log.Fatal(err)
		}

		count, err := fingerprint.StoreFingerprints(db.DB, trackID, sampleRate, peaks)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("stored %d hashes for track_id=%d\n", count, trackID)
	case "match":
		result, err := fingerprint.MatchFingerprints(db.DB, sampleRate, peaks, threshold)
		if err != nil {
			if errors.Is(err, fingerprint.ErrNoMatch) {
				fmt.Println("no matching track found")
				return
			}

			log.Fatal(err)
		}

		if result.Score < minScore {
			fmt.Printf("no matching track found (best score %d below min-score %d)\n", result.Score, minScore)
			return
		}

		track, err := models.GetTrackByID(result.TrackID)
		if err != nil {
			log.Fatal(err)
		}

		if result.EarlyStopped {
			fmt.Printf("best match: track_id=%d title=%q artist=%q path=%s [early stopped] offset_ms=%d\n", result.TrackID, track.Title, track.Artist.Name, track.Path, result.OffsetMS)
			return
		}

		fmt.Printf("best match: track_id=%d title=%q artist=%q path=%s score=%d offset_ms=%d\n", result.TrackID, track.Title, track.Artist.Name, track.Path, result.Score, result.OffsetMS)
	}
}

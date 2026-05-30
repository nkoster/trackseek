package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"trackseek/audio"
	"trackseek/db"
	"trackseek/fingerprint"
	"trackseek/models"
	"trackseek/server"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <index|match|serve> ...", os.Args[0])
	}

	command := os.Args[1]

	var audioPath string
	var addr string
	var preload bool
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
	case "serve":
		serveFlags := flag.NewFlagSet("serve", flag.ExitOnError)
		serveFlags.StringVar(&addr, "addr", ":8080", "http listen address")
		serveFlags.BoolVar(&preload, "preload", false, "preload fingerprints into an in-memory index for faster HTTP matching")
		if err := serveFlags.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown command %q, expected index, match or serve", command)
	}

	if err := loadDotEnv(".env"); err != nil {
		log.Fatal(err)
	}

	log.Printf("using sqlite database: %s", db.CurrentDBPath())

	if err := db.InitDB(); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if command == "serve" {
		if err := server.Run(addr, preload); err != nil {
			log.Fatal(err)
		}
		return
	}

	switch command {
	case "index":
		samples, sampleRate, err := readAudio(audioPath)
		if err != nil {
			log.Fatal(err)
		}

		peaks := fingerprint.ExtractPeaks(samples)
		fmt.Printf("found %d peaks\n", len(peaks))

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
		result, err := fingerprint.MatchAudioFile(db.DB, audioPath, minScore, threshold)
		if err != nil {
			log.Fatal(err)
		}

		if !result.Matched {
			if result.Score > 0 && minScore > 0 {
				fmt.Printf("no matching track found (best score %d below min-score %d)\n", result.Score, minScore)
				return
			}

			fmt.Println("no matching track found")
			return
		}

		if result.EarlyStopped {
			fmt.Printf("best match: track_id=%d title=%q artist=%q path=%s [early stopped] offset_ms=%d\n", result.TrackID, result.Title, result.Artist, result.Path, result.OffsetMS)
			return
		}

		fmt.Printf("best match: track_id=%d title=%q artist=%q path=%s score=%d offset_ms=%d\n", result.TrackID, result.Title, result.Artist, result.Path, result.Score, result.OffsetMS)
	}
}

func readAudio(audioPath string) ([]float64, int, error) {
	samples, sampleRate, err := audio.ReadMono(audioPath)
	if err != nil {
		return nil, 0, err
	}

	fmt.Printf("loaded %d samples, sample rate %d Hz\n", len(samples), sampleRate)

	return samples, sampleRate, nil
}

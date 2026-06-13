package commands

import (
	"flag"
	"fmt"
	"io"

	"trackseek/db"
	"trackseek/fingerprint"
)

func RunMatch(args []string) error {
	var minScore int
	var threshold int

	matchFlags := flag.NewFlagSet("match", flag.ContinueOnError)
	matchFlags.SetOutput(io.Discard)
	matchFlags.IntVar(&minScore, "min-score", 0, "minimum score required to accept a match")
	matchFlags.IntVar(&threshold, "threshold", 0, "score at which matching stops early and accepts the current best candidate")
	if err := matchFlags.Parse(args); err != nil {
		return fmt.Errorf("%w\nusage: match [--min-score N] [--threshold N] sample.wav", err)
	}

	if matchFlags.NArg() < 1 {
		return fmt.Errorf("usage: match [--min-score N] [--threshold N] sample.wav")
	}

	audioPath := matchFlags.Arg(0)
	result, err := fingerprint.MatchAudioFile(db.DB, audioPath, minScore, threshold)
	if err != nil {
		return err
	}

	if !result.Matched {
		if result.Score > 0 && minScore > 0 {
			fmt.Printf("no matching track found (best score %d below min-score %d)\n", result.Score, minScore)
			return nil
		}

		fmt.Println("no matching track found")
		return nil
	}

	if result.EarlyStopped {
		fmt.Printf("best match: track_id=%d title=%q artist=%q path=%s [early stopped] offset_ms=%d\n", result.TrackID, result.Title, result.Artist, result.Path, result.OffsetMS)
		return nil
	}

	fmt.Printf("best match: track_id=%d title=%q artist=%q path=%s score=%d offset_ms=%d\n", result.TrackID, result.Title, result.Artist, result.Path, result.Score, result.OffsetMS)
	return nil
}

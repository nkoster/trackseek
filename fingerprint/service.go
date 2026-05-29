package fingerprint

import (
	"database/sql"
	"errors"
	"math"

	"trackseek/models"

	"gonum.org/v1/gonum/dsp/fourier"
)

const (
	windowSize    = 4096
	hopSize       = 2048
	maxPeaksFrame = 5
	fanout        = 5
)

var ErrNoMatch = errors.New("no matching track found")

type Fingerprint struct {
	Hash   int64
	TimeMS int
}

type MatchResult struct {
	TrackID      int64
	Score        int
	OffsetMS     int
	EarlyStopped bool
}

func ExtractPeaks(samples []float64) []models.Peak {
	fft := fourier.NewFFT(windowSize)

	var peaks []models.Peak

	for start := 0; start+windowSize <= len(samples); start += hopSize {
		frame := make([]float64, windowSize)

		for i := 0; i < windowSize; i++ {
			w := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(windowSize-1)))
			frame[i] = samples[start+i] * w
		}

		coeffs := fft.Coefficients(nil, frame)

		framePeaks := topPeaks(coeffs, start/hopSize, maxPeaksFrame)
		peaks = append(peaks, framePeaks...)
	}

	return peaks
}

func StoreFingerprints(db *sql.DB, trackID int64, sampleRate int, peaks []models.Peak) (int, error) {
	fingerprints := fingerprintsFromPeaks(sampleRate, peaks)

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO fingerprints(hash, track_id, time_ms)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for _, fingerprint := range fingerprints {
		if _, err := stmt.Exec(fingerprint.Hash, trackID, fingerprint.TimeMS); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return len(fingerprints), nil
}

func MatchFingerprints(db *sql.DB, sampleRate int, peaks []models.Peak, threshold int) (*MatchResult, error) {
	fingerprints := fingerprintsFromPeaks(sampleRate, peaks)
	if len(fingerprints) == 0 {
		return nil, ErrNoMatch
	}

	stmt, err := db.Prepare(`
		SELECT track_id, time_ms
		FROM fingerprints
		WHERE hash = ?
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	scores := make(map[int64]map[int]int)
	best := MatchResult{}
	found := false

	for _, fingerprint := range fingerprints {
		rows, err := stmt.Query(fingerprint.Hash)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var trackID int64
			var dbTimeMS int

			if err := rows.Scan(&trackID, &dbTimeMS); err != nil {
				rows.Close()
				return nil, err
			}

			offsetMS := dbTimeMS - fingerprint.TimeMS
			if _, ok := scores[trackID]; !ok {
				scores[trackID] = make(map[int]int)
			}

			scores[trackID][offsetMS]++
			score := scores[trackID][offsetMS]

			if !found || score > best.Score {
				best = MatchResult{TrackID: trackID, Score: score, OffsetMS: offsetMS}
				found = true

				if threshold > 0 && score >= threshold {
					best.EarlyStopped = true
					rows.Close()
					return &best, nil
				}
			}
		}

		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}

		rows.Close()
	}

	if !found {
		return nil, ErrNoMatch
	}

	return &best, nil
}

func topPeaks(coeffs []complex128, timeFrame int, maxPeaks int) []models.Peak {
	var peaks []models.Peak

	limit := len(coeffs) / 2

	for bin := 5; bin < limit; bin++ {
		mag := cmplxAbs(coeffs[bin])

		if bin > 0 && bin < limit-1 {
			left := cmplxAbs(coeffs[bin-1])
			right := cmplxAbs(coeffs[bin+1])

			if mag <= left || mag <= right {
				continue
			}
		}

		peaks = append(peaks, models.Peak{
			TimeFrame: timeFrame,
			FreqBin:   bin,
			Magnitude: mag,
		})
	}

	for i := 0; i < len(peaks); i++ {
		for j := i + 1; j < len(peaks); j++ {
			if peaks[j].Magnitude > peaks[i].Magnitude {
				peaks[i], peaks[j] = peaks[j], peaks[i]
			}
		}
	}

	if len(peaks) > maxPeaks {
		peaks = peaks[:maxPeaks]
	}

	return peaks
}

func cmplxAbs(c complex128) float64 {
	return math.Sqrt(real(c)*real(c) + imag(c)*imag(c))
}

func fingerprintsFromPeaks(sampleRate int, peaks []models.Peak) []Fingerprint {
	fingerprints := make([]Fingerprint, 0, len(peaks)*fanout)

	for i := 0; i < len(peaks); i++ {
		anchor := peaks[i]

		for j := i + 1; j < len(peaks) && j <= i+fanout; j++ {
			target := peaks[j]

			delta := target.TimeFrame - anchor.TimeFrame
			if delta <= 0 {
				continue
			}

			fingerprints = append(fingerprints, Fingerprint{
				Hash:   makeHash(anchor.FreqBin, target.FreqBin, delta),
				TimeMS: frameToMs(anchor.TimeFrame, sampleRate),
			})
		}
	}

	return fingerprints
}

func makeHash(freq1, freq2, deltaFrames int) int64 {
	return int64((freq1&0xFFFF)<<32 | (freq2&0xFFFF)<<16 | (deltaFrames & 0xFFFF))
}

func frameToMs(frame int, sampleRate int) int {
	sampleIndex := frame * hopSize
	return int(float64(sampleIndex) / float64(sampleRate) * 1000.0)
}

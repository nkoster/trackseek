package fingerprint

import (
	"database/sql"
	"errors"
	"math"

	"trackseek/audio"
	"trackseek/models"

	"gonum.org/v1/gonum/dsp/fourier"
)

const (
	windowSize     = 4096
	hopSize        = 2048
	maxPeaksFrame  = 5
	fanout         = 5
	offsetBucketMS = 100
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

type AudioMatch struct {
	Matched      bool
	TrackID      int64
	Title        string
	Artist       string
	Path         string
	Score        int
	OffsetMS     int
	EarlyStopped bool
}

type IndexedHit struct {
	TrackID int64
	TimeMS  int
}

type IndexedTrack struct {
	ID     int64
	Title  string
	Artist string
	Path   string
}

type InMemoryIndex struct {
	HitsByHash map[int64][]IndexedHit
	Tracks     map[int64]IndexedTrack
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
			offsetBucket := bucketOffsetMS(offsetMS)
			if _, ok := scores[trackID]; !ok {
				scores[trackID] = make(map[int]int)
			}

			scores[trackID][offsetBucket]++
			score := scores[trackID][offsetBucket]

			if !found || score > best.Score {
				best = MatchResult{TrackID: trackID, Score: score, OffsetMS: offsetBucket * offsetBucketMS}
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

func MatchAudioFile(db *sql.DB, audioPath string, minScore int, threshold int) (*AudioMatch, error) {
	samples, sampleRate, err := audio.ReadMono(audioPath)
	if err != nil {
		return nil, err
	}

	peaks := ExtractPeaks(samples)
	result, err := MatchFingerprints(db, sampleRate, peaks, threshold)
	if err != nil {
		if errors.Is(err, ErrNoMatch) {
			return &AudioMatch{Matched: false}, nil
		}

		return nil, err
	}

	if result.Score < minScore {
		return &AudioMatch{Matched: false, Score: result.Score, OffsetMS: result.OffsetMS}, nil
	}

	track, err := models.GetTrackByID(result.TrackID)
	if err != nil {
		return nil, err
	}

	artistName := ""
	if track.Artist != nil {
		artistName = track.Artist.Name
	}

	return &AudioMatch{
		Matched:      true,
		TrackID:      result.TrackID,
		Title:        track.Title,
		Artist:       artistName,
		Path:         track.Path,
		Score:        result.Score,
		OffsetMS:     result.OffsetMS,
		EarlyStopped: result.EarlyStopped,
	}, nil
}

func BuildInMemoryIndex(db *sql.DB) (*InMemoryIndex, error) {
	rows, err := db.Query(`
		SELECT hash, track_id, time_ms
		FROM fingerprints
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	index := &InMemoryIndex{
		HitsByHash: make(map[int64][]IndexedHit),
		Tracks:     make(map[int64]IndexedTrack),
	}

	for rows.Next() {
		var hash int64
		var trackID int64
		var timeMS int

		if err := rows.Scan(&hash, &trackID, &timeMS); err != nil {
			return nil, err
		}

		index.HitsByHash[hash] = append(index.HitsByHash[hash], IndexedHit{TrackID: trackID, TimeMS: timeMS})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	trackRows, err := db.Query(`
		SELECT tracks.id, tracks.title, artists.name, tracks.path
		FROM tracks
		JOIN artists ON artists.id = tracks.artist_id
	`)
	if err != nil {
		return nil, err
	}
	defer trackRows.Close()

	for trackRows.Next() {
		var track IndexedTrack

		if err := trackRows.Scan(&track.ID, &track.Title, &track.Artist, &track.Path); err != nil {
			return nil, err
		}

		index.Tracks[track.ID] = track
	}

	if err := trackRows.Err(); err != nil {
		return nil, err
	}

	return index, nil
}

func MatchFingerprintsInMemory(index *InMemoryIndex, sampleRate int, peaks []models.Peak, threshold int) (*MatchResult, error) {
	fingerprints := fingerprintsFromPeaks(sampleRate, peaks)
	if len(fingerprints) == 0 {
		return nil, ErrNoMatch
	}

	scores := make(map[int64]map[int]int)
	best := MatchResult{}
	found := false

	for _, fingerprint := range fingerprints {
		hits := index.HitsByHash[fingerprint.Hash]
		for _, hit := range hits {
			offsetMS := hit.TimeMS - fingerprint.TimeMS
			offsetBucket := bucketOffsetMS(offsetMS)
			if _, ok := scores[hit.TrackID]; !ok {
				scores[hit.TrackID] = make(map[int]int)
			}

			scores[hit.TrackID][offsetBucket]++
			score := scores[hit.TrackID][offsetBucket]

			if !found || score > best.Score {
				best = MatchResult{TrackID: hit.TrackID, Score: score, OffsetMS: offsetBucket * offsetBucketMS}
				found = true

				if threshold > 0 && score >= threshold {
					best.EarlyStopped = true
					return &best, nil
				}
			}
		}
	}

	if !found {
		return nil, ErrNoMatch
	}

	return &best, nil
}

func MatchAudioFileInMemory(index *InMemoryIndex, audioPath string, minScore int, threshold int) (*AudioMatch, error) {
	samples, sampleRate, err := audio.ReadMono(audioPath)
	if err != nil {
		return nil, err
	}

	peaks := ExtractPeaks(samples)
	result, err := MatchFingerprintsInMemory(index, sampleRate, peaks, threshold)
	if err != nil {
		if errors.Is(err, ErrNoMatch) {
			return &AudioMatch{Matched: false}, nil
		}

		return nil, err
	}

	if result.Score < minScore {
		return &AudioMatch{Matched: false, Score: result.Score, OffsetMS: result.OffsetMS}, nil
	}

	track, ok := index.Tracks[result.TrackID]
	if !ok {
		return nil, sql.ErrNoRows
	}

	return &AudioMatch{
		Matched:      true,
		TrackID:      result.TrackID,
		Title:        track.Title,
		Artist:       track.Artist,
		Path:         track.Path,
		Score:        result.Score,
		OffsetMS:     result.OffsetMS,
		EarlyStopped: result.EarlyStopped,
	}, nil
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

func bucketOffsetMS(offsetMS int) int {
	if offsetMS >= 0 {
		return offsetMS / offsetBucketMS
	}

	return -((-offsetMS + offsetBucketMS - 1) / offsetBucketMS)
}

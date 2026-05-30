package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"trackseek/db"
	"trackseek/fingerprint"
)

const maxUploadSize = 32 << 20

type matchResponse struct {
	Matched      bool   `json:"matched"`
	TrackID      int64  `json:"trackId,omitempty"`
	Title        string `json:"title,omitempty"`
	Artist       string `json:"artist,omitempty"`
	Path         string `json:"path,omitempty"`
	Score        int    `json:"score,omitempty"`
	OffsetMS     int    `json:"offsetMs,omitempty"`
	EarlyStopped bool   `json:"earlyStopped,omitempty"`
	Error        string `json:"error,omitempty"`
}

func matchSample(index *fingerprint.InMemoryIndex) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		response, statusCode := buildMatchResponse(r, index)
		writeSSE(w, flusher, "match", response, statusCode)
	}
}

func buildMatchResponse(r *http.Request, index *fingerprint.InMemoryIndex) (matchResponse, int) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		return matchResponse{Error: "invalid multipart form upload"}, http.StatusBadRequest
	}

	file, header, err := r.FormFile("sample")
	if err != nil {
		return matchResponse{Error: "missing form file field 'sample'"}, http.StatusBadRequest
	}
	defer file.Close()

	tempPath, err := saveUpload(file, header.Filename)
	if err != nil {
		return matchResponse{Error: "failed to store uploaded sample"}, http.StatusInternalServerError
	}
	defer os.Remove(tempPath)

	minScore, err := parseOptionalInt(r.FormValue("minScore"))
	if err != nil {
		return matchResponse{Error: "invalid minScore"}, http.StatusBadRequest
	}

	threshold, err := parseOptionalInt(r.FormValue("threshold"))
	if err != nil {
		return matchResponse{Error: "invalid threshold"}, http.StatusBadRequest
	}

	var result *fingerprint.AudioMatch
	if index != nil {
		result, err = fingerprint.MatchAudioFileInMemory(index, tempPath, minScore, threshold)
	} else {
		result, err = fingerprint.MatchAudioFile(db.DB, tempPath, minScore, threshold)
	}
	if err != nil {
		return matchResponse{Error: err.Error()}, http.StatusInternalServerError
	}

	return matchResponse{
		Matched:      result.Matched,
		TrackID:      result.TrackID,
		Title:        result.Title,
		Artist:       result.Artist,
		Path:         result.Path,
		Score:        result.Score,
		OffsetMS:     result.OffsetMS,
		EarlyStopped: result.EarlyStopped,
	}, http.StatusOK
}

func saveUpload(file io.Reader, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}

	tempFile, err := os.CreateTemp("", "trackseek-upload-*"+ext)
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func parseOptionalInt(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}

	return strconv.Atoi(raw)
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload any, statusCode int) {
	w.WriteHeader(statusCode)

	encoded, err := json.Marshal(payload)
	if err != nil {
		encoded = []byte(`{"error":"failed to encode response"}`)
	}

	_, _ = fmt.Fprintf(w, "event: %s\n", event)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", encoded)
	flusher.Flush()
}

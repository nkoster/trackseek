package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-audio/wav"
	"github.com/hajimehoshi/go-mp3"
)

func ReadMono(path string) ([]float64, int, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".wav":
		return readWAVMono(path)
	case ".mp3":
		return readMP3Mono(path)
	default:
		return nil, 0, fmt.Errorf("unsupported audio format: %s", ext)
	}
}

func readWAVMono(path string) ([]float64, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	decoder := wav.NewDecoder(f)
	if !decoder.IsValidFile() {
		return nil, 0, fmt.Errorf("invalid wav file")
	}

	buf, err := decoder.FullPCMBuffer()
	if err != nil && err != io.EOF {
		return nil, 0, err
	}

	if buf == nil || buf.Format == nil {
		return nil, 0, fmt.Errorf("empty audio buffer")
	}

	channels := buf.Format.NumChannels
	sampleRate := buf.Format.SampleRate

	if channels < 1 {
		return nil, 0, fmt.Errorf("invalid channel count")
	}

	data := buf.Data
	samples := make([]float64, 0, len(data)/channels)

	for i := 0; i+channels <= len(data); i += channels {
		var sum int
		for c := 0; c < channels; c++ {
			sum += data[i+c]
		}

		avg := float64(sum) / float64(channels)
		samples = append(samples, avg/32768.0)
	}

	return samples, sampleRate, nil
}

func readMP3Mono(path string) ([]float64, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	decoder, err := mp3.NewDecoder(f)
	if err != nil {
		return nil, 0, err
	}

	raw, err := io.ReadAll(decoder)
	if err != nil {
		return nil, 0, err
	}

	if len(raw) == 0 {
		return nil, 0, fmt.Errorf("empty audio buffer")
	}

	const bytesPerSample = 2
	const channels = 2
	frameSize := bytesPerSample * channels
	if len(raw) < frameSize {
		return nil, 0, fmt.Errorf("invalid mp3 audio buffer")
	}

	samples := make([]float64, 0, len(raw)/frameSize)
	for i := 0; i+frameSize <= len(raw); i += frameSize {
		left := int16(binary.LittleEndian.Uint16(raw[i : i+2]))
		right := int16(binary.LittleEndian.Uint16(raw[i+2 : i+4]))
		avg := (float64(left) + float64(right)) / 2.0
		samples = append(samples, avg/32768.0)
	}

	return samples, decoder.SampleRate(), nil
}

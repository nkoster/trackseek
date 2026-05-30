package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-audio/wav"
	"github.com/hajimehoshi/go-mp3"
)

func ReadMono(path string) ([]float64, int, error) {
	ext := strings.ToLower(filepath.Ext(path))
	targetSampleRate, err := getTargetSampleRate()
	if err != nil {
		return nil, 0, err
	}

	var (
		samples    []float64
		sampleRate int
		readErr    error
	)

	switch ext {
	case ".wav":
		samples, sampleRate, readErr = readWAVMono(path)
	case ".mp3":
		samples, sampleRate, readErr = readMP3Mono(path)
	default:
		return nil, 0, fmt.Errorf("unsupported audio format: %s", ext)
	}

	if readErr != nil {
		return nil, 0, readErr
	}

	resampled, err := resampleMono(samples, sampleRate, targetSampleRate)
	if err != nil {
		return nil, 0, err
	}

	return resampled, targetSampleRate, nil
}

func getTargetSampleRate() (int, error) {
	value := strings.TrimSpace(os.Getenv("TRACKSEEK_TARGET_SAMPLE_RATE"))
	if value == "" {
		return defaultTargetSampleRate, nil
	}

	rate, err := strconv.Atoi(value)
	if err != nil || rate <= 0 {
		return 0, fmt.Errorf("invalid TRACKSEEK_TARGET_SAMPLE_RATE: %q", value)
	}

	return rate, nil
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
		samples = append(samples, avg/pcm16FullScale)
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

	frameSize := mp3DecoderBytesPerSample * mp3DecoderChannels
	if len(raw) < frameSize {
		return nil, 0, fmt.Errorf("invalid mp3 audio buffer")
	}

	samples := make([]float64, 0, len(raw)/frameSize)
	for i := 0; i+frameSize <= len(raw); i += frameSize {
		left := int16(binary.LittleEndian.Uint16(raw[i : i+2]))
		right := int16(binary.LittleEndian.Uint16(raw[i+2 : i+4]))
		avg := (float64(left) + float64(right)) / 2.0
		samples = append(samples, avg/pcm16FullScale)
	}

	return samples, decoder.SampleRate(), nil
}

func resampleMono(samples []float64, inputRate int, outputRate int) ([]float64, error) {
	if inputRate <= 0 || outputRate <= 0 {
		return nil, fmt.Errorf("invalid sample rate")
	}

	if len(samples) == 0 {
		return nil, fmt.Errorf("empty audio buffer")
	}

	if inputRate == outputRate {
		copied := append([]float64(nil), samples...)
		return copied, nil
	}

	if outputRate < inputRate {
		cutoffHz := lowPassCutoffRatio * float64(outputRate)
		samples = lowPassFilter(samples, inputRate, cutoffHz, lowPassFilterTaps)
	}

	outputLen := int(math.Round(float64(len(samples)) * float64(outputRate) / float64(inputRate)))
	if outputLen < 1 {
		outputLen = 1
	}

	resampled := make([]float64, outputLen)
	if outputLen == 1 {
		resampled[0] = samples[0]
		return resampled, nil
	}

	last := len(samples) - 1
	for i := 0; i < outputLen; i++ {
		position := float64(i) * float64(inputRate) / float64(outputRate)
		left := int(math.Floor(position))
		if left < 0 {
			left = 0
		}
		if left >= last {
			resampled[i] = samples[last]
			continue
		}

		right := left + 1
		frac := position - float64(left)
		resampled[i] = samples[left]*(1-frac) + samples[right]*frac
	}

	return resampled, nil
}

func lowPassFilter(samples []float64, sampleRate int, cutoffHz float64, taps int) []float64 {
	if len(samples) == 0 || sampleRate <= 0 || cutoffHz <= 0 {
		return append([]float64(nil), samples...)
	}

	if cutoffHz >= float64(sampleRate)/2 {
		return append([]float64(nil), samples...)
	}

	if taps < 3 {
		taps = 3
	}

	if taps%2 == 0 {
		taps++
	}

	kernel := make([]float64, taps)
	mid := taps / 2
	normalizedCutoff := cutoffHz / float64(sampleRate)
	var sum float64

	for i := 0; i < taps; i++ {
		n := float64(i - mid)
		var value float64
		if n == 0 {
			value = 2 * normalizedCutoff
		} else {
			value = math.Sin(2*math.Pi*normalizedCutoff*n) / (math.Pi * n)
		}

		window := hammingWindowAlpha - hammingWindowBeta*math.Cos(2*math.Pi*float64(i)/float64(taps-1))
		kernel[i] = value * window
		sum += kernel[i]
	}

	if sum == 0 {
		return append([]float64(nil), samples...)
	}

	for i := range kernel {
		kernel[i] /= sum
	}

	filtered := make([]float64, len(samples))
	last := len(samples) - 1
	for i := range samples {
		var acc float64
		for k := 0; k < taps; k++ {
			index := i + k - mid
			if index < 0 {
				index = 0
			} else if index > last {
				index = last
			}

			acc += samples[index] * kernel[k]
		}

		filtered[i] = acc
	}

	return filtered
}

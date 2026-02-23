package audio

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

const (
	SampleRate    = 24000
	NumChannels   = 1
	BitsPerSample = 16
)

type Audio struct {
	Samples    []float32
	SampleRate int
}

func NewAudio(samples []float32) *Audio {
	return &Audio{
		Samples:    samples,
		SampleRate: SampleRate,
	}
}

func (a *Audio) SaveWAV(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	numSamples := len(a.Samples)
	dataSize := numSamples * NumChannels * (BitsPerSample / 8)
	fileSize := 36 + dataSize

	if _, err := f.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(fileSize)); err != nil {
		return err
	}
	if _, err := f.Write([]byte("WAVE")); err != nil {
		return err
	}

	if _, err := f.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(NumChannels)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(a.SampleRate)); err != nil {
		return err
	}
	byteRate := a.SampleRate * NumChannels * (BitsPerSample / 8)
	if err := binary.Write(f, binary.LittleEndian, uint32(byteRate)); err != nil {
		return err
	}
	blockAlign := NumChannels * (BitsPerSample / 8)
	if err := binary.Write(f, binary.LittleEndian, uint16(blockAlign)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(BitsPerSample)); err != nil {
		return err
	}

	if _, err := f.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(dataSize)); err != nil {
		return err
	}

	for _, sample := range a.Samples {
		clamped := sample
		if clamped > 1.0 {
			clamped = 1.0
		} else if clamped < -1.0 {
			clamped = -1.0
		}

		intSample := int16(clamped * math.MaxInt16)
		if err := binary.Write(f, binary.LittleEndian, intSample); err != nil {
			return err
		}
	}

	return nil
}

func (a *Audio) Duration() float64 {
	return float64(len(a.Samples)) / float64(a.SampleRate)
}

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

func NewAudioWithSampleRate(samples []float32, sampleRate int) *Audio {
	return &Audio{
		Samples:    samples,
		SampleRate: sampleRate,
	}
}

func LoadWAV(path string) (*Audio, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	header := make([]byte, 44)
	if _, err := f.Read(header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return nil, fmt.Errorf("invalid WAV file format")
	}

	numChannels := int(binary.LittleEndian.Uint16(header[22:24]))
	sampleRate := int(binary.LittleEndian.Uint32(header[24:28]))
	bitsPerSample := int(binary.LittleEndian.Uint16(header[34:36]))

	var dataSize uint32
	pos := 36
	for {
		chunkHeader := make([]byte, 8)
		if _, err := f.Read(chunkHeader); err != nil {
			return nil, fmt.Errorf("failed to find data chunk: %w", err)
		}
		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		if chunkID == "data" {
			dataSize = chunkSize
			break
		}

		if _, err := f.Seek(int64(chunkSize), 1); err != nil {
			return nil, fmt.Errorf("failed to skip chunk: %w", err)
		}
		pos += 8 + int(chunkSize)
	}

	data := make([]byte, dataSize)
	if _, err := f.Read(data); err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	var samples []float32

	switch bitsPerSample {
	case 16:
		numSamples := len(data) / 2
		samples = make([]float32, numSamples/numChannels)
		for i := 0; i < numSamples; i += numChannels {
			sample := int16(binary.LittleEndian.Uint16(data[i*2 : (i+1)*2]))
			samples[i/numChannels] = float32(sample) / 32768.0
		}
	case 32:
		numSamples := len(data) / 4
		samples = make([]float32, numSamples/numChannels)
		for i := 0; i < numSamples; i += numChannels {
			bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
			samples[i/numChannels] = math.Float32frombits(bits)
		}
	default:
		return nil, fmt.Errorf("unsupported bits per sample: %d", bitsPerSample)
	}

	return &Audio{
		Samples:    samples,
		SampleRate: sampleRate,
	}, nil
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

package voice

import (
	"archive/zip"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const expectedEmbeddingDim = 256

type VoiceStore struct {
	voices       map[string][]float32
	embeddingDim int
}

func LoadVoices(path string) (*VoiceStore, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open NPZ file: %w", err)
	}
	defer r.Close()

	store := &VoiceStore{
		voices:       make(map[string][]float32),
		embeddingDim: expectedEmbeddingDim,
	}

	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, ".npy") {
			continue
		}

		name := strings.TrimSuffix(filepath.Base(f.Name), ".npy")

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", f.Name, err)
		}

		data, shape, err := readNpyFloat32WithShape(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", f.Name, err)
		}

		if len(shape) == 2 && shape[1] == expectedEmbeddingDim {
			store.voices[name] = data[:expectedEmbeddingDim]
		} else if len(data) == expectedEmbeddingDim {
			store.voices[name] = data
		} else if len(data) > expectedEmbeddingDim {
			store.voices[name] = data[:expectedEmbeddingDim]
		} else {
			store.voices[name] = data
		}
	}

	return store, nil
}

func (v *VoiceStore) Get(name string) ([]float32, error) {
	voice, ok := v.voices[name]
	if !ok {
		return nil, fmt.Errorf("voice not found: %s", name)
	}
	return voice, nil
}

func (v *VoiceStore) List() []string {
	names := make([]string, 0, len(v.voices))
	for name := range v.voices {
		names = append(names, name)
	}
	return names
}

func (v *VoiceStore) EmbeddingDim() int {
	return v.embeddingDim
}

func readNpyFloat32WithShape(r io.Reader) ([]float32, []int, error) {
	magic := make([]byte, 6)
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, nil, fmt.Errorf("failed to read magic: %w", err)
	}
	if string(magic) != "\x93NUMPY" {
		return nil, nil, fmt.Errorf("invalid NPY magic number")
	}

	version := make([]byte, 2)
	if _, err := io.ReadFull(r, version); err != nil {
		return nil, nil, fmt.Errorf("failed to read version: %w", err)
	}

	var headerLen uint32
	if version[0] == 1 {
		var hl uint16
		if err := binary.Read(r, binary.LittleEndian, &hl); err != nil {
			return nil, nil, fmt.Errorf("failed to read header length: %w", err)
		}
		headerLen = uint32(hl)
	} else {
		if err := binary.Read(r, binary.LittleEndian, &headerLen); err != nil {
			return nil, nil, fmt.Errorf("failed to read header length: %w", err)
		}
	}

	header := make([]byte, headerLen)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, nil, fmt.Errorf("failed to read header: %w", err)
	}

	headerStr := string(header)
	shape, err := parseShape(headerStr)
	if err != nil {
		return nil, nil, err
	}

	totalElements := 1
	for _, dim := range shape {
		totalElements *= dim
	}

	isFloat16 := strings.Contains(headerStr, "'<f2'") || strings.Contains(headerStr, "descr': '<f2")
	isFloat32 := strings.Contains(headerStr, "'<f4'") || strings.Contains(headerStr, "descr': '<f4")

	if isFloat16 {
		data := make([]uint16, totalElements)
		if err := binary.Read(r, binary.LittleEndian, data); err != nil {
			return nil, nil, fmt.Errorf("failed to read float16 data: %w", err)
		}
		result := make([]float32, totalElements)
		for i, v := range data {
			result[i] = float16ToFloat32(v)
		}
		return result, shape, nil
	} else if isFloat32 {
		result := make([]float32, totalElements)
		if err := binary.Read(r, binary.LittleEndian, result); err != nil {
			return nil, nil, fmt.Errorf("failed to read float32 data: %w", err)
		}
		return result, shape, nil
	}

	return nil, nil, fmt.Errorf("unsupported dtype in header: %s", headerStr)
}

func parseShape(header string) ([]int, error) {
	start := strings.Index(header, "'shape': (")
	if start == -1 {
		start = strings.Index(header, "\"shape\": (")
	}
	if start == -1 {
		return nil, fmt.Errorf("shape not found in header")
	}

	start += 10
	end := strings.Index(header[start:], ")")
	if end == -1 {
		return nil, fmt.Errorf("invalid shape format")
	}

	shapeStr := strings.TrimSpace(header[start : start+end])
	if shapeStr == "" {
		return []int{1}, nil
	}

	shapeStr = strings.TrimSuffix(shapeStr, ",")
	parts := strings.Split(shapeStr, ",")

	shape := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var dim int
		if _, err := fmt.Sscanf(p, "%d", &dim); err != nil {
			return nil, fmt.Errorf("invalid dimension: %s", p)
		}
		shape = append(shape, dim)
	}

	if len(shape) == 0 {
		return []int{1}, nil
	}

	return shape, nil
}

func float16ToFloat32(h uint16) float32 {
	sign := uint32((h >> 15) & 1)
	exp := uint32((h >> 10) & 0x1F)
	mant := uint32(h & 0x3FF)

	var f uint32
	if exp == 0 {
		if mant == 0 {
			f = sign << 31
		} else {
			for (mant & 0x400) == 0 {
				mant <<= 1
				exp--
			}
			exp++
			mant &= 0x3FF
			f = (sign << 31) | ((exp + 127 - 15) << 23) | (mant << 13)
		}
	} else if exp == 31 {
		if mant == 0 {
			f = (sign << 31) | 0x7F800000
		} else {
			f = (sign << 31) | 0x7FC00000 | (mant << 13)
		}
	} else {
		f = (sign << 31) | ((exp + 127 - 15) << 23) | (mant << 13)
	}

	return math.Float32frombits(f)
}

func LoadVoicesFromDir(dir string) (*VoiceStore, error) {
	store := &VoiceStore{
		voices: make(map[string][]float32),
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".npy") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".npy")
		path := filepath.Join(dir, entry.Name())

		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", path, err)
		}

		data, shape, err := readNpyFloat32WithShape(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		if len(shape) == 2 && shape[1] == expectedEmbeddingDim {
			store.voices[name] = data[:expectedEmbeddingDim]
		} else if len(data) == expectedEmbeddingDim {
			store.voices[name] = data
		} else if len(data) > expectedEmbeddingDim {
			store.voices[name] = data[:expectedEmbeddingDim]
		} else {
			store.voices[name] = data
		}
	}

	return store, nil
}

package pockettts

import (
	"fmt"
	"path/filepath"
	"strings"

	"tts2go/internal/pkg/tts2go/audio"
	"tts2go/internal/pkg/tts2go/engine"
)

const (
	pocketSampleRate = 24000
)

func init() {
	engine.Register("pocket", NewEngine)
}

type Engine struct {
	pipeline   *Pipeline
	tokenizer  *Tokenizer
	modelDir   string
	refEmbeds  map[string][]float32
	defaultRef []float32
}

func NewEngine(cfg engine.EngineConfig) (engine.Engine, error) {
	modelDir := cfg.ModelPath
	if strings.HasSuffix(modelDir, ".onnx") {
		modelDir = filepath.Dir(modelDir)
	}

	vocabPath := filepath.Join(modelDir, "vocab.json")
	scoresPath := filepath.Join(modelDir, "token_scores.json")

	tokenizer, err := NewTokenizer(vocabPath, scoresPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokenizer: %w", err)
	}

	useInt8 := cfg.ModelVariant == "int8"
	pipeline, err := NewPipeline(modelDir, useInt8)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	return &Engine{
		pipeline:   pipeline,
		tokenizer:  tokenizer,
		modelDir:   modelDir,
		refEmbeds:  make(map[string][]float32),
		defaultRef: nil,
	}, nil
}

func (e *Engine) Generate(text, voice string, speed float32) (*audio.Audio, error) {
	var speakerEmbeds []float32

	if voice != "" {
		if embeds, ok := e.refEmbeds[voice]; ok {
			speakerEmbeds = embeds
		}
	}

	if speakerEmbeds == nil && e.defaultRef != nil {
		speakerEmbeds = e.defaultRef
	}

	if speakerEmbeds == nil {
		speakerEmbeds = make([]float32, seqFeatureDim)
		for i := range speakerEmbeds {
			speakerEmbeds[i] = float32(randNormal()) * 0.1
		}
	}

	tokens := e.tokenizer.Encode(text)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("failed to tokenize text")
	}

	textEmbeds, err := e.pipeline.GetTextEmbeddings(tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to get text embeddings: %w", err)
	}

	audioData, err := e.pipeline.Generate(textEmbeds, speakerEmbeds, speed)
	if err != nil {
		return nil, fmt.Errorf("failed to generate audio: %w", err)
	}

	return audio.NewAudioWithSampleRate(audioData, pocketSampleRate), nil
}

func (e *Engine) GenerateWithReference(text string, refAudio *audio.Audio, speed float32) (*audio.Audio, error) {
	if refAudio == nil {
		return e.Generate(text, "", speed)
	}

	speakerEmbeds, err := e.pipeline.EncodeReference(refAudio.Samples)
	if err != nil {
		return nil, fmt.Errorf("failed to encode reference audio: %w", err)
	}

	tokens := e.tokenizer.Encode(text)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("failed to tokenize text")
	}

	textEmbeds, err := e.pipeline.GetTextEmbeddings(tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to get text embeddings: %w", err)
	}

	audioData, err := e.pipeline.Generate(textEmbeds, speakerEmbeds, speed)
	if err != nil {
		return nil, fmt.Errorf("failed to generate audio: %w", err)
	}

	return audio.NewAudioWithSampleRate(audioData, pocketSampleRate), nil
}

func (e *Engine) RegisterVoice(name string, refAudio *audio.Audio) error {
	if refAudio == nil {
		return fmt.Errorf("reference audio is nil")
	}

	audioEmbeds, err := e.pipeline.EncodeReference(refAudio.Samples)
	if err != nil {
		return fmt.Errorf("failed to encode reference audio: %w", err)
	}

	e.refEmbeds[name] = audioEmbeds
	return nil
}

func (e *Engine) SetDefaultReference(refAudio *audio.Audio) error {
	if refAudio == nil {
		e.defaultRef = nil
		return nil
	}

	audioEmbeds, err := e.pipeline.EncodeReference(refAudio.Samples)
	if err != nil {
		return fmt.Errorf("failed to encode reference audio: %w", err)
	}

	e.defaultRef = audioEmbeds
	return nil
}

func (e *Engine) ListVoices() []string {
	voices := make([]string, 0, len(e.refEmbeds))
	for name := range e.refEmbeds {
		voices = append(voices, name)
	}
	return voices
}

func (e *Engine) Info() engine.EngineInfo {
	return engine.EngineInfo{
		Name:       "pocket",
		Languages:  []string{"en", "zh", "multilingual"},
		SampleRate: pocketSampleRate,
	}
}

func (e *Engine) Close() error {
	return e.pipeline.Close()
}

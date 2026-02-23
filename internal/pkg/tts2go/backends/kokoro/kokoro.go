package kokoro

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	ort "github.com/yalue/onnxruntime_go"

	"tts2go/internal/pkg/tts2go/audio"
	"tts2go/internal/pkg/tts2go/engine"
	"tts2go/internal/pkg/tts2go/phonemizer"
	"tts2go/internal/pkg/tts2go/preprocess"
	"tts2go/internal/pkg/tts2go/tokenizer"
	"tts2go/internal/pkg/tts2go/voice"
)

func init() {
	engine.Register("kokoro", NewEngine)
}

type Engine struct {
	session      *ort.DynamicAdvancedSession
	voices       *voice.VoiceStore
	preprocessor *preprocess.Preprocessor
	phonemizer   *phonemizer.Phonemizer
	tokenizer    *tokenizer.Tokenizer
}

func getOnnxRuntimeLibPath() string {
	envPath := os.Getenv("ONNXRUNTIME_LIB_PATH")
	if envPath != "" {
		return envPath
	}

	switch runtime.GOOS {
	case "linux":
		paths := []string{
			"/usr/lib/libonnxruntime.so",
			"/usr/local/lib/libonnxruntime.so",
			"./libonnxruntime.so",
			"./lib/libonnxruntime.so",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return "libonnxruntime.so"
	case "windows":
		paths := []string{
			"onnxruntime.dll",
			"./onnxruntime.dll",
			"./lib/onnxruntime.dll",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return "onnxruntime.dll"
	case "darwin":
		paths := []string{
			"/usr/local/lib/libonnxruntime.dylib",
			"/opt/homebrew/lib/libonnxruntime.dylib",
			"./libonnxruntime.dylib",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return "libonnxruntime.dylib"
	default:
		return "libonnxruntime.so"
	}
}

func NewEngine(cfg engine.EngineConfig) (engine.Engine, error) {
	libPath := getOnnxRuntimeLibPath()
	ort.SetSharedLibraryPath(libPath)

	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX runtime: %w", err)
	}

	voices, err := voice.LoadVoices(cfg.VoicesPath)
	if err != nil {
		voicesDir := filepath.Dir(cfg.VoicesPath)
		voices, err = voice.LoadVoicesFromDir(filepath.Join(voicesDir, "voices"))
		if err != nil {
			return nil, fmt.Errorf("failed to load voices: %w", err)
		}
	}

	inputNames := []string{"input_ids", "style", "speed"}
	outputNames := []string{"waveform"}

	session, err := ort.NewDynamicAdvancedSession(
		cfg.ModelPath,
		inputNames,
		outputNames,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ONNX session: %w", err)
	}

	return &Engine{
		session:      session,
		voices:       voices,
		preprocessor: preprocess.NewPreprocessor(),
		phonemizer:   phonemizer.NewPhonemizer(),
		tokenizer:    tokenizer.NewTokenizer(),
	}, nil
}

func (e *Engine) Generate(text, voiceName string, speed float32) (*audio.Audio, error) {
	processedText := e.preprocessor.Process(text)
	phonemes := e.phonemizer.Phonemize(processedText)
	tokens := e.tokenizer.Encode(phonemes)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("failed to tokenize text")
	}

	voiceEmbedding, err := e.voices.Get(voiceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get voice embedding: %w", err)
	}

	inputIdsTensor, err := ort.NewTensor(ort.NewShape(1, int64(len(tokens))), tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputIdsTensor.Destroy()

	styleTensor, err := ort.NewTensor(ort.NewShape(1, int64(len(voiceEmbedding))), voiceEmbedding)
	if err != nil {
		return nil, fmt.Errorf("failed to create style tensor: %w", err)
	}
	defer styleTensor.Destroy()

	speedData := []float32{speed}
	speedTensor, err := ort.NewTensor(ort.NewShape(1), speedData)
	if err != nil {
		return nil, fmt.Errorf("failed to create speed tensor: %w", err)
	}
	defer speedTensor.Destroy()

	inputs := []ort.Value{inputIdsTensor, styleTensor, speedTensor}
	outputs := make([]ort.Value, 1)

	if err := e.session.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("failed to run inference: %w", err)
	}

	if outputs[0] == nil {
		return nil, fmt.Errorf("no output from model")
	}
	defer outputs[0].Destroy()

	outputTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected output tensor type")
	}

	outputData := outputTensor.GetData()
	return audio.NewAudio(outputData), nil
}

func (e *Engine) ListVoices() []string {
	return e.voices.List()
}

func (e *Engine) Info() engine.EngineInfo {
	return engine.EngineInfo{
		Name:       "kokoro",
		Languages:  []string{"en"},
		SampleRate: audio.SampleRate,
	}
}

func (e *Engine) Close() error {
	if e.session != nil {
		if err := e.session.Destroy(); err != nil {
			return err
		}
	}
	if err := ort.DestroyEnvironment(); err != nil {
		return err
	}
	return nil
}

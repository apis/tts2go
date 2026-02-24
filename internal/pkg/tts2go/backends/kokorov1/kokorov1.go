package kokorov1

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	ort "github.com/yalue/onnxruntime_go"

	"tts2go/internal/pkg/tts2go/audio"
	"tts2go/internal/pkg/tts2go/engine"
	"tts2go/internal/pkg/tts2go/voice"
)

func init() {
	engine.Register("kokoro-1.0", newEngineV10)
	engine.Register("kokoro-1.1", newEngineV11)
}

type Engine struct {
	session     *ort.DynamicAdvancedSession
	voiceStore  *voice.SherpaVoiceStore
	tokenizer   *Tokenizer
	version     string
	languages   []string
	numSpeakers int
	styleDim0   int
	styleDim1   int
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

func newEngineV10(cfg engine.EngineConfig) (engine.Engine, error) {
	return newEngine(cfg, "v1.0", []string{"en", "zh"}, 53, 510, 256)
}

func newEngineV11(cfg engine.EngineConfig) (engine.Engine, error) {
	return newEngine(cfg, "v1.1", []string{"zh", "en"}, 103, 510, 256)
}

func newEngine(cfg engine.EngineConfig, version string, languages []string, numSpeakers, styleDim0, styleDim1 int) (engine.Engine, error) {
	libPath := getOnnxRuntimeLibPath()
	ort.SetSharedLibraryPath(libPath)

	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX runtime: %w", err)
	}

	modelDir := filepath.Dir(cfg.ModelPath)
	if cfg.ModelPath == "models/model.onnx" || cfg.ModelPath == "models" {
		modelDir = "models"
	}

	modelPath := filepath.Join(modelDir, "model.onnx")

	tokensPath := cfg.TokensPath
	if tokensPath == "" {
		tokensPath = filepath.Join(modelDir, "tokens.txt")
	}

	voicesPath := filepath.Join(modelDir, "voices.bin")

	tokenizer, err := NewTokenizer(tokensPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer: %w", err)
	}

	voiceNames := voice.GetKokoroV10VoiceNames()
	if numSpeakers > len(voiceNames) {
		for i := len(voiceNames); i < numSpeakers; i++ {
			voiceNames = append(voiceNames, fmt.Sprintf("speaker_%d", i))
		}
	}

	voiceStore, err := voice.LoadSherpaVoicesBin(voicesPath, numSpeakers, styleDim0, styleDim1, voiceNames)
	if err != nil {
		return nil, fmt.Errorf("failed to load voices: %w", err)
	}

	inputNames := []string{"tokens", "style", "speed"}
	outputNames := []string{"audio"}

	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		inputNames,
		outputNames,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ONNX session: %w", err)
	}

	return &Engine{
		session:     session,
		voiceStore:  voiceStore,
		tokenizer:   tokenizer,
		version:     version,
		languages:   languages,
		numSpeakers: numSpeakers,
		styleDim0:   styleDim0,
		styleDim1:   styleDim1,
	}, nil
}

func (e *Engine) Generate(text, voiceName string, speed float32) (*audio.Audio, error) {
	lang := detectLanguage(text)
	tokens := e.tokenizer.EncodeWithLanguage(text, lang)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("failed to tokenize text")
	}

	tokenLen := len(tokens)
	styleEmbedding, err := e.voiceStore.GetStyle(voiceName, tokenLen)
	if err != nil {
		return nil, fmt.Errorf("failed to get voice embedding: %w", err)
	}

	inputIdsTensor, err := ort.NewTensor(ort.NewShape(1, int64(len(tokens))), tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputIdsTensor.Destroy()

	styleTensor, err := ort.NewTensor(ort.NewShape(1, int64(len(styleEmbedding))), styleEmbedding)
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
	return e.voiceStore.List()
}

func (e *Engine) Info() engine.EngineInfo {
	name := "kokoro-1.0"
	if e.version == "v1.1" {
		name = "kokoro-1.1"
	}
	return engine.EngineInfo{
		Name:       name,
		Languages:  e.languages,
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

func detectLanguage(text string) string {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return "zh"
		}
		if r >= 0x3400 && r <= 0x4DBF {
			return "zh"
		}
	}
	return "en"
}

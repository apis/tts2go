package engine

import "tts2go/internal/pkg/tts2go/audio"

type Engine interface {
	Generate(text, voice string, speed float32) (*audio.Audio, error)
	ListVoices() []string
	Info() EngineInfo
	Close() error
}

type EngineInfo struct {
	Name       string
	Languages  []string
	SampleRate int
}

type VoiceCloningEngine interface {
	Engine
	GenerateWithReference(text string, refAudio *audio.Audio, speed float32) (*audio.Audio, error)
}

type EngineConfig struct {
	ModelPath    string
	VoicesPath   string
	TokensPath   string
	Backend      string
	ModelVariant string
}

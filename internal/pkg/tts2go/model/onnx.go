package model

import (
	"tts2go/internal/pkg/tts2go/audio"
	"tts2go/internal/pkg/tts2go/engine"

	_ "tts2go/internal/pkg/tts2go/backends/kokoro"
)

type TTS struct {
	engine engine.Engine
}

func NewTTS(modelPath, voicesPath string) (*TTS, error) {
	cfg := engine.EngineConfig{
		ModelPath:  modelPath,
		VoicesPath: voicesPath,
	}
	eng, err := engine.New("kokoro", cfg)
	if err != nil {
		return nil, err
	}
	return &TTS{engine: eng}, nil
}

func (t *TTS) Generate(text, voiceName string, speed float32) (*audio.Audio, error) {
	return t.engine.Generate(text, voiceName, speed)
}

func (t *TTS) ListVoices() []string {
	return t.engine.ListVoices()
}

func (t *TTS) Close() error {
	return t.engine.Close()
}

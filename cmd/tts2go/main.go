package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"tts2go/internal/pkg/tts2go/audio"
	"tts2go/internal/pkg/tts2go/config"
	"tts2go/internal/pkg/tts2go/engine"

	_ "tts2go/internal/pkg/tts2go/backends/kokoro"
	_ "tts2go/internal/pkg/tts2go/backends/kokorov1"
	_ "tts2go/internal/pkg/tts2go/backends/pockettts"
)

func main() {
	fmt.Fprintf(os.Stderr, "tts2go %s\n", Version)

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	cfg, err := config.LoadAndParse()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse configuration")
	}

	if err := setupLogging(cfg); err != nil {
		log.Fatal().Err(err).Msg("Failed to setup logging")
	}

	log.Debug().
		Str("model", cfg.ModelPath).
		Str("voices", cfg.VoicesPath).
		Str("voice", cfg.Voice).
		Str("backend", cfg.Backend).
		Float32("speed", cfg.Speed).
		Msg("Configuration loaded")

	engineCfg := buildEngineConfig(cfg)

	log.Info().Str("backend", cfg.Backend).Msg("Loading TTS engine...")
	eng, err := engine.New(cfg.Backend, engineCfg)
	if err != nil {
		log.Fatal().Err(err).Str("backend", cfg.Backend).Msg("Failed to load engine")
	}
	defer eng.Close()

	info := eng.Info()
	log.Debug().
		Str("engine", info.Name).
		Strs("languages", info.Languages).
		Int("sample_rate", info.SampleRate).
		Msg("Engine loaded")

	voices := eng.ListVoices()
	sort.Strings(voices)

	if cfg.ListVoices {
		fmt.Fprintf(os.Stderr, "Backend: %s\n", info.Name)
		fmt.Fprintf(os.Stderr, "Languages: %s\n", strings.Join(info.Languages, ", "))
		fmt.Fprintf(os.Stderr, "Available voices (%d):\n", len(voices))
		for _, v := range voices {
			fmt.Fprintf(os.Stderr, "  %s\n", v)
		}
		return
	}

	if cfg.Voice == "" && cfg.ReferenceAudio == "" {
		if len(voices) == 0 && cfg.Backend != "pockettts" {
			log.Fatal().Msg("No voices available")
		}
		if len(voices) > 0 {
			cfg.Voice = voices[0]
			log.Info().Str("voice", cfg.Voice).Msg("Auto-selected voice")
		}
	}

	log.Debug().Strs("voices", voices).Msg("Available voices")

	var result *audio.Audio

	if cfg.ReferenceAudio != "" {
		cloningEngine, ok := eng.(engine.VoiceCloningEngine)
		if !ok {
			log.Fatal().Str("backend", cfg.Backend).Msg("Backend does not support voice cloning")
		}

		log.Info().Str("reference", cfg.ReferenceAudio).Msg("Loading reference audio...")
		refAudio, err := audio.LoadWAV(cfg.ReferenceAudio)
		if err != nil {
			log.Fatal().Err(err).Str("reference", cfg.ReferenceAudio).Msg("Failed to load reference audio")
		}

		log.Info().Str("text", truncateText(cfg.Text, 50)).Msg("Generating speech with voice cloning...")
		startTime := time.Now()

		result, err = cloningEngine.GenerateWithReference(cfg.Text, refAudio, cfg.Speed)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to generate audio")
		}

		elapsed := time.Since(startTime)
		log.Info().
			Dur("elapsed", elapsed).
			Float64("duration_sec", result.Duration()).
			Msg("Audio generated with voice cloning")
	} else {
		log.Info().Str("text", truncateText(cfg.Text, 50)).Msg("Generating speech...")
		startTime := time.Now()

		result, err = eng.Generate(cfg.Text, cfg.Voice, cfg.Speed)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to generate audio")
		}

		elapsed := time.Since(startTime)
		log.Info().
			Dur("elapsed", elapsed).
			Float64("duration_sec", result.Duration()).
			Msg("Audio generated")
	}

	if err := result.SaveWAV(cfg.Output); err != nil {
		log.Fatal().Err(err).Msg("Failed to save audio")
	}

	log.Info().Str("output", cfg.Output).Msg("Audio saved successfully")
}

func buildEngineConfig(cfg *config.Config) engine.EngineConfig {
	engineCfg := engine.EngineConfig{
		ModelPath:    cfg.ModelPath,
		VoicesPath:   cfg.VoicesPath,
		TokensPath:   cfg.TokensPath,
		Backend:      cfg.Backend,
		ModelVariant: cfg.ModelVariant,
	}

	switch cfg.Backend {
	case "kokoro-v1.0":
		if engineCfg.ModelPath == "models/model.onnx" {
			engineCfg.ModelPath = "models/kokoro-v1.0"
		}
		if engineCfg.VoicesPath == "models/voices.npz" || engineCfg.VoicesPath == "models/voices" {
			engineCfg.VoicesPath = filepath.Join(engineCfg.ModelPath, "voices")
		}
	case "kokoro-v1.1":
		if engineCfg.ModelPath == "models/model.onnx" {
			engineCfg.ModelPath = "models/kokoro-v1.1"
		}
		if engineCfg.VoicesPath == "models/voices.npz" || engineCfg.VoicesPath == "models/voices" {
			engineCfg.VoicesPath = filepath.Join(engineCfg.ModelPath, "voices")
		}
	case "pockettts":
		if engineCfg.ModelPath == "models/model.onnx" {
			engineCfg.ModelPath = "models/pockettts"
		}
	}

	return engineCfg
}

func setupLogging(cfg *config.Config) error {
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.LogLevel))
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		log.Logger = zerolog.New(f).With().Timestamp().Logger()
	}

	return nil
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

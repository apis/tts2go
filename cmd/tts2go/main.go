package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"tts2go/internal/pkg/tts2go/config"
	"tts2go/internal/pkg/tts2go/model"
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
		Float32("speed", cfg.Speed).
		Msg("Configuration loaded")

	log.Info().Msg("Loading TTS model...")
	tts, err := model.NewTTS(cfg.ModelPath, cfg.VoicesPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load model")
	}
	defer tts.Close()

	voices := tts.ListVoices()
	sort.Strings(voices)

	if cfg.ListVoices {
		fmt.Fprintf(os.Stderr, "Available voices (%d):\n", len(voices))
		for _, v := range voices {
			fmt.Fprintf(os.Stderr, "  %s\n", v)
		}
		return
	}

	if cfg.Voice == "" {
		if len(voices) == 0 {
			log.Fatal().Msg("No voices available")
		}
		cfg.Voice = voices[0]
		log.Info().Str("voice", cfg.Voice).Msg("Auto-selected voice")
	}

	log.Debug().Strs("voices", voices).Msg("Available voices")

	log.Info().Str("text", truncateText(cfg.Text, 50)).Msg("Generating speech...")
	startTime := time.Now()

	audio, err := tts.Generate(cfg.Text, cfg.Voice, cfg.Speed)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to generate audio")
	}

	elapsed := time.Since(startTime)
	log.Info().
		Dur("elapsed", elapsed).
		Float64("duration_sec", audio.Duration()).
		Msg("Audio generated")

	if err := audio.SaveWAV(cfg.Output); err != nil {
		log.Fatal().Err(err).Msg("Failed to save audio")
	}

	log.Info().Str("output", cfg.Output).Msg("Audio saved successfully")
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

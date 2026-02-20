package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	ModelPath  string  `mapstructure:"model_path"`
	VoicesPath string  `mapstructure:"voices_path"`
	Text       string  `mapstructure:"text"`
	Output     string  `mapstructure:"output"`
	Voice      string  `mapstructure:"voice"`
	Speed      float32 `mapstructure:"speed"`
	LogLevel   string  `mapstructure:"log_level"`
	LogFile    string  `mapstructure:"log_file"`
}

func LoadAndParse() (*Config, error) {
	viper.SetDefault("model_path", "models/model.onnx")
	viper.SetDefault("voices_path", "models/voices.npz")
	viper.SetDefault("output", "output.wav")
	viper.SetDefault("voice", "expr-voice-2-f")
	viper.SetDefault("speed", 1.0)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("log_file", "")

	flagSet := pflag.NewFlagSet("kittentts", pflag.ContinueOnError)
	configFile := flagSet.StringP("config", "c", "", "Path to config file")
	flagSet.StringP("text", "t", "", "Text to synthesize")
	flagSet.StringP("output", "o", "", "Output WAV file")
	flagSet.StringP("voice", "v", "", "Voice to use")
	flagSet.Float32P("speed", "s", 1.0, "Speech speed (0.5-2.0)")
	flagSet.StringP("model", "m", "", "Path to ONNX model file")
	flagSet.String("voices", "", "Path to voices NPZ file")
	flagSet.StringP("log-level", "l", "", "Log level (debug, info, warn, error)")
	flagSet.String("log-file", "", "Log file path")
	helpFlag := flagSet.BoolP("help", "h", false, "Show help message")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	if *helpFlag {
		fmt.Fprintf(os.Stderr, "Usage: kittentts [options] [text]\n\nOptions:\n")
		flagSet.PrintDefaults()
		os.Exit(0)
	}

	if err := viper.BindPFlag("text", flagSet.Lookup("text")); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag("output", flagSet.Lookup("output")); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag("voice", flagSet.Lookup("voice")); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag("speed", flagSet.Lookup("speed")); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag("model_path", flagSet.Lookup("model")); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag("voices_path", flagSet.Lookup("voices")); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag("log_level", flagSet.Lookup("log-level")); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag("log_file", flagSet.Lookup("log-file")); err != nil {
		return nil, err
	}

	if *configFile != "" {
		viper.SetConfigFile(*configFile)
	} else {
		viper.SetConfigName("kittentts.cfg")
		viper.SetConfigType("toml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("configs")
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(filepath.Join(home, ".config", "kittentts"))
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	viper.SetEnvPrefix("KITTENTTS")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.Text == "" {
		args := flagSet.Args()
		if len(args) > 0 {
			cfg.Text = strings.Join(args, " ")
		}
	}

	if cfg.Text == "" {
		return nil, fmt.Errorf("text is required (use -t flag or provide as argument)")
	}

	if cfg.Speed < 0.5 || cfg.Speed > 2.0 {
		return nil, fmt.Errorf("speed must be between 0.5 and 2.0")
	}

	return &cfg, nil
}

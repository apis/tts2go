package config

import (
	"fmt"
	"io"
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
	ListVoices bool    `mapstructure:"list_voices"`
}

func detectVoicesPath() string {
	if info, err := os.Stat("models/voices"); err == nil && info.IsDir() {
		entries, err := os.ReadDir("models/voices")
		if err == nil && len(entries) > 0 {
			return "models/voices"
		}
	}
	if _, err := os.Stat("models/voices.npz"); err == nil {
		return "models/voices.npz"
	}
	return "models/voices.npz"
}

func LoadAndParse() (*Config, error) {
	viper.SetDefault("model_path", "models/model.onnx")
	viper.SetDefault("voices_path", detectVoicesPath())
	viper.SetDefault("output", "output.wav")
	viper.SetDefault("voice", "")
	viper.SetDefault("speed", 1.0)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("log_file", "")

	flagSet := pflag.NewFlagSet("tts2go", pflag.ContinueOnError)
	configFile := flagSet.StringP("config", "c", "", "Path to config file")
	flagSet.StringP("text", "t", "", "Text to synthesize (use '-' to read from stdin)")
	flagSet.StringP("file", "f", "", "Read text from file")
	flagSet.StringP("output", "o", "", "Output WAV file")
	flagSet.StringP("voice", "v", "", "Voice to use")
	flagSet.Float32P("speed", "s", 1.0, "Speech speed (0.5-2.0)")
	flagSet.StringP("model", "m", "", "Path to ONNX model file")
	flagSet.String("voices", "", "Path to voices (NPZ file or directory with .npy/.bin files)")
	flagSet.StringP("log-level", "l", "", "Log level (debug, info, warn, error)")
	flagSet.String("log-file", "", "Log file path")
	flagSet.Bool("list-voices", false, "List available voices and exit")
	helpFlag := flagSet.BoolP("help", "h", false, "Show help message")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	if *helpFlag {
		fmt.Fprintf(os.Stderr, "Usage: tts2go [options] [text]\n\nOptions:\n")
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
	if err := viper.BindPFlag("list_voices", flagSet.Lookup("list-voices")); err != nil {
		return nil, err
	}

	if *configFile != "" {
		viper.SetConfigFile(*configFile)
	} else {
		viper.SetConfigName("tts2go.cfg")
		viper.SetConfigType("toml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("configs")
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(filepath.Join(home, ".config", "tts2go"))
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	viper.SetEnvPrefix("TTS2GO")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.VoicesPath == "" {
		cfg.VoicesPath = detectVoicesPath()
	}

	textFile, _ := flagSet.GetString("file")
	if textFile != "" {
		content, err := os.ReadFile(textFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read text file: %w", err)
		}
		cfg.Text = strings.TrimSpace(string(content))
	} else if cfg.Text == "-" {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %w", err)
		}
		cfg.Text = strings.TrimSpace(string(content))
	} else if cfg.Text == "" {
		args := flagSet.Args()
		if len(args) > 0 {
			cfg.Text = strings.Join(args, " ")
		}
	}

	if cfg.Text == "" && !cfg.ListVoices {
		return nil, fmt.Errorf("text is required (use -t, -f, or provide as argument)")
	}

	if cfg.Speed < 0.5 || cfg.Speed > 2.0 {
		return nil, fmt.Errorf("speed must be between 0.5 and 2.0")
	}

	return &cfg, nil
}

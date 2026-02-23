# TTS2Go

A Go implementation of text-to-speech synthesis using ONNX Runtime, supporting Kokoro and Kitten TTS models.

## Features

- Text-to-speech synthesis using Kokoro or Kitten TTS ONNX models
- Multiple voice support
- Configurable speech speed (0.5x - 2.0x)
- WAV audio output at 24kHz

## Requirements

- Go 1.25+
- ONNX Runtime 1.24.2+ shared library
- Model files from HuggingFace

## Installation

### 1. Install ONNX Runtime

Use the justfile recipe to download ONNX Runtime locally:

```bash
just fetch-onnxruntime
```

This downloads the library to `lib/` and prints the export command for `ONNXRUNTIME_LIB_PATH`.

Or install system-wide:
```bash
# Linux example
wget https://github.com/microsoft/onnxruntime/releases/download/v1.24.2/onnxruntime-linux-x64-1.24.2.tgz
tar xzf onnxruntime-linux-x64-1.24.2.tgz
sudo cp onnxruntime-linux-x64-1.24.2/lib/libonnxruntime.so /usr/local/lib/
sudo ldconfig
```

### 2. Download Model Files

Choose either Kitten TTS or Kokoro models:

#### Kitten TTS Models (smaller, faster)

```bash
# Download default model (nano-fp32, 57MB)
just fetch-models

# Or choose a specific variant:
just fetch-models nano-int8   # Quantized, smallest (18MB)
just fetch-models nano-fp32   # Full precision (57MB)
just fetch-models micro       # Medium model (41MB)
just fetch-models mini        # Best quality (78MB)
```

Models are downloaded from https://huggingface.co/KittenML

#### Kokoro Models (higher quality)

```bash
# Download Kokoro model (q8 quantized, 92MB, recommended)
just fetch-kokoro

# Or choose a specific variant:
just fetch-kokoro q8      # 8-bit quantized (92MB, recommended)
just fetch-kokoro fp16    # Half precision (163MB)
just fetch-kokoro fp32    # Full precision (326MB)
just fetch-kokoro q4f16   # 4-bit + fp16 hybrid (154MB)
```

Models are downloaded from https://huggingface.co/onnx-community/Kokoro-82M-ONNX

### 3. Build

```bash
just build
# or
go build -o bin/tts2go ./cmd/tts2go
```

## Usage

```bash
# Using just run (automatically sets ONNXRUNTIME_LIB_PATH)
# Note: Use nested quotes for text with spaces
just run -t '"Hello, world!"' -o output.wav

# Or run directly (requires ONNXRUNTIME_LIB_PATH or system-installed library)
export ONNXRUNTIME_LIB_PATH=/path/to/lib/libonnxruntime.so
./bin/tts2go -t "Hello, world!" -o output.wav

# With voice selection (Kokoro example)
./bin/tts2go -t "Hello, world!" -v af_bella -o output.wav

# With speed adjustment
./bin/tts2go -t "Hello, world!" -s 1.2 -o output.wav

# Read text from file
./bin/tts2go -f input.txt -o output.wav

# Read from stdin
echo "Hello, world!" | ./bin/tts2go -t - -o output.wav

# List available voices
./bin/tts2go --list-voices
```

### Available Voices

Voices depend on which model you downloaded:

**Kitten TTS** (voices.npz):
- Female: `expr-voice-2-f`, `expr-voice-3-f`, `expr-voice-4-f`, `expr-voice-5-f`
- Male: `expr-voice-2-m`, `expr-voice-3-m`, `expr-voice-4-m`, `expr-voice-5-m`

**Kokoro** (individual .bin files):
- American Female: `af`, `af_bella`, `af_nicole`, `af_sarah`, `af_sky`
- American Male: `am_adam`, `am_michael`
- British Female: `bf_emma`, `bf_isabella`
- British Male: `bm_george`, `bm_lewis`

Use `--list-voices` to see available voices for your installed model.

### Configuration

Configuration can be provided via:
1. Command-line flags (highest priority)
2. Config file (`tts2go.cfg.toml`)
3. Environment variables (prefixed with `TTS2GO_`)

See `configs/tts2go.cfg.toml` for an example configuration file.

## Development

```bash
# Run tests
just test

# Format code
just fmt

# Full rebuild
just rebuild
```

## License

MIT

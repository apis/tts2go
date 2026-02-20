# KittenTTS Go

A Go implementation of KittenTTS text-to-speech synthesis using ONNX Runtime.

## Features

- Text-to-speech synthesis using Kokoro/Kitten TTS models
- Multiple voice support
- Configurable speech speed
- WAV audio output at 24kHz

## Requirements

- Go 1.24+
- ONNX Runtime shared library (`libonnxruntime.so` on Linux)
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
wget https://github.com/microsoft/onnxruntime/releases/download/v1.20.1/onnxruntime-linux-x64-1.20.1.tgz
tar xzf onnxruntime-linux-x64-1.20.1.tgz
sudo cp onnxruntime-linux-x64-1.20.1/lib/libonnxruntime.so /usr/local/lib/
sudo ldconfig
```

### 2. Download Model Files

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

### 3. Build

```bash
just build
# or
go build -o bin/kittentts ./cmd/kittentts
```

## Usage

```bash
# Using just run (automatically sets ONNXRUNTIME_LIB_PATH)
just run -t "Hello, world!" -o output.wav

# Or run directly (requires ONNXRUNTIME_LIB_PATH or system-installed library)
./bin/kittentts -t "Hello, world!" -o output.wav

# With voice selection
./bin/kittentts -t "Hello, world!" -v af_heart -o output.wav

# With speed adjustment
./bin/kittentts -t "Hello, world!" -s 1.2 -o output.wav

# Full options
./bin/kittentts \
    --text "Hello, world!" \
    --voice af_heart \
    --speed 1.0 \
    --model models/model.onnx \
    --voices models/voices.npz \
    --output output.wav
```

### Available Voices

- Female: `af_heart`, `af_alloy`, `af_aoede`, `af_bella`, `af_jessica`, `af_kore`, `af_nicole`, `af_nova`, `af_river`, `af_sarah`, `af_sky`
- Male: `am_adam`, `am_echo`, `am_eric`, `am_fenrir`, `am_liam`, `am_michael`, `am_onyx`, `am_puck`, `am_santa`
- British Female: `bf_emma`, `bf_isabella`
- British Male: `bm_george`, `bm_lewis`

### Configuration

Configuration can be provided via:
1. Command-line flags
2. Config file (`kittentts.cfg.toml`)
3. Environment variables (prefixed with `KITTENTTS_`)

See `configs/kittentts.cfg.toml` for an example configuration file.

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

# Integration Plan: Kokoro v1.0/v1.1 + PocketTTS

## Context

Extend tts2Go to support additional TTS models:
1. **Kokoro multi-lang v1.0** - Chinese + English, 53 speakers
2. **Kokoro multi-lang v1.1** - Chinese optimized, mixed zh/en support
3. **PocketTTS** - Flow-based TTS with zero-shot voice cloning

## Approach: Build-Time Model Selection

Each model type has its own build command producing a dedicated binary. This keeps binaries smaller and avoids runtime complexity.

---

## Build Commands

```bash
# Current (Kokoro 0.8 / Kitten)
just build                    # -> bin/tts2go

# Kokoro multi-lang v1.0
just build-kokoro-v1.0        # -> bin/tts2go-kokoro-v1.0

# Kokoro multi-lang v1.1
just build-kokoro-v1.1        # -> bin/tts2go-kokoro-v1.1

# PocketTTS
just build-pockettts          # -> bin/tts2go-pockettts
```

## Model Fetch Commands

```bash
# Current (unchanged)
just fetch-kokoro             # Downloads Kokoro 0.8 to models/

# Kokoro v1.0 multi-lang
just fetch-kokoro-v1.0        # Downloads to models/kokoro-v1.0/

# Kokoro v1.1 multi-lang
just fetch-kokoro-v1.1        # Downloads to models/kokoro-v1.1/

# PocketTTS
just fetch-pockettts          # Downloads to models/pockettts/
just fetch-pockettts int8     # Downloads quantized version
```

---

## CLI Usage Examples

### Current tts2go (Kokoro 0.8)

```bash
# Build and run
just build
just run -t "Hello world" -o output.wav

# Or direct execution
export ONNXRUNTIME_LIB_PATH=./lib/libonnxruntime.so
./bin/tts2go -t "Hello world" -o output.wav
./bin/tts2go -t "Hello world" -v af_bella -o output.wav
./bin/tts2go -t "Hello world" -s 1.2 -o output.wav
./bin/tts2go --list-voices
```

### tts2go-kokoro-v1.0 (Multi-lang)

```bash
# Fetch models and build
just fetch-kokoro-v1.0
just build-kokoro-v1.0

# Run with English voice
./bin/tts2go-kokoro-v1.0 -t "Hello world" -v af_bella -o output.wav

# Run with Chinese voice
./bin/tts2go-kokoro-v1.0 -t "你好世界" -v zf_xiaoxiao -o output.wav

# List available voices (53 speakers)
./bin/tts2go-kokoro-v1.0 --list-voices
```

### tts2go-kokoro-v1.1 (Chinese Optimized)

```bash
# Fetch models and build
just fetch-kokoro-v1.1
just build-kokoro-v1.1

# Run with mixed Chinese/English
./bin/tts2go-kokoro-v1.1 -t "Hello, 你好" -v zf_xiaoxiao -o output.wav
```

### tts2go-pockettts (Voice Cloning)

```bash
# Fetch models and build
just fetch-pockettts
just build-pockettts

# Voice cloning from reference audio
./bin/tts2go-pockettts -t "Hello world" --reference speaker.wav -o output.wav

# With speed adjustment
./bin/tts2go-pockettts -t "Hello world" --reference speaker.wav -s 0.9 -o output.wav
```

---

## Architecture: Separate Binaries

Each binary has its own entry point and model-specific code:

```
cmd/
├── tts2go/                    # Current Kokoro 0.8 / Kitten
│   └── main.go
├── tts2go-kokoro-v1/          # Kokoro v1.0 and v1.1
│   └── main.go
└── tts2go-pockettts/          # PocketTTS
    └── main.go

internal/pkg/tts2go/
├── model/
│   ├── onnx.go                # Current implementation
│   ├── kokorov1.go            # Kokoro v1.0/v1.1 implementation
│   └── pockettts.go           # PocketTTS implementation
├── tokenizer/
│   ├── tokenizer.go           # Current IPA tokenizer
│   ├── kokorov1_tokens.go     # tokens.txt loader for Kokoro v1.x
│   └── pockettts_vocab.go     # vocab.json loader for PocketTTS
├── voice/
│   ├── voice.go               # Current voice embeddings
│   └── audioclone.go          # Reference audio for PocketTTS
├── audio/
│   └── wav.go                 # Extend with LoadWAV()
├── preprocess/
│   └── preprocess.go          # Shared text preprocessing
├── phonemizer/
│   └── phonemizer.go          # Current goruut phonemizer
└── config/
    └── config.go              # Shared config loading
```

---

## Model Directory Structure

```
models/
├── model.onnx                 # Current Kokoro 0.8
├── voices/                    # Current voice .bin files
│
├── kokoro-v1.0/
│   ├── model.onnx
│   ├── tokens.txt
│   └── voices/
│       ├── af_*.bin           # American English
│       ├── bf_*.bin           # British English
│       ├── zf_*.bin           # Chinese female
│       └── zm_*.bin           # Chinese male
│
├── kokoro-v1.1/
│   ├── model.onnx
│   ├── tokens.txt
│   └── voices/
│
└── pockettts/
    ├── text_conditioner.onnx
    ├── lm_main.onnx           # (or lm_main_int8.onnx)
    ├── lm_flow.onnx           # (or lm_flow_int8.onnx)
    ├── encoder.onnx
    ├── decoder.onnx           # (or decoder_int8.onnx)
    ├── vocab.json
    └── token_scores.json
```

---

## Implementation Phases

### Phase 1: Kokoro v1.0/v1.1

1. **Create `internal/pkg/tts2go/tokenizer/kokorov1_tokens.go`**
   - Load `tokens.txt` vocabulary
   - Simple token-to-index mapping

2. **Create `internal/pkg/tts2go/model/kokorov1.go`**
   - Same ONNX interface: `input_ids`, `style`, `speed` → `waveform`
   - Use tokens.txt tokenizer instead of IPA phonemizer
   - Support version flag for v1.0 vs v1.1 model paths

3. **Create `cmd/tts2go-kokoro-v1/main.go`**
   - Entry point for Kokoro v1.x builds
   - Accept `--version` flag (1.0 or 1.1, default 1.0)

4. **Update justfile**
   - Add `fetch-kokoro-v1.0` and `fetch-kokoro-v1.1` recipes
   - Add `build-kokoro-v1.0` and `build-kokoro-v1.1` recipes

### Phase 2: PocketTTS

5. **Create `internal/pkg/tts2go/tokenizer/pockettts_vocab.go`**
   - Load `vocab.json` vocabulary
   - Character-level tokenization

6. **Create `internal/pkg/tts2go/voice/audioclone.go`**
   - Load reference audio WAV
   - Normalize and resample to 24kHz

7. **Extend `internal/pkg/tts2go/audio/wav.go`**
   - Add `LoadWAV(path string) (*Audio, error)`

8. **Create `internal/pkg/tts2go/model/pockettts.go`**
   - Multi-session ONNX manager (5 models)
   - Pipeline: text_conditioner → lm_main → lm_flow (ODE loop) → decoder
   - Voice embedding from encoder on reference audio

9. **Create `cmd/tts2go-pockettts/main.go`**
   - Entry point for PocketTTS
   - Require `--reference` flag for voice cloning

10. **Update justfile**
    - Add `fetch-pockettts` recipe
    - Add `build-pockettts` recipe

### Phase 3: Documentation

11. **Update README.md**
    - Document all model variants
    - Add build/run examples

---

## Key Differences Between Models

| Feature | Kokoro 0.8 | Kokoro v1.x | PocketTTS |
|---------|------------|-------------|-----------|
| Languages | English | EN + Chinese | Multi |
| Voices | 11 | 53+ | Voice cloning |
| Phonemizer | goruut (IPA) | tokens.txt | vocab.json |
| ONNX Models | 1 | 1 | 5 |
| Voice Input | .bin embedding | .bin embedding | Reference WAV |
| Sample Rate | 24kHz | 24kHz | 24kHz |

---

## Verification

1. **Current build unchanged**: `just build && just run -t "Hello" -o test.wav`
2. **Kokoro v1.0**:
   ```bash
   just fetch-kokoro-v1.0
   just build-kokoro-v1.0
   ./bin/tts2go-kokoro-v1.0 -t "Hello" -o test.wav
   ```
3. **Kokoro v1.1**:
   ```bash
   just fetch-kokoro-v1.1
   just build-kokoro-v1.1
   ./bin/tts2go-kokoro-v1.1 -t "你好" -o test.wav
   ```
4. **PocketTTS**:
   ```bash
   just fetch-pockettts
   just build-pockettts
   ./bin/tts2go-pockettts --reference voice.wav -t "Hello" -o test.wav
   ```

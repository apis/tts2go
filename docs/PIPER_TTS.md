# Piper TTS Integration

This document describes the Piper TTS model structure, naming conventions, and integration details for TTS2Go.

## Overview

Piper is a fast, local neural text-to-speech system that uses VITS (Variational Inference with adversarial learning for end-to-end Text-to-Speech). Models are available in ONNX format and support 30+ languages with hundreds of voices.

## Model Naming Convention

### Repository Names
```
vits-piper-{lang_COUNTRY}-{voice}-{quality}
```

Examples:
- `vits-piper-en_US-lessac-high`
- `vits-piper-en_GB-cori-medium`
- `vits-piper-de_DE-thorsten-low`

### File Names
```
{lang_COUNTRY}-{voice}-{quality}.onnx      # ONNX model
{lang_COUNTRY}-{voice}-{quality}.onnx.json # Config file
```

## Quality Levels

| Quality | Sample Rate | Description |
|---------|-------------|-------------|
| x_low   | 16000 Hz    | Smallest, fastest |
| low     | 16000 Hz    | Small, fast |
| medium  | 22050 Hz    | Balanced quality/speed |
| high    | 22050 Hz    | Best quality |

## Model File Structure

Each Piper model package contains:

```
vits-piper-en_US-lessac-high/
├── en_US-lessac-high.onnx           # ONNX model file
├── en_US-lessac-high.onnx.json      # Model configuration
├── MODEL_CARD                        # Model information
├── tokens.txt                        # Phoneme token definitions
└── espeak-ng-data/                   # eSpeak-NG phonemization data
    ├── en_dict                       # Language dictionaries
    ├── phondata                      # Phoneme data
    ├── phonindex
    ├── phontab
    └── lang/                         # Language definitions
```

## Configuration File Format

The `.onnx.json` config file contains:

```json
{
    "audio": {
        "sample_rate": 22050
    },
    "espeak": {
        "voice": "en-us"
    },
    "inference": {
        "noise_scale": 0.667,
        "length_scale": 1,
        "noise_w": 0.8
    },
    "phoneme_type": "espeak",
    "phoneme_id_map": {
        " ": [3],
        "a": [14],
        "ə": [59],
        ...
    },
    "num_symbols": 256,
    "num_speakers": 1,
    "speaker_id_map": {},
    "piper_version": "1.0.0"
}
```

### Configuration Fields

| Field | Description |
|-------|-------------|
| `audio.sample_rate` | Output audio sample rate (16000 or 22050) |
| `espeak.voice` | eSpeak-NG voice code for phonemization |
| `inference.noise_scale` | VITS noise scale parameter (typically 0.667) |
| `inference.length_scale` | Speech speed multiplier (1.0 = normal) |
| `inference.noise_w` | VITS noise weight (typically 0.8) |
| `phoneme_type` | Phonemizer type ("espeak" for eSpeak-NG) |
| `phoneme_id_map` | Phoneme character to token ID mapping |
| `num_symbols` | Total vocabulary size |
| `num_speakers` | Number of speakers (1 for single-speaker) |
| `speaker_id_map` | Speaker name to ID mapping (multi-speaker models) |

## HuggingFace Model Locations

### 1. Official Piper Voices (rhasspy)

Primary source for Piper voices:
```
https://huggingface.co/rhasspy/piper-voices/resolve/main/{lang_code}/{lang_COUNTRY}/{voice}/{quality}/{filename}
```

Example:
```
https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/high/en_US-lessac-high.onnx
```

### 2. Sherpa-ONNX Converted Models (csukuangfj)

Pre-packaged with espeak-ng-data:
```
https://huggingface.co/csukuangfj/vits-piper-{lang_COUNTRY}-{voice}-{quality}/
```

Example:
```
https://huggingface.co/csukuangfj/vits-piper-en_US-glados-high/
```

### 3. GitHub Releases

Bundled as `.tar.bz2` archives:
```
https://github.com/k2-fsa/sherpa-onnx/releases/tag/tts-models
```

## Available English Voices

### en_US (American English)

| Voice | Quality | Speakers | Notes |
|-------|---------|----------|-------|
| amy | low, medium | 1 | Female |
| arctic | medium | 18 | Multi-speaker |
| bryce | medium | 1 | Male |
| danny | low | 1 | Male |
| glados | high | 1 | Portal GLaDOS voice |
| hfc_female | medium | 1 | Female |
| hfc_male | medium | 1 | Male |
| joe | medium | 1 | Male |
| john | medium | 1 | Male |
| kathleen | low | 1 | Female |
| kristin | medium | 1 | Female |
| kusal | medium | 1 | Male |
| l2arctic | medium | 24 | Multi-speaker, accented |
| lessac | high, low, medium | 1 | High quality female |
| libritts | high | 904 | Multi-speaker corpus |
| libritts_r | medium | 904 | Multi-speaker corpus |
| ljspeech | high, medium | 1 | Classic female voice |
| miro | high | 1 | Male |
| norman | medium | 1 | Male |
| reza_ibrahim | medium | 1 | Male |
| ryan | high, low, medium | 1 | Male |
| sam | medium | 1 | Male |

### en_GB (British English)

| Voice | Quality | Speakers | Notes |
|-------|---------|----------|-------|
| alan | low, medium | 1 | Male |
| alba | medium | 1 | Female, Scottish |
| aru | medium | 12 | Multi-speaker |
| cori | high, medium | 1 | Female |
| dii | high | 1 | Female |
| jenny_dioco | medium | 1 | Female |
| miro | high | 1 | Male |
| northern_english_male | medium | 1 | Male, Northern accent |
| semaine | medium | 4 | Multi-speaker |
| southern_english_female | low, medium | 1-6 | Female |
| southern_english_male | medium | 8 | Male |
| vctk | medium | 109 | Multi-speaker corpus |

## Other Supported Languages

Piper supports 30+ languages including:

| Language | Code | Example Voices |
|----------|------|----------------|
| German | de_DE | thorsten, eva_k, karlsson, glados |
| Spanish | es_ES, es_MX, es_AR | carlfm, davefx, miro, claude |
| French | fr_FR | siwis, gilles, tom, miro |
| Italian | it_IT | paola, riccardo, miro |
| Dutch | nl_NL, nl_BE | mls, nathalie, miro |
| Polish | pl_PL | darkman, gosia, mc_speech |
| Portuguese | pt_BR, pt_PT | faber, edresson, miro |
| Russian | ru_RU | denis, dmitri, irina |
| Chinese | zh_CN | huayan |
| Japanese | ja_JP | kokoro |
| Korean | ko_KR | mls |
| Arabic | ar_JO | kareem |
| Hindi | hi_IN | pratham, priyamvada, rohan |
| And many more... | | |

## ONNX Model Inputs/Outputs

### Inputs

| Name | Shape | Type | Description |
|------|-------|------|-------------|
| input | [1, phoneme_length] | int64 | Phoneme token IDs |
| input_lengths | [1] | int64 | Length of phoneme sequence |
| scales | [3] | float32 | [noise_scale, length_scale, noise_w] |
| sid | [1] | int64 | Speaker ID (0 for single-speaker) |

### Outputs

| Name | Shape | Type | Description |
|------|-------|------|-------------|
| output | [1, 1, audio_length] | float32 | Generated audio waveform |

## Phonemization

Piper uses eSpeak-NG for grapheme-to-phoneme conversion. The phonemes are then mapped to token IDs using the `phoneme_id_map` from the config.

### Phonemization Pipeline

```
Text → eSpeak-NG → IPA Phonemes → Token IDs → VITS Model → Audio
```

### Example

```
Input: "Hello world"
eSpeak: "həˈloʊ wˈɜːld"
Tokens: [20, 59, 120, 24, 27, 100, 3, 35, 120, 62, 122, 24, 17]
```

## Differences from Kitten/Kokoro

| Feature | Piper | Kitten/Kokoro |
|---------|-------|---------------|
| Phonemization | eSpeak-NG (external) | Built-in phonemizer |
| Voice Selection | Speaker ID | Voice embeddings |
| Model Architecture | VITS | StyleTTS2 / Kokoro |
| Multi-speaker | speaker_id input | Voice embedding files |
| Data Requirements | espeak-ng-data folder | None |

## Integration Notes

### Required Components

1. **eSpeak-NG Data**: Must include `espeak-ng-data/` folder with language dictionaries
2. **Config Parser**: Parse `.onnx.json` for phoneme mapping and inference params
3. **Phonemizer**: Use espeak-ng library or bundled data for phonemization
4. **VITS Inference**: Different input tensor structure than Kitten/Kokoro

### Recommended Approach

Use the Sherpa-ONNX converted models (`csukuangfj/vits-piper-*`) as they include:
- Pre-packaged espeak-ng-data
- Consistent file naming
- tokens.txt for reference
- MODEL_CARD with licensing info

## Integration Analysis for TTS2Go

### ONNX Runtime Compatibility

**No changes required** to the current ONNX runtime (`onnxruntime_go`). Piper uses standard ONNX format.

The difference is in model inputs:

| Current (Kitten/Kokoro) | Piper VITS |
|-------------------------|------------|
| `input_ids` [1, seq_len] int64 | `input` [1, seq_len] int64 |
| `style` [1, 256] float32 | `input_lengths` [1] int64 |
| `speed` [1] float32 | `scales` [3] float32 |
| | `sid` [1] int64 (optional) |

Where `scales` = [noise_scale, length_scale, noise_scale_w] from config.

### Phonemization Requirements

**goruut WILL NOT WORK** for Piper models because:

1. Piper models are trained with **eSpeak-NG phonemes**
2. The `phoneme_id_map` in config expects eSpeak-NG IPA symbols
3. Gruut uses different phoneme inventory and representations
4. Token IDs won't match if wrong phonemizer is used

### Phonemization Options for Go

#### Option 1: External espeak-ng command (Recommended)
```bash
# Install on Linux
sudo apt install espeak-ng

# Phonemize text
espeak-ng --ipa -q -v en-us "Hello world"
# Output: həlˈoʊ wˈɜːld
```

Pros:
- Simple to implement (exec.Command)
- No CGO dependencies
- Uses system espeak-ng (well maintained)

Cons:
- Requires espeak-ng installed on system
- Process spawn overhead per request

#### Option 2: CGO Bindings

Available Go packages:
- [gopkg.in/BenLubar/espeak.v2](https://pkg.go.dev/gopkg.in/BenLubar/espeak.v2) - espeak bindings with phonemization
- [github.com/djangulo/go-espeak](https://github.com/djangulo/go-espeak) - CGO bindings for espeak

```go
// Example with BenLubar/espeak
import "gopkg.in/BenLubar/espeak.v2"

espeak.SetVoiceByName("en-us")
phonemes, _ := espeak.TextToPhonemes("Hello world", espeak.PhonemeIPA)
```

Pros:
- Direct library calls (faster)
- No process spawn

Cons:
- CGO compilation complexity
- Links against system libespeak
- Older packages (may need updating)

#### Option 3: Bundled espeak-ng-data (Sherpa-ONNX approach)

Sherpa-ONNX bundles the `espeak-ng-data` directory and uses the piper-phonemize C++ library:
```cpp
#include "espeak-ng/speak_lib.h"
#include "phonemize.hpp"
```

This would require:
- CGO wrapper for piper-phonemize
- Bundling espeak-ng-data (~50MB)

#### Option 4: Hybrid Architecture

Use goruut for Kitten/Kokoro models, espeak-ng for Piper models:

```go
type Phonemizer interface {
    Phonemize(text string) ([]int64, error)
}

type GruutPhonemizer struct { /* for Kitten/Kokoro */ }
type EspeakPhonemizer struct { /* for Piper */ }
```

### Recommended Implementation Strategy

1. **Phase 1**: Use external `espeak-ng` command
   - Quick to implement
   - Works immediately
   - Can optimize later

2. **Phase 2**: Add CGO bindings (optional)
   - If performance is critical
   - Bundle espeak-ng statically

3. **Phase 3**: Support bundled espeak-ng-data
   - For standalone distribution
   - No system dependencies

### Token Mapping

After phonemization, map IPA symbols to token IDs using `phoneme_id_map` from config:

```go
func (p *PiperPhonemizer) TextToTokens(text string) ([]int64, error) {
    // 1. Run espeak-ng to get IPA phonemes
    phonemes := p.espeakPhonemes(text)

    // 2. Map each phoneme to token ID using config
    tokens := make([]int64, 0, len(phonemes))
    for _, ph := range phonemes {
        if id, ok := p.phonemeIDMap[ph]; ok {
            tokens = append(tokens, int64(id[0]))
        }
    }

    return tokens, nil
}
```

## References

- [Piper GitHub](https://github.com/rhasspy/piper)
- [Piper Voices on HuggingFace](https://huggingface.co/rhasspy/piper-voices)
- [Sherpa-ONNX Piper Integration](https://github.com/k2-fsa/sherpa-onnx)
- [eSpeak-NG](https://github.com/espeak-ng/espeak-ng)
- [BenLubar espeak Go bindings](https://pkg.go.dev/gopkg.in/BenLubar/espeak.v2)
- [djangulo go-espeak](https://github.com/djangulo/go-espeak)

# Set shell for Windows compatibility
set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

# Platform-specific executable suffix
exe_suffix := if os() == "windows" { ".exe" } else { "" }

default: build

# Build the tts2go binary
build:
    @echo "Building tts2go..."
    @go build -o bin/tts2go{{exe_suffix}} ./cmd/tts2go
    @echo "Build complete: bin/tts2go{{exe_suffix}}"

# Build production binary (stripped, no debug info)
release: clean
    @echo "Building production tts2go..."
    @go build -trimpath -ldflags="-s -w" -o bin/tts2go{{exe_suffix}} ./cmd/tts2go
    @echo "Production build complete: bin/tts2go{{exe_suffix}}"

# Clean build artifacts
[unix]
clean:
    @echo "Cleaning..."
    @rm -rf bin/
    @go clean
    @echo "Clean complete"

[windows]
clean:
    @echo "Cleaning..."
    @if (Test-Path bin) { Remove-Item -Recurse -Force bin }
    @go clean
    @echo "Clean complete"

# Run tests
test:
    @echo "Running tests..."
    @go test -v ./...

# Download and verify dependencies
deps:
    @echo "Downloading dependencies..."
    @go mod download
    @go mod verify
    @go mod tidy
    @echo "Dependencies ready"

# Format code
fmt:
    @echo "Formatting code..."
    @go fmt ./...
    @echo "Format complete"

# ONNX Runtime version (must match onnxruntime_go version)
ort_version := "1.24.2"

# HuggingFace base URLs
hf_kitten := "https://huggingface.co/KittenML"
hf_kokoro := "https://huggingface.co/onnx-community/Kokoro-82M-ONNX"

# Fetch model files from HuggingFace
# Usage: just fetch-models [variant]
# Available variants:
#   nano-int8  - kitten-tts-nano-0.8-int8 (18 MB model, quantized, fastest)
#   nano-fp32  - kitten-tts-nano-0.8-fp32 (57 MB model, default)
#   micro      - kitten-tts-micro-0.8 (41 MB model)
#   mini       - kitten-tts-mini-0.8 (78 MB model, best quality)
[unix]
fetch-models variant="nano-fp32":
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p models
    case "{{variant}}" in
        nano-fp32)
            REPO="kitten-tts-nano-0.8-fp32"
            ONNX="kitten_tts_nano_v0_8.onnx"
            ;;
        nano-int8)
            REPO="kitten-tts-nano-0.8-int8"
            ONNX="kitten_tts_nano_v0_8.onnx"
            ;;
        micro)
            REPO="kitten-tts-micro-0.8"
            ONNX="kitten_tts_micro_v0_8.onnx"
            ;;
        mini)
            REPO="kitten-tts-mini-0.8"
            ONNX="kitten_tts_mini_v0_8.onnx"
            ;;
        *)
            echo "Unknown variant: {{variant}}"
            echo "Available: nano-int8, nano-fp32, micro, mini"
            exit 1
            ;;
    esac
    echo "Fetching $REPO model files..."
    echo "Downloading $ONNX -> models/model.onnx..."
    curl -L -o models/model.onnx "{{hf_kitten}}/$REPO/resolve/main/$ONNX"
    echo "Downloading voices.npz..."
    curl -L -o models/voices.npz "{{hf_kitten}}/$REPO/resolve/main/voices.npz"
    echo "Downloading config.json..."
    curl -L -o models/config.json "{{hf_kitten}}/$REPO/resolve/main/config.json"
    echo "Model files downloaded to models/"
    ls -lh models/

[windows]
fetch-models variant="nano-fp32":
    @echo "Fetching model files..."
    @New-Item -ItemType Directory -Force -Path models | Out-Null
    @$repo = switch ("{{variant}}") { \
        "nano-fp32" { @{repo="kitten-tts-nano-0.8-fp32"; onnx="kitten_tts_nano_v0_8.onnx"} } \
        "nano-int8" { @{repo="kitten-tts-nano-0.8-int8"; onnx="kitten_tts_nano_v0_8.onnx"} } \
        "micro" { @{repo="kitten-tts-micro-0.8"; onnx="kitten_tts_micro_v0_8.onnx"} } \
        "mini" { @{repo="kitten-tts-mini-0.8"; onnx="kitten_tts_mini_v0_8.onnx"} } \
        default { Write-Error "Unknown variant: {{variant}}"; exit 1 } \
    }; \
    Write-Host "Downloading $($repo.onnx) -> models/model.onnx..."; \
    Invoke-WebRequest -Uri "{{hf_kitten}}/$($repo.repo)/resolve/main/$($repo.onnx)" -OutFile models\model.onnx; \
    Write-Host "Downloading voices.npz..."; \
    Invoke-WebRequest -Uri "{{hf_kitten}}/$($repo.repo)/resolve/main/voices.npz" -OutFile models\voices.npz; \
    Write-Host "Downloading config.json..."; \
    Invoke-WebRequest -Uri "{{hf_kitten}}/$($repo.repo)/resolve/main/config.json" -OutFile models\config.json; \
    Write-Host "Model files downloaded to models/"; \
    Get-ChildItem models\

# Fetch Kokoro model files from HuggingFace
# Usage: just fetch-kokoro [variant]
# Available variants:
#   fp32   - model.onnx (326 MB, full precision)
#   fp16   - model_fp16.onnx (163 MB, half precision)
#   q8     - model_quantized.onnx (92 MB, 8-bit quantized, recommended)
#   q4f16  - model_q4f16.onnx (154 MB, 4-bit + fp16 hybrid)
[unix]
fetch-kokoro variant="q8":
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p models/voices
    case "{{variant}}" in
        fp32)
            ONNX="model.onnx"
            ;;
        fp16)
            ONNX="model_fp16.onnx"
            ;;
        q8)
            ONNX="model_quantized.onnx"
            ;;
        q4f16)
            ONNX="model_q4f16.onnx"
            ;;
        *)
            echo "Unknown variant: {{variant}}"
            echo "Available: fp32, fp16, q8, q4f16"
            exit 1
            ;;
    esac
    echo "Fetching Kokoro-82M-ONNX ($ONNX)..."
    echo "Downloading $ONNX -> models/model.onnx..."
    curl -L -o models/model.onnx "{{hf_kokoro}}/resolve/main/onnx/$ONNX"
    echo "Downloading voice files..."
    VOICES="af af_bella af_nicole af_sarah af_sky am_adam am_michael bf_emma bf_isabella bm_george bm_lewis"
    for voice in $VOICES; do
        echo "  -> $voice.bin"
        curl -sL -o "models/voices/${voice}.bin" "{{hf_kokoro}}/resolve/main/voices/${voice}.bin"
    done
    echo "Kokoro model files downloaded to models/"
    ls -lh models/
    ls -lh models/voices/

[windows]
fetch-kokoro variant="q8":
    @echo "Fetching Kokoro-82M-ONNX model files..."
    @New-Item -ItemType Directory -Force -Path models\voices | Out-Null
    @$onnx = switch ("{{variant}}") { \
        "fp32" { "model.onnx" } \
        "fp16" { "model_fp16.onnx" } \
        "q8" { "model_quantized.onnx" } \
        "q4f16" { "model_q4f16.onnx" } \
        default { Write-Error "Unknown variant: {{variant}}. Available: fp32, fp16, q8, q4f16"; exit 1 } \
    }; \
    Write-Host "Downloading $onnx -> models/model.onnx..."; \
    Invoke-WebRequest -Uri "{{hf_kokoro}}/resolve/main/onnx/$onnx" -OutFile models\model.onnx; \
    $voices = @("af", "af_bella", "af_nicole", "af_sarah", "af_sky", "am_adam", "am_michael", "bf_emma", "bf_isabella", "bm_george", "bm_lewis"); \
    foreach ($voice in $voices) { \
        Write-Host "  -> $voice.bin"; \
        Invoke-WebRequest -Uri "{{hf_kokoro}}/resolve/main/voices/$voice.bin" -OutFile "models\voices\$voice.bin" \
    }; \
    Write-Host "Kokoro model files downloaded to models/"; \
    Get-ChildItem models\; \
    Get-ChildItem models\voices\

# HuggingFace URLs for Kokoro v1.x multi-lang models
hf_kokoro_v1 := "https://huggingface.co/k2-fsa/sherpa-onnx-kokoro-en-v0_19"
hf_kokoro_v1_zh := "https://huggingface.co/k2-fsa/sherpa-onnx-kokoro-multi-lang-v1_0"
hf_kokoro_v11_zh := "https://huggingface.co/k2-fsa/sherpa-onnx-kokoro-multi-lang-v1_1"

# Fetch Kokoro v1.0/v1.1 multi-lang model files from HuggingFace
# Usage: just fetch-kokoro-v1 [version]
# Available versions:
#   v1.0  - Kokoro multi-lang v1.0 (Chinese + English, 53 speakers)
#   v1.1  - Kokoro multi-lang v1.1 (Chinese optimized, mixed zh/en)
[unix]
fetch-kokoro-v1 version="v1.0":
    #!/usr/bin/env bash
    set -euo pipefail
    case "{{version}}" in
        v1.0)
            MODEL_DIR="models/kokoro-v1.0"
            REPO_URL="{{hf_kokoro_v1_zh}}"
            ;;
        v1.1)
            MODEL_DIR="models/kokoro-v1.1"
            REPO_URL="{{hf_kokoro_v11_zh}}"
            ;;
        *)
            echo "Unknown version: {{version}}"
            echo "Available: v1.0, v1.1"
            exit 1
            ;;
    esac
    mkdir -p "$MODEL_DIR/voices"
    echo "Fetching Kokoro {{version}} multi-lang model..."
    echo "Downloading model.onnx..."
    curl -L -o "$MODEL_DIR/model.onnx" "$REPO_URL/resolve/main/model.onnx"
    echo "Downloading tokens.txt..."
    curl -L -o "$MODEL_DIR/tokens.txt" "$REPO_URL/resolve/main/tokens.txt"
    echo "Downloading voice files..."
    VOICES=$(curl -sL "$REPO_URL/tree/main/voices" | grep -oP '(?<=href=")[^"]*\.bin(?=")' | sed 's|.*/||' | sort -u)
    if [ -z "$VOICES" ]; then
        VOICES="af_alloy af_aoede af_bella af_jessica af_kore af_nicole af_nova af_river af_sarah af_sky am_adam am_echo am_eric am_fenrir am_liam am_michael am_onyx am_puck am_santa bf_alice bf_emma bf_lily bm_daniel bm_fable bm_george bm_lewis zf_xiaobei zf_xiaoni zf_xiaoxiao zf_xiaoyi zm_yunjian zm_yunxi zm_yunxia zm_yunyang"
    fi
    for voice in $VOICES; do
        voice_file="${voice%.bin}.bin"
        echo "  -> $voice_file"
        curl -sL -o "$MODEL_DIR/voices/$voice_file" "$REPO_URL/resolve/main/voices/$voice_file" || true
    done
    echo "Kokoro {{version}} downloaded to $MODEL_DIR/"
    ls -lh "$MODEL_DIR/"
    ls -lh "$MODEL_DIR/voices/" 2>/dev/null || true

[windows]
fetch-kokoro-v1 version="v1.0":
    @echo "Fetching Kokoro {{version}} multi-lang model..."
    @$modelDir = switch ("{{version}}") { \
        "v1.0" { "models\kokoro-v1.0"; $repoUrl = "{{hf_kokoro_v1_zh}}" } \
        "v1.1" { "models\kokoro-v1.1"; $repoUrl = "{{hf_kokoro_v11_zh}}" } \
        default { Write-Error "Unknown version: {{version}}. Available: v1.0, v1.1"; exit 1 } \
    }; \
    New-Item -ItemType Directory -Force -Path "$modelDir\voices" | Out-Null; \
    Write-Host "Downloading model.onnx..."; \
    Invoke-WebRequest -Uri "$repoUrl/resolve/main/model.onnx" -OutFile "$modelDir\model.onnx"; \
    Write-Host "Downloading tokens.txt..."; \
    Invoke-WebRequest -Uri "$repoUrl/resolve/main/tokens.txt" -OutFile "$modelDir\tokens.txt"; \
    $defaultVoices = @("af_alloy", "af_bella", "af_nicole", "af_sarah", "af_sky", "am_adam", "am_michael", "zf_xiaoxiao", "zm_yunxi"); \
    foreach ($voice in $defaultVoices) { \
        Write-Host "  -> $voice.bin"; \
        try { Invoke-WebRequest -Uri "$repoUrl/resolve/main/voices/$voice.bin" -OutFile "$modelDir\voices\$voice.bin" } catch {} \
    }; \
    Write-Host "Kokoro {{version}} downloaded to $modelDir/"; \
    Get-ChildItem "$modelDir\"

# HuggingFace URL for PocketTTS
hf_pockettts := "https://huggingface.co/KevinAHM/pocket-tts-onnx"

# Fetch PocketTTS model files from HuggingFace
# Usage: just fetch-pockettts [variant]
# Available variants:
#   fp32  - Full precision models
#   int8  - 8-bit quantized models (smaller, faster)
[unix]
fetch-pockettts variant="fp32":
    #!/usr/bin/env bash
    set -euo pipefail
    MODEL_DIR="models/pockettts"
    mkdir -p "$MODEL_DIR"
    echo "Fetching PocketTTS ({{variant}})..."
    echo "Downloading text_conditioner.onnx..."
    curl -L -o "$MODEL_DIR/text_conditioner.onnx" "{{hf_pockettts}}/resolve/main/text_conditioner.onnx"
    echo "Downloading encoder.onnx..."
    curl -L -o "$MODEL_DIR/encoder.onnx" "{{hf_pockettts}}/resolve/main/encoder.onnx"
    case "{{variant}}" in
        fp32)
            echo "Downloading lm_main.onnx..."
            curl -L -o "$MODEL_DIR/lm_main.onnx" "{{hf_pockettts}}/resolve/main/lm_main.onnx"
            echo "Downloading lm_flow.onnx..."
            curl -L -o "$MODEL_DIR/lm_flow.onnx" "{{hf_pockettts}}/resolve/main/lm_flow.onnx"
            echo "Downloading decoder.onnx..."
            curl -L -o "$MODEL_DIR/decoder.onnx" "{{hf_pockettts}}/resolve/main/decoder.onnx"
            ;;
        int8)
            echo "Downloading lm_main_int8.onnx..."
            curl -L -o "$MODEL_DIR/lm_main_int8.onnx" "{{hf_pockettts}}/resolve/main/lm_main_int8.onnx"
            echo "Downloading lm_flow_int8.onnx..."
            curl -L -o "$MODEL_DIR/lm_flow_int8.onnx" "{{hf_pockettts}}/resolve/main/lm_flow_int8.onnx"
            echo "Downloading decoder_int8.onnx..."
            curl -L -o "$MODEL_DIR/decoder_int8.onnx" "{{hf_pockettts}}/resolve/main/decoder_int8.onnx"
            ;;
        *)
            echo "Unknown variant: {{variant}}"
            echo "Available: fp32, int8"
            exit 1
            ;;
    esac
    echo "Downloading vocab.json..."
    curl -L -o "$MODEL_DIR/vocab.json" "{{hf_pockettts}}/resolve/main/vocab.json"
    echo "Downloading token_scores.json..."
    curl -L -o "$MODEL_DIR/token_scores.json" "{{hf_pockettts}}/resolve/main/token_scores.json" || true
    echo "PocketTTS downloaded to $MODEL_DIR/"
    ls -lh "$MODEL_DIR/"

[windows]
fetch-pockettts variant="fp32":
    @echo "Fetching PocketTTS ({{variant}})..."
    @$modelDir = "models\pockettts"; \
    New-Item -ItemType Directory -Force -Path $modelDir | Out-Null; \
    Write-Host "Downloading text_conditioner.onnx..."; \
    Invoke-WebRequest -Uri "{{hf_pockettts}}/resolve/main/text_conditioner.onnx" -OutFile "$modelDir\text_conditioner.onnx"; \
    Write-Host "Downloading encoder.onnx..."; \
    Invoke-WebRequest -Uri "{{hf_pockettts}}/resolve/main/encoder.onnx" -OutFile "$modelDir\encoder.onnx"; \
    switch ("{{variant}}") { \
        "fp32" { \
            Write-Host "Downloading lm_main.onnx..."; \
            Invoke-WebRequest -Uri "{{hf_pockettts}}/resolve/main/lm_main.onnx" -OutFile "$modelDir\lm_main.onnx"; \
            Write-Host "Downloading lm_flow.onnx..."; \
            Invoke-WebRequest -Uri "{{hf_pockettts}}/resolve/main/lm_flow.onnx" -OutFile "$modelDir\lm_flow.onnx"; \
            Write-Host "Downloading decoder.onnx..."; \
            Invoke-WebRequest -Uri "{{hf_pockettts}}/resolve/main/decoder.onnx" -OutFile "$modelDir\decoder.onnx" \
        } \
        "int8" { \
            Write-Host "Downloading lm_main_int8.onnx..."; \
            Invoke-WebRequest -Uri "{{hf_pockettts}}/resolve/main/lm_main_int8.onnx" -OutFile "$modelDir\lm_main_int8.onnx"; \
            Write-Host "Downloading lm_flow_int8.onnx..."; \
            Invoke-WebRequest -Uri "{{hf_pockettts}}/resolve/main/lm_flow_int8.onnx" -OutFile "$modelDir\lm_flow_int8.onnx"; \
            Write-Host "Downloading decoder_int8.onnx..."; \
            Invoke-WebRequest -Uri "{{hf_pockettts}}/resolve/main/decoder_int8.onnx" -OutFile "$modelDir\decoder_int8.onnx" \
        } \
        default { Write-Error "Unknown variant: {{variant}}. Available: fp32, int8"; exit 1 } \
    }; \
    Write-Host "Downloading vocab.json..."; \
    Invoke-WebRequest -Uri "{{hf_pockettts}}/resolve/main/vocab.json" -OutFile "$modelDir\vocab.json"; \
    try { Invoke-WebRequest -Uri "{{hf_pockettts}}/resolve/main/token_scores.json" -OutFile "$modelDir\token_scores.json" } catch {}; \
    Write-Host "PocketTTS downloaded to $modelDir/"; \
    Get-ChildItem "$modelDir\"

# Full rebuild
rebuild: clean build

# Download ONNX Runtime library (Linux x64)
[unix]
fetch-onnxruntime:
    @echo "Downloading ONNX Runtime {{ort_version}} for Linux x64..."
    @mkdir -p lib
    @curl -L -o /tmp/onnxruntime.tgz \
        "https://github.com/microsoft/onnxruntime/releases/download/v{{ort_version}}/onnxruntime-linux-x64-{{ort_version}}.tgz"
    @tar -xzf /tmp/onnxruntime.tgz -C /tmp
    @cp /tmp/onnxruntime-linux-x64-{{ort_version}}/lib/libonnxruntime.so.{{ort_version}} lib/
    @ln -sf libonnxruntime.so.{{ort_version}} lib/libonnxruntime.so
    @rm -rf /tmp/onnxruntime.tgz /tmp/onnxruntime-linux-x64-{{ort_version}}
    @echo "ONNX Runtime installed to lib/"
    @echo "Set: export ONNXRUNTIME_LIB_PATH={{justfile_directory()}}/lib/libonnxruntime.so"

# Download ONNX Runtime library (Windows x64)
[windows]
fetch-onnxruntime:
    @echo "Downloading ONNX Runtime {{ort_version}} for Windows x64..."
    @New-Item -ItemType Directory -Force -Path lib | Out-Null
    @Invoke-WebRequest -Uri "https://github.com/microsoft/onnxruntime/releases/download/v{{ort_version}}/onnxruntime-win-x64-{{ort_version}}.zip" -OutFile "$env:TEMP\onnxruntime.zip"
    @Expand-Archive -Path "$env:TEMP\onnxruntime.zip" -DestinationPath "$env:TEMP\onnxruntime" -Force
    @Copy-Item "$env:TEMP\onnxruntime\onnxruntime-win-x64-{{ort_version}}\lib\onnxruntime.dll" -Destination lib\
    @Remove-Item -Recurse -Force "$env:TEMP\onnxruntime.zip", "$env:TEMP\onnxruntime"
    @echo "ONNX Runtime installed to lib/"
    @echo "Set: $env:ONNXRUNTIME_LIB_PATH='{{justfile_directory()}}\lib\onnxruntime.dll'"

# Run tts2go with local ONNX Runtime
[unix]
run +ARGS:
    #!/usr/bin/env bash
    export ONNXRUNTIME_LIB_PATH={{justfile_directory()}}/lib/libonnxruntime.so
    ./bin/tts2go {{ARGS}}

# Run tts2go with local ONNX Runtime
[windows]
run +ARGS:
    @$env:ONNXRUNTIME_LIB_PATH='{{justfile_directory()}}\lib\onnxruntime.dll'; ./bin/tts2go {{ARGS}}

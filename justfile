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
hf_kokoro_1_0 := "https://huggingface.co/csukuangfj/kokoro-multi-lang-v1_0"
hf_kokoro_1_1 := "https://huggingface.co/csukuangfj/kokoro-multi-lang-v1_1"
hf_pocket := "https://huggingface.co/csukuangfj2/sherpa-onnx-pocket-tts-2026-01-26"
hf_pocket_int8 := "https://huggingface.co/csukuangfj2/sherpa-onnx-pocket-tts-int8-2026-01-26"

# Fetch Kitten TTS model files
# Usage: just fetch-kitten [variant]
# Variants: nano-int8, nano-fp32 (default), micro, mini
[unix]
fetch-kitten variant="nano-fp32":
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf models
    mkdir -p models
    V="{{variant}}"
    V="${V#variant=}"
    case "$V" in
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
            echo "Unknown variant: $V"
            echo "Available: nano-int8, nano-fp32, micro, mini"
            exit 1
            ;;
    esac
    echo "Fetching Kitten TTS ($REPO)..."
    curl -L -o models/model.onnx "{{hf_kitten}}/$REPO/resolve/main/$ONNX"
    curl -L -o models/voices.npz "{{hf_kitten}}/$REPO/resolve/main/voices.npz"
    echo "Done: models/"
    ls -lh models/

[windows]
fetch-kitten variant="nano-fp32":
    @Remove-Item -Recurse -Force models -ErrorAction SilentlyContinue
    @New-Item -ItemType Directory -Force -Path models | Out-Null
    @$repo = switch ("{{variant}}") { \
        "nano-fp32" { @{repo="kitten-tts-nano-0.8-fp32"; onnx="kitten_tts_nano_v0_8.onnx"} } \
        "nano-int8" { @{repo="kitten-tts-nano-0.8-int8"; onnx="kitten_tts_nano_v0_8.onnx"} } \
        "micro" { @{repo="kitten-tts-micro-0.8"; onnx="kitten_tts_micro_v0_8.onnx"} } \
        "mini" { @{repo="kitten-tts-mini-0.8"; onnx="kitten_tts_mini_v0_8.onnx"} } \
        default { Write-Error "Unknown variant: {{variant}}"; exit 1 } \
    }; \
    Invoke-WebRequest -Uri "{{hf_kitten}}/$($repo.repo)/resolve/main/$($repo.onnx)" -OutFile models\model.onnx; \
    Invoke-WebRequest -Uri "{{hf_kitten}}/$($repo.repo)/resolve/main/voices.npz" -OutFile models\voices.npz; \
    Get-ChildItem models\

# Fetch Kokoro TTS model files (original English-only)
# Usage: just fetch-kokoro [variant]
# Variants: fp32, fp16, q8 (default), q4f16
[unix]
fetch-kokoro variant="q8":
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf models
    mkdir -p models/voices
    V="{{variant}}"
    V="${V#variant=}"
    case "$V" in
        fp32)   ONNX="model.onnx" ;;
        fp16)   ONNX="model_fp16.onnx" ;;
        q8)     ONNX="model_quantized.onnx" ;;
        q4f16)  ONNX="model_q4f16.onnx" ;;
        *)
            echo "Unknown variant: $V"
            echo "Available: fp32, fp16, q8, q4f16"
            exit 1
            ;;
    esac
    echo "Fetching Kokoro ($ONNX)..."
    curl -L -o models/model.onnx "{{hf_kokoro}}/resolve/main/onnx/$ONNX"
    for voice in af af_bella af_nicole af_sarah af_sky am_adam am_michael bf_emma bf_isabella bm_george bm_lewis; do
        curl -sL -o "models/voices/${voice}.bin" "{{hf_kokoro}}/resolve/main/voices/${voice}.bin"
    done
    echo "Done: models/"
    ls -lh models/

[windows]
fetch-kokoro variant="q8":
    @Remove-Item -Recurse -Force models -ErrorAction SilentlyContinue
    @New-Item -ItemType Directory -Force -Path models\voices | Out-Null
    @$onnx = switch ("{{variant}}") { \
        "fp32" { "model.onnx" } \
        "fp16" { "model_fp16.onnx" } \
        "q8" { "model_quantized.onnx" } \
        "q4f16" { "model_q4f16.onnx" } \
        default { Write-Error "Unknown variant"; exit 1 } \
    }; \
    Invoke-WebRequest -Uri "{{hf_kokoro}}/resolve/main/onnx/$onnx" -OutFile models\model.onnx; \
    @("af","af_bella","af_nicole","af_sarah","af_sky","am_adam","am_michael","bf_emma","bf_isabella","bm_george","bm_lewis") | ForEach-Object { \
        Invoke-WebRequest -Uri "{{hf_kokoro}}/resolve/main/voices/$_.bin" -OutFile "models\voices\$_.bin" \
    }; \
    Get-ChildItem models\

# Fetch Kokoro 1.0 multi-lang model (Chinese + English)
[unix]
fetch-kokoro-1_0:
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf models
    mkdir -p models
    echo "Fetching Kokoro 1.0 multi-lang..."
    curl -L -o models/model.onnx "{{hf_kokoro_1_0}}/resolve/main/model.onnx"
    curl -L -o models/tokens.txt "{{hf_kokoro_1_0}}/resolve/main/tokens.txt"
    curl -L -o models/voices.bin "{{hf_kokoro_1_0}}/resolve/main/voices.bin"
    echo "1.0" > models/.version
    echo "Done: models/"
    ls -lh models/

[windows]
fetch-kokoro-1_0:
    @Remove-Item -Recurse -Force models -ErrorAction SilentlyContinue
    @New-Item -ItemType Directory -Force -Path models | Out-Null
    @Invoke-WebRequest -Uri "{{hf_kokoro_1_0}}/resolve/main/model.onnx" -OutFile models\model.onnx
    @Invoke-WebRequest -Uri "{{hf_kokoro_1_0}}/resolve/main/tokens.txt" -OutFile models\tokens.txt
    @Invoke-WebRequest -Uri "{{hf_kokoro_1_0}}/resolve/main/voices.bin" -OutFile models\voices.bin
    @"1.0" | Out-File -FilePath models\.version -Encoding ascii
    @Get-ChildItem models\

# Fetch Kokoro 1.1 multi-lang model (Chinese optimized)
[unix]
fetch-kokoro-1_1:
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf models
    mkdir -p models
    echo "Fetching Kokoro 1.1 multi-lang..."
    curl -L -o models/model.onnx "{{hf_kokoro_1_1}}/resolve/main/model.onnx"
    curl -L -o models/tokens.txt "{{hf_kokoro_1_1}}/resolve/main/tokens.txt"
    curl -L -o models/voices.bin "{{hf_kokoro_1_1}}/resolve/main/voices.bin"
    echo "1.1" > models/.version
    echo "Done: models/"
    ls -lh models/

[windows]
fetch-kokoro-1_1:
    @Remove-Item -Recurse -Force models -ErrorAction SilentlyContinue
    @New-Item -ItemType Directory -Force -Path models | Out-Null
    @Invoke-WebRequest -Uri "{{hf_kokoro_1_1}}/resolve/main/model.onnx" -OutFile models\model.onnx
    @Invoke-WebRequest -Uri "{{hf_kokoro_1_1}}/resolve/main/tokens.txt" -OutFile models\tokens.txt
    @Invoke-WebRequest -Uri "{{hf_kokoro_1_1}}/resolve/main/voices.bin" -OutFile models\voices.bin
    @"1.1" | Out-File -FilePath models\.version -Encoding ascii
    @Get-ChildItem models\

# Fetch PocketTTS model files (voice cloning)
# Usage: just fetch-pocket [variant]
# Variants: fp32 (default), int8
[unix]
fetch-pocket variant="fp32":
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf models
    mkdir -p models
    V="{{variant}}"
    V="${V#variant=}"
    case "$V" in
        fp32)
            REPO="{{hf_pocket}}"
            ;;
        int8)
            REPO="{{hf_pocket_int8}}"
            ;;
        *)
            echo "Unknown variant: $V"
            echo "Available: fp32, int8"
            exit 1
            ;;
    esac
    echo "Fetching PocketTTS ($V)..."
    curl -L -o models/text_conditioner.onnx "$REPO/resolve/main/text_conditioner.onnx"
    curl -L -o models/encoder.onnx "$REPO/resolve/main/encoder.onnx"
    curl -L -o models/lm_main.onnx "$REPO/resolve/main/lm_main.onnx"
    curl -L -o models/lm_flow.onnx "$REPO/resolve/main/lm_flow.onnx"
    curl -L -o models/decoder.onnx "$REPO/resolve/main/decoder.onnx"
    curl -L -o models/vocab.json "$REPO/resolve/main/vocab.json"
    curl -L -o models/token_scores.json "$REPO/resolve/main/token_scores.json"
    echo "Done: models/"
    ls -lh models/

[windows]
fetch-pocket variant="fp32":
    @Remove-Item -Recurse -Force models -ErrorAction SilentlyContinue
    @New-Item -ItemType Directory -Force -Path models | Out-Null
    @$repo = if ("{{variant}}" -eq "int8") { "{{hf_pocket_int8}}" } else { "{{hf_pocket}}" }
    @Invoke-WebRequest -Uri "$repo/resolve/main/text_conditioner.onnx" -OutFile models\text_conditioner.onnx
    @Invoke-WebRequest -Uri "$repo/resolve/main/encoder.onnx" -OutFile models\encoder.onnx
    @Invoke-WebRequest -Uri "$repo/resolve/main/lm_main.onnx" -OutFile models\lm_main.onnx
    @Invoke-WebRequest -Uri "$repo/resolve/main/lm_flow.onnx" -OutFile models\lm_flow.onnx
    @Invoke-WebRequest -Uri "$repo/resolve/main/decoder.onnx" -OutFile models\decoder.onnx
    @Invoke-WebRequest -Uri "$repo/resolve/main/vocab.json" -OutFile models\vocab.json
    @Invoke-WebRequest -Uri "$repo/resolve/main/token_scores.json" -OutFile models\token_scores.json
    @Get-ChildItem models\

# Full rebuild
rebuild: clean build

# Download ONNX Runtime library (Linux x64)
[unix]
fetch-onnxruntime:
    @echo "Downloading ONNX Runtime {{ort_version}}..."
    @mkdir -p lib
    @curl -L -o /tmp/onnxruntime.tgz \
        "https://github.com/microsoft/onnxruntime/releases/download/v{{ort_version}}/onnxruntime-linux-x64-{{ort_version}}.tgz"
    @tar -xzf /tmp/onnxruntime.tgz -C /tmp
    @cp /tmp/onnxruntime-linux-x64-{{ort_version}}/lib/libonnxruntime.so.{{ort_version}} lib/
    @ln -sf libonnxruntime.so.{{ort_version}} lib/libonnxruntime.so
    @rm -rf /tmp/onnxruntime.tgz /tmp/onnxruntime-linux-x64-{{ort_version}}
    @echo "Done: lib/libonnxruntime.so"

# Download ONNX Runtime library (Windows x64)
[windows]
fetch-onnxruntime:
    @New-Item -ItemType Directory -Force -Path lib | Out-Null
    @Invoke-WebRequest -Uri "https://github.com/microsoft/onnxruntime/releases/download/v{{ort_version}}/onnxruntime-win-x64-{{ort_version}}.zip" -OutFile "$env:TEMP\onnxruntime.zip"
    @Expand-Archive -Path "$env:TEMP\onnxruntime.zip" -DestinationPath "$env:TEMP\onnxruntime" -Force
    @Copy-Item "$env:TEMP\onnxruntime\onnxruntime-win-x64-{{ort_version}}\lib\onnxruntime.dll" -Destination lib\
    @Remove-Item -Recurse -Force "$env:TEMP\onnxruntime.zip", "$env:TEMP\onnxruntime"
    @echo "Done: lib\onnxruntime.dll"

# Run tts2go with local ONNX Runtime
[unix]
run +ARGS:
    #!/usr/bin/env bash
    export ONNXRUNTIME_LIB_PATH={{justfile_directory()}}/lib/libonnxruntime.so
    ./bin/tts2go {{ARGS}}

[windows]
run +ARGS:
    @$env:ONNXRUNTIME_LIB_PATH='{{justfile_directory()}}\lib\onnxruntime.dll'; ./bin/tts2go {{ARGS}}

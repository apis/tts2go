# Set shell for Windows compatibility
set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

# Platform-specific executable suffix
exe_suffix := if os() == "windows" { ".exe" } else { "" }

default: build

# Build the kittentts binary
build:
    @echo "Building kittentts..."
    @go build -o bin/kittentts{{exe_suffix}} ./cmd/kittentts
    @echo "Build complete: bin/kittentts{{exe_suffix}}"

# Build production binary (stripped, no debug info)
release: clean
    @echo "Building production kittentts..."
    @go build -trimpath -ldflags="-s -w" -o bin/kittentts{{exe_suffix}} ./cmd/kittentts
    @echo "Production build complete: bin/kittentts{{exe_suffix}}"

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

# HuggingFace base URL
hf_base := "https://huggingface.co/KittenML"

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
    curl -L -o models/model.onnx "{{hf_base}}/$REPO/resolve/main/$ONNX"
    echo "Downloading voices.npz..."
    curl -L -o models/voices.npz "{{hf_base}}/$REPO/resolve/main/voices.npz"
    echo "Downloading config.json..."
    curl -L -o models/config.json "{{hf_base}}/$REPO/resolve/main/config.json"
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
    Invoke-WebRequest -Uri "{{hf_base}}/$($repo.repo)/resolve/main/$($repo.onnx)" -OutFile models\model.onnx; \
    Write-Host "Downloading voices.npz..."; \
    Invoke-WebRequest -Uri "{{hf_base}}/$($repo.repo)/resolve/main/voices.npz" -OutFile models\voices.npz; \
    Write-Host "Downloading config.json..."; \
    Invoke-WebRequest -Uri "{{hf_base}}/$($repo.repo)/resolve/main/config.json" -OutFile models\config.json; \
    Write-Host "Model files downloaded to models/"; \
    Get-ChildItem models\

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

# Run kittentts with local ONNX Runtime
[unix]
run +ARGS:
    #!/usr/bin/env bash
    export ONNXRUNTIME_LIB_PATH={{justfile_directory()}}/lib/libonnxruntime.so
    ./bin/kittentts {{ARGS}}

# Run kittentts with local ONNX Runtime
[windows]
run +ARGS:
    @$env:ONNXRUNTIME_LIB_PATH='{{justfile_directory()}}\lib\onnxruntime.dll'; ./bin/kittentts {{ARGS}}

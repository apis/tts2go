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

# ONNX Runtime version
ort_version := "1.20.1"

# Model URL (HuggingFace)
hf_repo := "KittenML/kitten-tts-nano-0.2"

# Fetch model files from HuggingFace
[unix]
fetch-models:
    @echo "Fetching model files..."
    @mkdir -p models
    @echo "Download model.onnx, voices.npz, config.json from:"
    @echo "https://huggingface.co/{{hf_repo}}/tree/main"
    @echo "Place them in the models/ directory"

[windows]
fetch-models:
    @echo "Fetching model files..."
    @New-Item -ItemType Directory -Force -Path models | Out-Null
    @echo "Download model.onnx, voices.npz, config.json from:"
    @echo "https://huggingface.co/{{hf_repo}}/tree/main"
    @echo "Place them in the models/ directory"

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
run *ARGS:
    @ONNXRUNTIME_LIB_PATH={{justfile_directory()}}/lib/libonnxruntime.so ./bin/kittentts {{ARGS}}

# Run kittentts with local ONNX Runtime
[windows]
run *ARGS:
    @$env:ONNXRUNTIME_LIB_PATH='{{justfile_directory()}}\lib\onnxruntime.dll'; ./bin/kittentts {{ARGS}}

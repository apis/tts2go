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

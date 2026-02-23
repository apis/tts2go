# CLAUDE.md

## All documents should be stored in the `docs` folder

## Don't write any comments in Go code unless specifically told to do so
## You could create TODO comments in Go code to remind yourself to update something later
## For existing comments in Go code keep them up to date in sync with the code

## Project structure should follow the standard Go project layout

## Build Commands (justfile)

- `just build` - Build the tts2go binary to `bin/tts2go`
- `just clean` - Remove build artifacts
- `just test` - Run tests
- `just deps` - Download and verify Go dependencies
- `just fmt` - Format Go code
- `just fetch-models` - Download model files from HuggingFace
- `just fetch-onnxruntime` - Download ONNX Runtime library to `lib/`
- `just rebuild` - Full rebuild: clean and build
- `just run` - Run tts2go with local ONNX Runtime (auto-sets ONNXRUNTIME_LIB_PATH)

# TTS2Go Implementation Plan

See the full implementation plan in the project conversation history.

## Quick Start

1. Run `just fetch-onnxruntime` to download ONNX Runtime
2. Run `just fetch-models` to download Kitten TTS models (or `just fetch-kokoro` for Kokoro)
3. Run `just deps` to install Go dependencies
4. Run `just build` to build the binary
5. Run `just run -t '"Hello world"' -o test.wav`

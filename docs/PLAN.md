# KittenTTS Go Implementation Plan

See the full implementation plan in the project conversation history.

## Quick Start

1. Download models from HuggingFace: https://huggingface.co/KittenML/kitten-tts-nano-0.2
2. Place `model.onnx`, `voices.npz`, and `config.json` in `models/` directory
3. Run `just deps` to install dependencies
4. Run `just build` to build the binary
5. Run `./bin/kittentts -t "Hello world" -o test.wav`

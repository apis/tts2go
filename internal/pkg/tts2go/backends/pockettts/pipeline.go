package pockettts

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"

	ort "github.com/yalue/onnxruntime_go"
)

type Pipeline struct {
	modelDir        string
	textConditioner *ort.DynamicAdvancedSession
	lmMain          *ort.DynamicAdvancedSession
	lmFlow          *ort.DynamicAdvancedSession
	encoder         *ort.DynamicAdvancedSession
	decoder         *ort.DynamicAdvancedSession
	initialized     bool
}

func getOnnxRuntimeLibPath() string {
	envPath := os.Getenv("ONNXRUNTIME_LIB_PATH")
	if envPath != "" {
		return envPath
	}

	switch runtime.GOOS {
	case "linux":
		paths := []string{
			"/usr/lib/libonnxruntime.so",
			"/usr/local/lib/libonnxruntime.so",
			"./libonnxruntime.so",
			"./lib/libonnxruntime.so",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return "libonnxruntime.so"
	case "windows":
		paths := []string{
			"onnxruntime.dll",
			"./onnxruntime.dll",
			"./lib/onnxruntime.dll",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return "onnxruntime.dll"
	case "darwin":
		paths := []string{
			"/usr/local/lib/libonnxruntime.dylib",
			"/opt/homebrew/lib/libonnxruntime.dylib",
			"./libonnxruntime.dylib",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return "libonnxruntime.dylib"
	default:
		return "libonnxruntime.so"
	}
}

func NewPipeline(modelDir string, useInt8 bool) (*Pipeline, error) {
	libPath := getOnnxRuntimeLibPath()
	ort.SetSharedLibraryPath(libPath)

	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX runtime: %w", err)
	}

	p := &Pipeline{
		modelDir: modelDir,
	}

	suffix := ""
	if useInt8 {
		suffix = "_int8"
	}

	textConditionerPath := filepath.Join(modelDir, "text_conditioner.onnx")
	lmMainPath := filepath.Join(modelDir, "lm_main"+suffix+".onnx")
	lmFlowPath := filepath.Join(modelDir, "lm_flow"+suffix+".onnx")
	encoderPath := filepath.Join(modelDir, "encoder.onnx")
	decoderPath := filepath.Join(modelDir, "decoder"+suffix+".onnx")

	if _, err := os.Stat(lmMainPath); os.IsNotExist(err) {
		lmMainPath = filepath.Join(modelDir, "lm_main.onnx")
	}
	if _, err := os.Stat(lmFlowPath); os.IsNotExist(err) {
		lmFlowPath = filepath.Join(modelDir, "lm_flow.onnx")
	}
	if _, err := os.Stat(decoderPath); os.IsNotExist(err) {
		decoderPath = filepath.Join(modelDir, "decoder.onnx")
	}

	var err error

	p.textConditioner, err = ort.NewDynamicAdvancedSession(
		textConditionerPath,
		[]string{"input_ids"},
		[]string{"text_embeds"},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load text_conditioner: %w", err)
	}

	p.lmMain, err = ort.NewDynamicAdvancedSession(
		lmMainPath,
		[]string{"text_embeds", "audio_embeds", "cond_scale"},
		[]string{"conditioning"},
		nil,
	)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to load lm_main: %w", err)
	}

	p.lmFlow, err = ort.NewDynamicAdvancedSession(
		lmFlowPath,
		[]string{"conditioning", "latents", "timestep"},
		[]string{"output"},
		nil,
	)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to load lm_flow: %w", err)
	}

	p.encoder, err = ort.NewDynamicAdvancedSession(
		encoderPath,
		[]string{"audio"},
		[]string{"audio_embeds"},
		nil,
	)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to load encoder: %w", err)
	}

	p.decoder, err = ort.NewDynamicAdvancedSession(
		decoderPath,
		[]string{"latents"},
		[]string{"audio"},
		nil,
	)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to load decoder: %w", err)
	}

	p.initialized = true
	return p, nil
}

func (p *Pipeline) EncodeReference(audio []float32) ([]float32, error) {
	audioTensor, err := ort.NewTensor(ort.NewShape(1, 1, int64(len(audio))), audio)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio tensor: %w", err)
	}
	defer audioTensor.Destroy()

	inputs := []ort.Value{audioTensor}
	outputs := make([]ort.Value, 1)

	if err := p.encoder.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("failed to run encoder: %w", err)
	}

	if outputs[0] == nil {
		return nil, fmt.Errorf("no output from encoder")
	}
	defer outputs[0].Destroy()

	outputTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected encoder output type")
	}

	return outputTensor.GetData(), nil
}

func (p *Pipeline) Generate(textEmbeds, audioEmbeds []float32, textLen, audioLen int64, speed float32) ([]float32, error) {
	textEmbedsTensor, err := ort.NewTensor(ort.NewShape(1, textLen, int64(len(textEmbeds))/textLen), textEmbeds)
	if err != nil {
		return nil, fmt.Errorf("failed to create text_embeds tensor: %w", err)
	}
	defer textEmbedsTensor.Destroy()

	audioEmbedsTensor, err := ort.NewTensor(ort.NewShape(1, audioLen, int64(len(audioEmbeds))/audioLen), audioEmbeds)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio_embeds tensor: %w", err)
	}
	defer audioEmbedsTensor.Destroy()

	condScale := []float32{1.0 / speed}
	condScaleTensor, err := ort.NewTensor(ort.NewShape(1), condScale)
	if err != nil {
		return nil, fmt.Errorf("failed to create cond_scale tensor: %w", err)
	}
	defer condScaleTensor.Destroy()

	lmMainInputs := []ort.Value{textEmbedsTensor, audioEmbedsTensor, condScaleTensor}
	lmMainOutputs := make([]ort.Value, 1)

	if err := p.lmMain.Run(lmMainInputs, lmMainOutputs); err != nil {
		return nil, fmt.Errorf("failed to run lm_main: %w", err)
	}

	if lmMainOutputs[0] == nil {
		return nil, fmt.Errorf("no output from lm_main")
	}
	defer lmMainOutputs[0].Destroy()

	condTensor, ok := lmMainOutputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected lm_main output type")
	}
	condData := condTensor.GetData()
	condShape := condTensor.GetShape()

	latents, err := p.runODESolver(condData, condShape)
	if err != nil {
		return nil, fmt.Errorf("failed to run ODE solver: %w", err)
	}

	latentsTensor, err := ort.NewTensor(ort.NewShape(1, int64(len(latents)/128), 128), latents)
	if err != nil {
		return nil, fmt.Errorf("failed to create latents tensor: %w", err)
	}
	defer latentsTensor.Destroy()

	decoderInputs := []ort.Value{latentsTensor}
	decoderOutputs := make([]ort.Value, 1)

	if err := p.decoder.Run(decoderInputs, decoderOutputs); err != nil {
		return nil, fmt.Errorf("failed to run decoder: %w", err)
	}

	if decoderOutputs[0] == nil {
		return nil, fmt.Errorf("no output from decoder")
	}
	defer decoderOutputs[0].Destroy()

	audioTensor, ok := decoderOutputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected decoder output type")
	}

	return audioTensor.GetData(), nil
}

func (p *Pipeline) runODESolver(conditioning []float32, condShape []int64) ([]float32, error) {
	numSteps := 32
	dt := 1.0 / float32(numSteps)

	latentDim := 128
	seqLen := int(condShape[1])
	latents := make([]float32, seqLen*latentDim)
	for i := range latents {
		latents[i] = float32(randNormal())
	}

	for step := 0; step < numSteps; step++ {
		t := float32(step) / float32(numSteps)

		condTensor, err := ort.NewTensor(condShape, conditioning)
		if err != nil {
			return nil, fmt.Errorf("failed to create conditioning tensor: %w", err)
		}

		latentsTensor, err := ort.NewTensor(ort.NewShape(1, int64(seqLen), int64(latentDim)), latents)
		if err != nil {
			condTensor.Destroy()
			return nil, fmt.Errorf("failed to create latents tensor: %w", err)
		}

		timestepTensor, err := ort.NewTensor(ort.NewShape(1), []float32{t})
		if err != nil {
			condTensor.Destroy()
			latentsTensor.Destroy()
			return nil, fmt.Errorf("failed to create timestep tensor: %w", err)
		}

		inputs := []ort.Value{condTensor, latentsTensor, timestepTensor}
		outputs := make([]ort.Value, 1)

		if err := p.lmFlow.Run(inputs, outputs); err != nil {
			condTensor.Destroy()
			latentsTensor.Destroy()
			timestepTensor.Destroy()
			return nil, fmt.Errorf("failed to run lm_flow at step %d: %w", step, err)
		}

		condTensor.Destroy()
		latentsTensor.Destroy()
		timestepTensor.Destroy()

		if outputs[0] == nil {
			return nil, fmt.Errorf("no output from lm_flow at step %d", step)
		}

		velTensor, ok := outputs[0].(*ort.Tensor[float32])
		if !ok {
			outputs[0].Destroy()
			return nil, fmt.Errorf("unexpected lm_flow output type")
		}

		velocity := velTensor.GetData()
		for i := range latents {
			latents[i] += velocity[i] * dt
		}
		outputs[0].Destroy()
	}

	return latents, nil
}

func (p *Pipeline) GetTextEmbeddings(inputIDs []int64) ([]float32, error) {
	inputTensor, err := ort.NewTensor(ort.NewShape(1, int64(len(inputIDs))), inputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputTensor.Destroy()

	inputs := []ort.Value{inputTensor}
	outputs := make([]ort.Value, 1)

	if err := p.textConditioner.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("failed to run text_conditioner: %w", err)
	}

	if outputs[0] == nil {
		return nil, fmt.Errorf("no output from text_conditioner")
	}
	defer outputs[0].Destroy()

	outputTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected text_conditioner output type")
	}

	return outputTensor.GetData(), nil
}

func (p *Pipeline) Close() error {
	var lastErr error

	if p.textConditioner != nil {
		if err := p.textConditioner.Destroy(); err != nil {
			lastErr = err
		}
	}
	if p.lmMain != nil {
		if err := p.lmMain.Destroy(); err != nil {
			lastErr = err
		}
	}
	if p.lmFlow != nil {
		if err := p.lmFlow.Destroy(); err != nil {
			lastErr = err
		}
	}
	if p.encoder != nil {
		if err := p.encoder.Destroy(); err != nil {
			lastErr = err
		}
	}
	if p.decoder != nil {
		if err := p.decoder.Destroy(); err != nil {
			lastErr = err
		}
	}

	if p.initialized {
		if err := ort.DestroyEnvironment(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

var rngState uint64 = 42

func randNormal() float64 {
	rngState ^= rngState << 13
	rngState ^= rngState >> 7
	rngState ^= rngState << 17

	u1 := float64(rngState) / float64(^uint64(0))
	rngState ^= rngState << 13
	rngState ^= rngState >> 7
	rngState ^= rngState << 17
	u2 := float64(rngState) / float64(^uint64(0))

	return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

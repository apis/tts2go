package pockettts

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"

	ort "github.com/yalue/onnxruntime_go"
)

const (
	lmMainStateCount  = 18
	decoderStateCount = 56
	seqFeatureDim     = 32
	hiddenDim         = 1024
	numFlowSteps      = 32
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
		[]string{"token_ids"},
		[]string{"embeddings"},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load text_conditioner: %w", err)
	}

	lmMainInputs := []string{"sequence", "text_embeddings"}
	for i := 0; i < lmMainStateCount; i++ {
		lmMainInputs = append(lmMainInputs, fmt.Sprintf("state_%d", i))
	}
	lmMainOutputs := []string{"conditioning", "eos_logit"}
	for i := 0; i < lmMainStateCount; i++ {
		lmMainOutputs = append(lmMainOutputs, fmt.Sprintf("out_state_%d", i))
	}
	p.lmMain, err = ort.NewDynamicAdvancedSession(
		lmMainPath,
		lmMainInputs,
		lmMainOutputs,
		nil,
	)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to load lm_main: %w", err)
	}

	p.lmFlow, err = ort.NewDynamicAdvancedSession(
		lmFlowPath,
		[]string{"c", "s", "t", "x"},
		[]string{"flow_dir"},
		nil,
	)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to load lm_flow: %w", err)
	}

	p.encoder, err = ort.NewDynamicAdvancedSession(
		encoderPath,
		[]string{"audio"},
		[]string{"latents"},
		nil,
	)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to load encoder: %w", err)
	}

	decoderInputs := []string{"latent"}
	for i := 0; i < decoderStateCount; i++ {
		decoderInputs = append(decoderInputs, fmt.Sprintf("state_%d", i))
	}
	decoderOutputs := []string{"audio_frame"}
	for i := 0; i < decoderStateCount; i++ {
		decoderOutputs = append(decoderOutputs, fmt.Sprintf("out_state_%d", i))
	}
	p.decoder, err = ort.NewDynamicAdvancedSession(
		decoderPath,
		decoderInputs,
		decoderOutputs,
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

func (p *Pipeline) Generate(textEmbeds, speakerEmbeds []float32, speed float32) ([]float32, error) {
	textLen := int64(len(textEmbeds)) / hiddenDim

	conditioning, err := p.runLMMain(textEmbeds, textLen)
	if err != nil {
		return nil, fmt.Errorf("failed to run lm_main: %w", err)
	}

	condLen := len(conditioning) / hiddenDim
	latents, err := p.runFlowMatching(conditioning, condLen)
	if err != nil {
		return nil, fmt.Errorf("failed to run flow matching: %w", err)
	}

	audio, err := p.runDecoder(latents)
	if err != nil {
		return nil, fmt.Errorf("failed to run decoder: %w", err)
	}

	return audio, nil
}

type lmMainState struct {
	kvCache  []float32
	emptyBuf []float32
	posIndex []int64
}

func newLMMainStates() []lmMainState {
	states := make([]lmMainState, 6)
	for i := range states {
		states[i] = lmMainState{
			kvCache:  make([]float32, 2*1*1000*16*64),
			emptyBuf: []float32{},
			posIndex: []int64{0},
		}
	}
	return states
}

func (p *Pipeline) runLMMain(textEmbeds []float32, seqLen int64) ([]float32, error) {
	states := newLMMainStates()
	var allConditioning []float32

	maxSteps := seqLen * 2
	if maxSteps > 500 {
		maxSteps = 500
	}

	for step := int64(0); step < maxSteps; step++ {
		seqData := make([]float32, 1*seqFeatureDim)
		seqTensor, err := ort.NewTensor(ort.NewShape(1, 1, seqFeatureDim), seqData)
		if err != nil {
			return nil, fmt.Errorf("failed to create sequence tensor: %w", err)
		}

		textEmbedsTensor, err := ort.NewTensor(ort.NewShape(1, seqLen, hiddenDim), textEmbeds)
		if err != nil {
			seqTensor.Destroy()
			return nil, fmt.Errorf("failed to create text_embeddings tensor: %w", err)
		}

		inputs := []ort.Value{seqTensor, textEmbedsTensor}

		for i := 0; i < 6; i++ {
			kvTensor, err := ort.NewTensor(ort.NewShape(2, 1, 1000, 16, 64), states[i].kvCache)
			if err != nil {
				destroyAll(inputs)
				return nil, fmt.Errorf("failed to create state_%d tensor: %w", i*3, err)
			}
			inputs = append(inputs, kvTensor)

			emptyTensor, err := ort.NewTensor(ort.NewShape(0), []float32{})
			if err != nil {
				destroyAll(inputs)
				return nil, fmt.Errorf("failed to create state_%d tensor: %w", i*3+1, err)
			}
			inputs = append(inputs, emptyTensor)

			posData := []int64{step}
			posTensor, err := ort.NewTensor(ort.NewShape(1), posData)
			if err != nil {
				destroyAll(inputs)
				return nil, fmt.Errorf("failed to create state_%d tensor: %w", i*3+2, err)
			}
			inputs = append(inputs, posTensor)
		}

		outputs := make([]ort.Value, 2+lmMainStateCount)
		if err := p.lmMain.Run(inputs, outputs); err != nil {
			destroyAll(inputs)
			return nil, fmt.Errorf("failed to run lm_main at step %d: %w", step, err)
		}

		destroyAll(inputs)

		if outputs[0] == nil {
			destroyAll(outputs)
			return nil, fmt.Errorf("no conditioning output at step %d", step)
		}

		condTensor, ok := outputs[0].(*ort.Tensor[float32])
		if !ok {
			destroyAll(outputs)
			return nil, fmt.Errorf("unexpected conditioning output type")
		}
		allConditioning = append(allConditioning, condTensor.GetData()...)

		if outputs[1] != nil {
			if eosTensor, ok := outputs[1].(*ort.Tensor[float32]); ok {
				eosData := eosTensor.GetData()
				if len(eosData) > 0 && eosData[0] > 0.5 {
					destroyAll(outputs)
					break
				}
			}
		}

		for i := 0; i < 6; i++ {
			baseIdx := 2 + i*3
			if outputs[baseIdx] != nil {
				if kvTensor, ok := outputs[baseIdx].(*ort.Tensor[float32]); ok {
					states[i].kvCache = append([]float32(nil), kvTensor.GetData()...)
				}
			}
		}

		destroyAll(outputs)
	}

	return allConditioning, nil
}

func (p *Pipeline) runFlowMatching(conditioning []float32, condLen int) ([]float32, error) {
	dt := 1.0 / float32(numFlowSteps)

	latents := make([]float32, condLen*seqFeatureDim)
	for i := range latents {
		latents[i] = float32(randNormal())
	}

	for step := 0; step < numFlowSteps; step++ {
		t := float32(step) / float32(numFlowSteps)

		cTensor, err := ort.NewTensor(ort.NewShape(int64(condLen), hiddenDim), conditioning)
		if err != nil {
			return nil, fmt.Errorf("failed to create c tensor: %w", err)
		}

		sData := make([]float32, condLen)
		for i := range sData {
			sData[i] = 1.0
		}
		sTensor, err := ort.NewTensor(ort.NewShape(int64(condLen), 1), sData)
		if err != nil {
			cTensor.Destroy()
			return nil, fmt.Errorf("failed to create s tensor: %w", err)
		}

		tData := make([]float32, condLen)
		for i := range tData {
			tData[i] = t
		}
		tTensor, err := ort.NewTensor(ort.NewShape(int64(condLen), 1), tData)
		if err != nil {
			cTensor.Destroy()
			sTensor.Destroy()
			return nil, fmt.Errorf("failed to create t tensor: %w", err)
		}

		xTensor, err := ort.NewTensor(ort.NewShape(int64(condLen), seqFeatureDim), latents)
		if err != nil {
			cTensor.Destroy()
			sTensor.Destroy()
			tTensor.Destroy()
			return nil, fmt.Errorf("failed to create x tensor: %w", err)
		}

		inputs := []ort.Value{cTensor, sTensor, tTensor, xTensor}
		outputs := make([]ort.Value, 1)

		if err := p.lmFlow.Run(inputs, outputs); err != nil {
			destroyAll(inputs)
			return nil, fmt.Errorf("failed to run lm_flow at step %d: %w", step, err)
		}

		destroyAll(inputs)

		if outputs[0] == nil {
			return nil, fmt.Errorf("no output from lm_flow at step %d", step)
		}

		flowTensor, ok := outputs[0].(*ort.Tensor[float32])
		if !ok {
			outputs[0].Destroy()
			return nil, fmt.Errorf("unexpected lm_flow output type")
		}

		flowDir := flowTensor.GetData()
		for i := range latents {
			if i < len(flowDir) {
				latents[i] += flowDir[i] * dt
			}
		}
		outputs[0].Destroy()
	}

	return latents, nil
}

func (p *Pipeline) runDecoder(latents []float32) ([]float32, error) {
	latentLen := len(latents) / seqFeatureDim

	latentTensor, err := ort.NewTensor(ort.NewShape(1, int64(latentLen), seqFeatureDim), latents)
	if err != nil {
		return nil, fmt.Errorf("failed to create latent tensor: %w", err)
	}

	inputs := []ort.Value{latentTensor}

	stateShapes := []struct {
		shape []int64
		dtype string
	}{
		{[]int64{1}, "bool"},
		{[]int64{1, 512, 6}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 64, 2}, "float"},
		{[]int64{1, 256, 6}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 256, 2}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 128, 0}, "float"},
		{[]int64{1, 128, 5}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 128, 2}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 64, 0}, "float"},
		{[]int64{1, 64, 4}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 64, 2}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 32, 0}, "float"},
		{[]int64{2, 1, 8, 1000, 64}, "float"},
		{[]int64{1}, "int64"},
		{[]int64{1}, "int64"},
		{[]int64{2, 1, 8, 1000, 64}, "float"},
		{[]int64{1}, "int64"},
		{[]int64{1}, "int64"},
		{[]int64{1}, "bool"},
		{[]int64{1, 512, 16}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 1, 6}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 64, 2}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 32, 0}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 512, 2}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 64, 4}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 128, 2}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 64, 0}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 128, 5}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 256, 2}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 128, 0}, "float"},
		{[]int64{1}, "bool"},
		{[]int64{1, 256, 6}, "float"},
		{[]int64{2, 1, 8, 1000, 64}, "float"},
		{[]int64{1}, "int64"},
		{[]int64{1}, "int64"},
		{[]int64{2, 1, 8, 1000, 64}, "float"},
		{[]int64{1}, "int64"},
		{[]int64{1}, "int64"},
		{[]int64{1, 512, 16}, "float"},
	}

	for i, ss := range stateShapes {
		size := int64(1)
		for _, d := range ss.shape {
			size *= d
		}

		var tensor ort.Value
		switch ss.dtype {
		case "bool":
			data := make([]bool, size)
			tensor, err = ort.NewTensor(ort.NewShape(ss.shape...), data)
		case "float":
			data := make([]float32, size)
			tensor, err = ort.NewTensor(ort.NewShape(ss.shape...), data)
		case "int64":
			data := make([]int64, size)
			tensor, err = ort.NewTensor(ort.NewShape(ss.shape...), data)
		}

		if err != nil {
			destroyAll(inputs)
			return nil, fmt.Errorf("failed to create decoder state_%d tensor: %w", i, err)
		}
		inputs = append(inputs, tensor)
	}

	outputs := make([]ort.Value, 1+decoderStateCount)
	if err := p.decoder.Run(inputs, outputs); err != nil {
		destroyAll(inputs)
		return nil, fmt.Errorf("failed to run decoder: %w", err)
	}

	destroyAll(inputs)

	if outputs[0] == nil {
		destroyAll(outputs)
		return nil, fmt.Errorf("no audio output from decoder")
	}

	audioTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		destroyAll(outputs)
		return nil, fmt.Errorf("unexpected audio output type")
	}

	audioData := append([]float32(nil), audioTensor.GetData()...)
	destroyAll(outputs)

	return audioData, nil
}

func (p *Pipeline) GetTextEmbeddings(inputIDs []int64) ([]float32, error) {
	inputTensor, err := ort.NewTensor(ort.NewShape(1, int64(len(inputIDs))), inputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create token_ids tensor: %w", err)
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

func destroyAll(values []ort.Value) {
	for _, v := range values {
		if v != nil {
			v.Destroy()
		}
	}
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

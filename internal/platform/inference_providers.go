package platform

import (
	"context"
	"errors"
)

type OllamaProvider struct {
	Endpoint string
	Enabled  bool
}

func (p OllamaProvider) Name() string { return "ollama" }
func (p OllamaProvider) Available(context.Context) bool {
	return p.Enabled && p.Endpoint != ""
}
func (p OllamaProvider) Infer(_ context.Context, req InferenceRequest) (InferenceResult, error) {
	if !p.Available(context.Background()) {
		return InferenceResult{}, errors.New("ollama unavailable")
	}
	return InferenceResult{Text: "", NonCanonical: true, Provider: p.Name(), ExecutionMode: ExecutionInline, Hardware: HardwareCPU}, nil
}

type LlamaCppProvider struct {
	Endpoint string
	Enabled  bool
}

func (p LlamaCppProvider) Name() string { return "llama.cpp" }
func (p LlamaCppProvider) Available(context.Context) bool {
	return p.Enabled && p.Endpoint != ""
}
func (p LlamaCppProvider) Infer(_ context.Context, req InferenceRequest) (InferenceResult, error) {
	if !p.Available(context.Background()) {
		return InferenceResult{}, errors.New("llama.cpp unavailable")
	}
	_ = req
	return InferenceResult{Text: "", NonCanonical: true, Provider: p.Name(), ExecutionMode: ExecutionInline, Hardware: HardwareCPU}, nil
}

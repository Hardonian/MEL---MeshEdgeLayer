package platform

import "testing"

func TestRuntimePolicyFallsBackWithoutProviders(t *testing.T) {
	p := TaskAwareRuntimePolicy{}
	decision := p.Select(InferenceRequest{TaskClass: TaskRealtimeAssist}, RuntimeEnvironment{})
	if decision.Availability != AssistUnavailable {
		t.Fatalf("expected unavailable, got %s", decision.Availability)
	}
	if decision.Provider != "none" {
		t.Fatalf("expected provider none, got %s", decision.Provider)
	}
}

func TestRuntimePolicyQueuesLowLatencyWhenAllowed(t *testing.T) {
	p := TaskAwareRuntimePolicy{}
	decision := p.Select(InferenceRequest{TaskClass: TaskDraftAndCompress, AllowBackgroundHandoff: true, LatencyBudgetMillis: 700}, RuntimeEnvironment{OllamaEnabled: true, QueueAvailable: true})
	if decision.Mode != ExecutionQueued {
		t.Fatalf("expected queued mode, got %s", decision.Mode)
	}
	if !decision.HandoffToQueue {
		t.Fatalf("expected queue handoff")
	}
}

func TestRuntimePolicySelectsLlamaForHeavyTaskWithTurboQuant(t *testing.T) {
	p := TaskAwareRuntimePolicy{}
	decision := p.Select(InferenceRequest{TaskClass: TaskProofpackSummary, ContextTokensEstimate: 8000}, RuntimeEnvironment{OllamaEnabled: true, LlamaCPPEnabled: true, PreferGPU: true, GPUAvailable: true, AllowExperimentalTurboQ: true})
	if decision.Provider != "llama.cpp" {
		t.Fatalf("expected llama.cpp provider, got %s", decision.Provider)
	}
	if decision.Hardware != HardwareGPU {
		t.Fatalf("expected gpu, got %s", decision.Hardware)
	}
	if decision.Compression != "experimental_turboquant_compatible" {
		t.Fatalf("expected turboquant-compatible compression, got %s", decision.Compression)
	}
}

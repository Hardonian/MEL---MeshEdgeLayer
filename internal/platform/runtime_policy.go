package platform

import "strings"

type TaskAwareRuntimePolicy struct{}

func (TaskAwareRuntimePolicy) Select(req InferenceRequest, env RuntimeEnvironment) RuntimeDecision {
	decision := RuntimeDecision{
		Provider:          "none",
		Mode:              ExecutionDisabled,
		Hardware:          HardwareCPU,
		Compression:       "none",
		Concurrency:       "single_thread",
		Availability:      AssistUnavailable,
		FallbackReason:    "assist_disabled_or_unavailable",
		NonCanonicalTruth: true,
	}

	if !env.OllamaEnabled && !env.LlamaCPPEnabled {
		return decision
	}

	if env.LlamaCPPEnabled && shouldPreferHeavyRuntime(req.TaskClass) {
		decision.Provider = "llama.cpp"
	} else if env.OllamaEnabled {
		decision.Provider = "ollama"
	} else if env.LlamaCPPEnabled {
		decision.Provider = "llama.cpp"
	}

	if req.TaskClass == TaskOfflineBatch {
		decision.Mode = ExecutionScheduled
		decision.Availability = AssistQueued
		decision.HandoffToQueue = true
		decision.FallbackReason = ""
	} else if req.AllowBackgroundHandoff && req.LatencyBudgetMillis > 0 && req.LatencyBudgetMillis < 900 {
		if env.QueueAvailable {
			decision.Mode = ExecutionQueued
			decision.Availability = AssistQueued
			decision.HandoffToQueue = true
			decision.FallbackReason = ""
		} else {
			decision.Mode = ExecutionDisabled
			decision.Availability = AssistPartial
			decision.FallbackReason = "inline_budget_exceeded_without_queue"
		}
	} else {
		decision.Mode = ExecutionInline
		decision.Availability = AssistAvailable
		decision.FallbackReason = ""
	}

	if env.PreferGPU && env.GPUAvailable && shouldPreferGPU(req.TaskClass) {
		decision.Hardware = HardwareGPU
	}
	if req.TaskClass == TaskOfflineBatch || req.TaskClass == TaskProofpackSummary || req.TaskClass == TaskIncidentComparison {
		decision.Concurrency = "multi_thread"
	}

	if shouldUseTurboQuant(req, env) {
		decision.Compression = "experimental_turboquant_compatible"
	} else if env.AllowStandardQuantization && req.ContextTokensEstimate >= 2048 {
		decision.Compression = "standard_quantization"
	}

	if strings.TrimSpace(decision.Provider) == "" || decision.Provider == "none" {
		decision.Mode = ExecutionDisabled
		decision.Availability = AssistUnavailable
		decision.FallbackReason = "no_runtime_provider_selected"
	}

	return decision
}

func shouldPreferHeavyRuntime(task TaskExecutionClass) bool {
	switch task {
	case TaskProofpackSummary, TaskIncidentComparison, TaskOfflineBatch:
		return true
	default:
		return false
	}
}

func shouldPreferGPU(task TaskExecutionClass) bool {
	switch task {
	case TaskProofpackSummary, TaskIncidentComparison, TaskOfflineBatch:
		return true
	default:
		return false
	}
}

func shouldUseTurboQuant(req InferenceRequest, env RuntimeEnvironment) bool {
	return env.AllowExperimentalTurboQ && req.ContextTokensEstimate >= 4096 && req.TaskClass != TaskRealtimeAssist
}

type DefaultInferenceJobRouter struct {
	Policy RuntimePolicy
}

func (r DefaultInferenceJobRouter) Route(req InferenceRequest, env RuntimeEnvironment) RuntimeDecision {
	if r.Policy == nil {
		r.Policy = TaskAwareRuntimePolicy{}
	}
	return r.Policy.Select(req, env)
}

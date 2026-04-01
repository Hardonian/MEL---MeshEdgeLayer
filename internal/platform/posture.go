package platform

import (
	"strings"

	"github.com/mel-project/mel/internal/config"
)

type ExportDeleteSemantics struct {
	ExportEnabled bool     `json:"export_enabled"`
	DeleteEnabled bool     `json:"delete_enabled"`
	DeleteScope   []string `json:"delete_scope"`
	DeleteCaveat  string   `json:"delete_caveat,omitempty"`
}

type ProviderPosture struct {
	Name               string `json:"name"`
	Enabled            bool   `json:"enabled"`
	EndpointConfigured bool   `json:"endpoint_configured"`
	AvailableByConfig  bool   `json:"available_by_config"`
}

type AssistTaskPolicy struct {
	TaskClass          TaskExecutionClass       `json:"task_class"`
	Availability       AssistAvailabilityStatus `json:"availability"`
	ExecutionMode      ExecutionMode            `json:"execution_mode"`
	Provider           string                   `json:"provider"`
	Hardware           HardwareTarget           `json:"hardware"`
	Compression        string                   `json:"compression"`
	Concurrency        string                   `json:"concurrency"`
	FallbackReason     string                   `json:"fallback_reason,omitempty"`
	LatencyBudgetMs    int                      `json:"latency_budget_ms"`
	ContextTokenBudget int                      `json:"context_token_budget"`
	NonCanonicalTruth  bool                     `json:"non_canonical_truth"`
}

type PlatformPosture struct {
	Mode                   string                 `json:"mode"`
	TelemetryEnabled       bool                   `json:"telemetry_enabled"`
	TelemetryOutbound      bool                   `json:"telemetry_outbound"`
	TelemetryExplicitOptIn bool                   `json:"telemetry_require_explicit_opt_in"`
	TelemetryStatus        string                 `json:"telemetry_status"`
	RetentionDefaultDays   int                    `json:"retention_default_days"`
	Retention              config.RetentionConfig `json:"retention"`
	EvidenceExportDelete   ExportDeleteSemantics  `json:"evidence_export_delete"`
	ExportRedactionEnabled bool                   `json:"export_redaction_enabled"`
	InferenceEnabled       bool                   `json:"inference_enabled"`
	InferenceRuntimeReady  bool                   `json:"inference_runtime_ready"`
	InferenceDegraded      bool                   `json:"inference_degraded"`
	InferenceCaveat        string                 `json:"inference_caveat,omitempty"`
	InferenceProviders     []ProviderPosture      `json:"inference_providers"`
	AssistPolicies         []AssistTaskPolicy     `json:"assist_policies"`
	OperatorIntelligence   OperatorIntelligencePosture `json:"operator_intelligence_posture"`
}

func BuildPosture(cfg config.Config) PlatformPosture {
	env := RuntimeEnvironment{
		OllamaEnabled:             cfg.Platform.Inference.Ollama.Enabled,
		LlamaCPPEnabled:           cfg.Platform.Inference.LlamaCPP.Enabled,
		PreferGPU:                 cfg.Platform.Inference.PreferGPU,
		GPUAvailable:              false,
		QueueAvailable:            cfg.Platform.Inference.AllowQueuedFallback,
		AllowExperimentalTurboQ:   cfg.Platform.Inference.Compression.AllowExperimentalTurboQuant,
		AllowStandardQuantization: cfg.Platform.Inference.Compression.AllowStandard,
	}
	policy := TaskAwareRuntimePolicy{}

	contextBudget := cfg.Platform.Inference.Budget.MaxContextTokens
	latencyBudget := cfg.Platform.Inference.Budget.RealtimeLatencyBudgetMs
	if contextBudget <= 0 {
		contextBudget = 4096
	}
	if latencyBudget <= 0 {
		latencyBudget = 900
	}

	tasks := []TaskExecutionClass{TaskRealtimeAssist, TaskDraftAndCompress, TaskProofpackSummary, TaskIncidentComparison, TaskOfflineBatch}
	assist := make([]AssistTaskPolicy, 0, len(tasks))
	for _, task := range tasks {
		req := InferenceRequest{
			TaskClass:              task,
			ContextTokensEstimate:  contextBudget,
			LatencyBudgetMillis:    latencyBudget,
			AllowBackgroundHandoff: cfg.Platform.Inference.AllowQueuedFallback,
		}
		if task == TaskOfflineBatch {
			req.LatencyBudgetMillis = cfg.Platform.Inference.Budget.BackgroundTimeoutMs
		}
		decision := policy.Select(req, env)
		assist = append(assist, AssistTaskPolicy{
			TaskClass:          task,
			Availability:       decision.Availability,
			ExecutionMode:      decision.Mode,
			Provider:           decision.Provider,
			Hardware:           decision.Hardware,
			Compression:        decision.Compression,
			Concurrency:        decision.Concurrency,
			FallbackReason:     decision.FallbackReason,
			LatencyBudgetMs:    req.LatencyBudgetMillis,
			ContextTokenBudget: contextBudget,
			NonCanonicalTruth:  true,
		})
	}

	deleteScope := []string{"topology_bookmarks"}
	deleteCaveat := "Delete APIs are currently scoped to selected artifacts (e.g. topology bookmarks); core evidence records are retention-pruned, not operator-hard-deleted."
	if !cfg.Platform.Retention.AllowDelete {
		deleteScope = []string{}
		deleteCaveat = "Delete APIs are disabled by policy (platform.retention.allow_delete=false)."
	}

	providers := []ProviderPosture{
		providerFromConfig("ollama", cfg.Platform.Inference.Ollama),
		providerFromConfig("llama.cpp", cfg.Platform.Inference.LlamaCPP),
	}
	runtimeReady := false
	for _, p := range providers {
		if p.AvailableByConfig {
			runtimeReady = true
			break
		}
	}
	inferenceDegraded := cfg.Platform.Inference.Enabled && !runtimeReady
	inferenceCaveat := ""
	if inferenceDegraded {
		inferenceCaveat = "Inference is enabled but no runtime provider is available by config; assistive output is unavailable and canonical MEL truth remains deterministic."
	}
	telemetryStatus := "disabled"
	switch {
	case cfg.Platform.Telemetry.Enabled && cfg.Platform.Telemetry.AllowOutbound:
		telemetryStatus = "enabled"
	case !cfg.Platform.Telemetry.Enabled && cfg.Platform.Telemetry.AllowOutbound:
		telemetryStatus = "degraded"
	case cfg.Platform.Telemetry.Enabled && !cfg.Platform.Telemetry.AllowOutbound:
		telemetryStatus = "degraded"
	}

	opIntel := BuildOperatorIntelligencePosture(cfg, inferenceDegraded, runtimeReady)

	return PlatformPosture{
		Mode:                   cfg.Platform.Mode,
		TelemetryEnabled:       cfg.Platform.Telemetry.Enabled,
		TelemetryOutbound:      cfg.Platform.Telemetry.AllowOutbound,
		TelemetryExplicitOptIn: cfg.Platform.Telemetry.RequireExplicit,
		TelemetryStatus:        telemetryStatus,
		RetentionDefaultDays:   cfg.Platform.Retention.DefaultDays,
		Retention:              cfg.Retention,
		EvidenceExportDelete: ExportDeleteSemantics{
			ExportEnabled: cfg.Platform.Retention.AllowExport,
			DeleteEnabled: cfg.Platform.Retention.AllowDelete,
			DeleteScope:   deleteScope,
			DeleteCaveat:  deleteCaveat,
		},
		ExportRedactionEnabled: cfg.Privacy.RedactExports,
		InferenceEnabled:       cfg.Platform.Inference.Enabled,
		InferenceRuntimeReady:  runtimeReady,
		InferenceDegraded:      inferenceDegraded,
		InferenceCaveat:        inferenceCaveat,
		InferenceProviders:     providers,
		AssistPolicies:         assist,
		OperatorIntelligence:   opIntel,
	}
}

func providerFromConfig(name string, cfg config.PlatformProviderConfig) ProviderPosture {
	endpointConfigured := strings.TrimSpace(cfg.Endpoint) != ""
	return ProviderPosture{
		Name:               name,
		Enabled:            cfg.Enabled,
		EndpointConfigured: endpointConfigured,
		AvailableByConfig:  cfg.Enabled && endpointConfigured,
	}
}

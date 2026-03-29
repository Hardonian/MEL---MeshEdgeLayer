package config

import (
	"fmt"
	"strings"
)

type PlatformConfig struct {
	Mode            string                 `json:"mode"`
	Telemetry       PlatformTelemetry      `json:"telemetry"`
	Retention       PlatformRetention      `json:"retention"`
	Crypto          PlatformProviderConfig `json:"crypto"`
	EventBus        PlatformProviderConfig `json:"event_bus"`
	BlobStore       PlatformProviderConfig `json:"blob_store"`
	Relay           PlatformProviderConfig `json:"relay"`
	SpeechToText    PlatformProviderConfig `json:"speech_to_text"`
	Inference       InferenceConfig        `json:"inference"`
	OptionalInterop PlatformProviderConfig `json:"optional_interop"`
}

type PlatformTelemetry struct {
	Enabled         bool `json:"enabled"`
	AllowOutbound   bool `json:"allow_outbound"`
	RequireExplicit bool `json:"require_explicit_opt_in"`
}

type PlatformRetention struct {
	DefaultDays int  `json:"default_days"`
	AllowExport bool `json:"allow_export"`
	AllowDelete bool `json:"allow_delete"`
}

type PlatformProviderConfig struct {
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	Required bool   `json:"required"`
}

type InferenceConfig struct {
	Enabled             bool                   `json:"enabled"`
	DefaultProvider     string                 `json:"default_provider"`
	AllowQueuedFallback bool                   `json:"allow_queued_fallback"`
	PreferGPU           bool                   `json:"prefer_gpu"`
	Ollama              PlatformProviderConfig `json:"ollama"`
	LlamaCPP            PlatformProviderConfig `json:"llama_cpp"`
	Compression         CompressionConfig      `json:"compression"`
	Budget              InferenceBudgetConfig  `json:"budget"`
}

type CompressionConfig struct {
	DefaultStrategy             string `json:"default_strategy"`
	AllowStandard               bool   `json:"allow_standard"`
	AllowExperimentalTurboQuant bool   `json:"allow_experimental_turboquant_compatible"`
}

type InferenceBudgetConfig struct {
	MaxContextTokens          int `json:"max_context_tokens"`
	RealtimeLatencyBudgetMs   int `json:"realtime_latency_budget_ms"`
	BackgroundTimeoutMs       int `json:"background_timeout_ms"`
	QueueTimeoutMs            int `json:"queue_timeout_ms"`
	MaxParallelInferenceTasks int `json:"max_parallel_inference_tasks"`
}

func defaultPlatformConfig() PlatformConfig {
	return PlatformConfig{
		Mode:         "self_hosted",
		Telemetry:    PlatformTelemetry{Enabled: false, AllowOutbound: false, RequireExplicit: true},
		Retention:    PlatformRetention{DefaultDays: 30, AllowExport: true, AllowDelete: true},
		Crypto:       PlatformProviderConfig{Provider: "libsignal-compatible", Enabled: true, Required: true},
		EventBus:     PlatformProviderConfig{Provider: "nats-jetstream", Enabled: true, Endpoint: "nats://127.0.0.1:4222", Required: false},
		BlobStore:    PlatformProviderConfig{Provider: "s3-compatible", Enabled: false, Endpoint: "http://127.0.0.1:9000", Required: false},
		Relay:        PlatformProviderConfig{Provider: "coturn", Enabled: false, Endpoint: "turn:127.0.0.1:3478", Required: false},
		SpeechToText: PlatformProviderConfig{Provider: "whisper.cpp", Enabled: false, Endpoint: "http://127.0.0.1:8088", Required: false},
		Inference: InferenceConfig{
			Enabled:             false,
			DefaultProvider:     "none",
			AllowQueuedFallback: true,
			PreferGPU:           true,
			Ollama:              PlatformProviderConfig{Provider: "ollama", Enabled: false, Endpoint: "http://127.0.0.1:11434", Required: false},
			LlamaCPP:            PlatformProviderConfig{Provider: "llama.cpp", Enabled: false, Endpoint: "http://127.0.0.1:8089", Required: false},
			Compression:         CompressionConfig{DefaultStrategy: "none", AllowStandard: true, AllowExperimentalTurboQuant: false},
			Budget: InferenceBudgetConfig{
				MaxContextTokens:          4096,
				RealtimeLatencyBudgetMs:   900,
				BackgroundTimeoutMs:       30000,
				QueueTimeoutMs:            120000,
				MaxParallelInferenceTasks: 2,
			},
		},
		OptionalInterop: PlatformProviderConfig{Provider: "matrix-bridge", Enabled: false, Required: false},
	}
}

func normalizePlatform(cfg *Config) {
	d := defaultPlatformConfig()
	if cfg.Platform.Mode == "" {
		cfg.Platform.Mode = d.Mode
	}
	if cfg.Platform.Retention.DefaultDays <= 0 {
		cfg.Platform.Retention.DefaultDays = d.Retention.DefaultDays
	}
	if cfg.Platform.Crypto.Provider == "" {
		cfg.Platform.Crypto.Provider = d.Crypto.Provider
	}
	if cfg.Platform.EventBus.Provider == "" {
		cfg.Platform.EventBus.Provider = d.EventBus.Provider
	}
	if cfg.Platform.EventBus.Endpoint == "" {
		cfg.Platform.EventBus.Endpoint = d.EventBus.Endpoint
	}
	if cfg.Platform.BlobStore.Provider == "" {
		cfg.Platform.BlobStore.Provider = d.BlobStore.Provider
	}
	if cfg.Platform.BlobStore.Endpoint == "" {
		cfg.Platform.BlobStore.Endpoint = d.BlobStore.Endpoint
	}
	if cfg.Platform.Relay.Provider == "" {
		cfg.Platform.Relay.Provider = d.Relay.Provider
	}
	if cfg.Platform.Relay.Endpoint == "" {
		cfg.Platform.Relay.Endpoint = d.Relay.Endpoint
	}
	if cfg.Platform.SpeechToText.Provider == "" {
		cfg.Platform.SpeechToText.Provider = d.SpeechToText.Provider
	}
	if cfg.Platform.SpeechToText.Endpoint == "" {
		cfg.Platform.SpeechToText.Endpoint = d.SpeechToText.Endpoint
	}
	if cfg.Platform.Inference.DefaultProvider == "" {
		cfg.Platform.Inference.DefaultProvider = d.Inference.DefaultProvider
	}
	if cfg.Platform.Inference.Ollama.Provider == "" {
		cfg.Platform.Inference.Ollama.Provider = d.Inference.Ollama.Provider
	}
	if cfg.Platform.Inference.Ollama.Endpoint == "" {
		cfg.Platform.Inference.Ollama.Endpoint = d.Inference.Ollama.Endpoint
	}
	if cfg.Platform.Inference.LlamaCPP.Provider == "" {
		cfg.Platform.Inference.LlamaCPP.Provider = d.Inference.LlamaCPP.Provider
	}
	if cfg.Platform.Inference.LlamaCPP.Endpoint == "" {
		cfg.Platform.Inference.LlamaCPP.Endpoint = d.Inference.LlamaCPP.Endpoint
	}
	if cfg.Platform.Inference.Compression.DefaultStrategy == "" {
		cfg.Platform.Inference.Compression.DefaultStrategy = d.Inference.Compression.DefaultStrategy
	}
	if cfg.Platform.Inference.Budget.MaxContextTokens <= 0 {
		cfg.Platform.Inference.Budget.MaxContextTokens = d.Inference.Budget.MaxContextTokens
	}
	if cfg.Platform.Inference.Budget.RealtimeLatencyBudgetMs <= 0 {
		cfg.Platform.Inference.Budget.RealtimeLatencyBudgetMs = d.Inference.Budget.RealtimeLatencyBudgetMs
	}
	if cfg.Platform.Inference.Budget.BackgroundTimeoutMs <= 0 {
		cfg.Platform.Inference.Budget.BackgroundTimeoutMs = d.Inference.Budget.BackgroundTimeoutMs
	}
	if cfg.Platform.Inference.Budget.QueueTimeoutMs <= 0 {
		cfg.Platform.Inference.Budget.QueueTimeoutMs = d.Inference.Budget.QueueTimeoutMs
	}
	if cfg.Platform.Inference.Budget.MaxParallelInferenceTasks <= 0 {
		cfg.Platform.Inference.Budget.MaxParallelInferenceTasks = d.Inference.Budget.MaxParallelInferenceTasks
	}
	if cfg.Platform.OptionalInterop.Provider == "" {
		cfg.Platform.OptionalInterop.Provider = d.OptionalInterop.Provider
	}
}

func validatePlatform(cfg Config) []string {
	var errs []string
	mode := strings.ToLower(strings.TrimSpace(cfg.Platform.Mode))
	if mode != "" && mode != "self_hosted" {
		errs = append(errs, "platform.mode must be self_hosted")
	}
	if cfg.Platform.Telemetry.Enabled && !cfg.Platform.Telemetry.AllowOutbound {
		errs = append(errs, "platform.telemetry.enabled=true requires platform.telemetry.allow_outbound=true")
	}
	if cfg.Platform.Retention.DefaultDays < 1 || cfg.Platform.Retention.DefaultDays > 365 {
		errs = append(errs, "platform.retention.default_days must be between 1 and 365")
	}
	if !cfg.Platform.Retention.AllowExport {
		errs = append(errs, "platform.retention.allow_export must remain true to preserve operator evidence portability")
	}
	if cfg.Platform.Crypto.Provider == "" {
		errs = append(errs, "platform.crypto.provider is required")
	}
	if cfg.Platform.EventBus.Enabled && cfg.Platform.EventBus.Endpoint == "" {
		errs = append(errs, "platform.event_bus.endpoint required when enabled")
	}
	if cfg.Platform.BlobStore.Enabled && cfg.Platform.BlobStore.Endpoint == "" {
		errs = append(errs, "platform.blob_store.endpoint required when enabled")
	}
	if cfg.Platform.Relay.Enabled && cfg.Platform.Relay.Endpoint == "" {
		errs = append(errs, "platform.relay.endpoint required when enabled")
	}
	if cfg.Platform.SpeechToText.Enabled && cfg.Platform.SpeechToText.Endpoint == "" {
		errs = append(errs, "platform.speech_to_text.endpoint required when enabled")
	}
	if cfg.Platform.Inference.Enabled {
		allowed := map[string]bool{"ollama": true, "llama.cpp": true, "mixed": true}
		if !allowed[cfg.Platform.Inference.DefaultProvider] {
			errs = append(errs, fmt.Sprintf("platform.inference.default_provider must be one of ollama, llama.cpp, mixed when enabled (got %q)", cfg.Platform.Inference.DefaultProvider))
		}
		if cfg.Platform.Inference.DefaultProvider == "ollama" && !cfg.Platform.Inference.Ollama.Enabled {
			errs = append(errs, "platform.inference.default_provider=ollama requires platform.inference.ollama.enabled=true")
		}
		if cfg.Platform.Inference.DefaultProvider == "llama.cpp" && !cfg.Platform.Inference.LlamaCPP.Enabled {
			errs = append(errs, "platform.inference.default_provider=llama.cpp requires platform.inference.llama_cpp.enabled=true")
		}
	}
	strategy := cfg.Platform.Inference.Compression.DefaultStrategy
	allowedStrategies := map[string]bool{"none": true, "standard_quantization": true, "experimental_turboquant_compatible": true}
	if !allowedStrategies[strategy] {
		errs = append(errs, "platform.inference.compression.default_strategy must be none, standard_quantization, or experimental_turboquant_compatible")
	}
	if strategy == "experimental_turboquant_compatible" && !cfg.Platform.Inference.Compression.AllowExperimentalTurboQuant {
		errs = append(errs, "experimental turboquant-compatible compression needs platform.inference.compression.allow_experimental_turboquant_compatible=true")
	}
	if cfg.Platform.Inference.Budget.MaxContextTokens < 512 || cfg.Platform.Inference.Budget.MaxContextTokens > 32768 {
		errs = append(errs, "platform.inference.budget.max_context_tokens must be between 512 and 32768")
	}
	if cfg.Platform.Inference.Budget.RealtimeLatencyBudgetMs < 100 || cfg.Platform.Inference.Budget.RealtimeLatencyBudgetMs > 30000 {
		errs = append(errs, "platform.inference.budget.realtime_latency_budget_ms must be between 100 and 30000")
	}
	if cfg.Platform.Inference.Budget.BackgroundTimeoutMs < 1000 || cfg.Platform.Inference.Budget.BackgroundTimeoutMs > 300000 {
		errs = append(errs, "platform.inference.budget.background_timeout_ms must be between 1000 and 300000")
	}
	if cfg.Platform.Inference.Budget.QueueTimeoutMs < cfg.Platform.Inference.Budget.BackgroundTimeoutMs {
		errs = append(errs, "platform.inference.budget.queue_timeout_ms must be >= background_timeout_ms")
	}
	if cfg.Platform.Inference.Budget.MaxParallelInferenceTasks < 1 || cfg.Platform.Inference.Budget.MaxParallelInferenceTasks > 16 {
		errs = append(errs, "platform.inference.budget.max_parallel_inference_tasks must be between 1 and 16")
	}
	return errs
}

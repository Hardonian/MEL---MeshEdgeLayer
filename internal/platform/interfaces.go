package platform

import "context"

type Event struct {
	Kind      string
	Source    string
	Payload   []byte
	Timestamp int64
}

type ObjectRef struct {
	Bucket string
	Key    string
	Size   int64
}

type TaskExecutionClass string

const (
	TaskRealtimeAssist     TaskExecutionClass = "realtime_assist"
	TaskDraftAndCompress   TaskExecutionClass = "draft_and_note_compression"
	TaskProofpackSummary   TaskExecutionClass = "proofpack_summarization"
	TaskIncidentComparison TaskExecutionClass = "incident_comparison"
	TaskOfflineBatch       TaskExecutionClass = "offline_batch"
)

type ExecutionMode string

const (
	ExecutionInline    ExecutionMode = "inline"
	ExecutionQueued    ExecutionMode = "queued"
	ExecutionScheduled ExecutionMode = "scheduled"
	ExecutionDisabled  ExecutionMode = "disabled"
)

type HardwareTarget string

const (
	HardwareCPU HardwareTarget = "cpu"
	HardwareGPU HardwareTarget = "gpu"
)

type CompressionStrategy interface {
	Name() string
	Encode(input []byte) ([]byte, error)
}

type CryptoProvider interface {
	Name() string
	EncryptEnvelope(ctx context.Context, plaintext []byte, peerID string) ([]byte, error)
	DecryptEnvelope(ctx context.Context, ciphertext []byte, peerID string) ([]byte, error)
}

type EventBus interface {
	Name() string
	Publish(ctx context.Context, subject string, evt Event) error
	Subscribe(ctx context.Context, subject string, handler func(Event) error) error
}

type BlobStore interface {
	Name() string
	Put(ctx context.Context, ref ObjectRef, content []byte) error
	Get(ctx context.Context, ref ObjectRef) ([]byte, error)
}

type RelayService interface {
	Name() string
	Healthy(ctx context.Context) bool
}

type SpeechToTextProvider interface {
	Name() string
	Transcribe(ctx context.Context, audio []byte) (string, error)
}

type LocalInferenceProvider interface {
	Name() string
	Available(ctx context.Context) bool
	Infer(ctx context.Context, req InferenceRequest) (InferenceResult, error)
}

type OptionalInteropProvider interface {
	Name() string
	Send(ctx context.Context, room string, payload []byte) error
}

type RuntimePolicy interface {
	Select(req InferenceRequest, env RuntimeEnvironment) RuntimeDecision
}

type InferenceJobRouter interface {
	Route(req InferenceRequest, env RuntimeEnvironment) RuntimeDecision
}

type AssistAvailabilityStatus string

const (
	AssistAvailable   AssistAvailabilityStatus = "available"
	AssistQueued      AssistAvailabilityStatus = "queued"
	AssistPartial     AssistAvailabilityStatus = "partial"
	AssistUnavailable AssistAvailabilityStatus = "unavailable"
)

type InferenceRequest struct {
	TaskClass              TaskExecutionClass
	Prompt                 string
	ContextTokensEstimate  int
	LatencyBudgetMillis    int
	AllowBackgroundHandoff bool
}

type InferenceResult struct {
	Text            string
	NonCanonical    bool
	Provider        string
	ExecutionMode   ExecutionMode
	Hardware        HardwareTarget
	CompressionUsed string
	Partial         bool
}

type RuntimeEnvironment struct {
	OllamaEnabled             bool
	LlamaCPPEnabled           bool
	PreferGPU                 bool
	GPUAvailable              bool
	QueueAvailable            bool
	AllowExperimentalTurboQ   bool
	AllowStandardQuantization bool
}

type RuntimeDecision struct {
	Provider          string
	Mode              ExecutionMode
	Hardware          HardwareTarget
	Compression       string
	Concurrency       string
	Availability      AssistAvailabilityStatus
	FallbackReason    string
	HandoffToQueue    bool
	NonCanonicalTruth bool
}

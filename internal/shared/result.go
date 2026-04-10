// Package shared provides standardized patterns for MEL
// matching the TypeScript types in @hardonian/shared
package shared

import (
	"fmt"
	"time"
)

// ResultState represents the standardized operation states
type ResultState string

const (
	ResultStateSuccess         ResultState = "success"
	ResultStateValidationError ResultState = "validation_error"
	ResultStateNotFound        ResultState = "not_found"
	ResultStateUnauthorized    ResultState = "unauthorized"
	ResultStateForbidden       ResultState = "forbidden"
	ResultStateRateLimited     ResultState = "rate_limited"
	ResultStateDegraded        ResultState = "degraded"
	ResultStateUnavailable     ResultState = "unavailable"
	ResultStateConflict        ResultState = "conflict"
	ResultStateInternalError   ResultState = "internal_error"
)

// ToHTTPStatus returns the HTTP status code for a ResultState
func (s ResultState) ToHTTPStatus() int {
	switch s {
	case ResultStateSuccess:
		return 200
	case ResultStateValidationError:
		return 400
	case ResultStateUnauthorized:
		return 401
	case ResultStateForbidden:
		return 403
	case ResultStateNotFound:
		return 404
	case ResultStateConflict:
		return 409
	case ResultStateRateLimited:
		return 429
	case ResultStateDegraded, ResultStateUnavailable:
		return 503
	case ResultStateInternalError:
		return 500
	default:
		return 500
	}
}

// ErrorDetails provides structured error information
type ErrorDetails struct {
	Message string                 `json:"message"`
	Code    string                 `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e ErrorDetails) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ResultMetadata provides observability information
type ResultMetadata struct {
	TraceID     string   `json:"traceId"`
	Timestamp   string   `json:"timestamp"`
	DurationMs  int64    `json:"durationMs"`
	ReasonCodes []string `json:"reasonCodes,omitempty"`
}

// StandardResult is the unified result type
type StandardResult[T any] struct {
	OK       bool           `json:"ok"`
	State    ResultState    `json:"state"`
	Data     *T             `json:"data,omitempty"`
	Error    *ErrorDetails  `json:"error,omitempty"`
	Metadata ResultMetadata `json:"metadata"`
}

// IsSuccess returns true if the result is successful
func (r StandardResult[T]) IsSuccess() bool {
	return r.OK
}

// IsFailure returns true if the result is a failure
func (r StandardResult[T]) IsFailure() bool {
	return !r.OK
}

// Unwrap returns the data or panics on failure
func (r StandardResult[T]) Unwrap() T {
	if r.IsFailure() {
		if r.Error != nil {
			panic(r.Error.Error())
		}
		panic("operation failed")
	}
	if r.Data == nil {
		var zero T
		return zero
	}
	return *r.Data
}

// UnwrapOr returns the data or a default value on failure
func (r StandardResult[T]) UnwrapOr(defaultValue T) T {
	if r.IsSuccess() && r.Data != nil {
		return *r.Data
	}
	return defaultValue
}

// Map transforms a successful result
func (r StandardResult[T]) Map(fn func(T) T) StandardResult[T] {
	if r.IsSuccess() && r.Data != nil {
		newData := fn(*r.Data)
		return StandardResult[T]{
			OK:       true,
			State:    ResultStateSuccess,
			Data:     &newData,
			Metadata: r.Metadata,
		}
	}
	return r
}

// Success creates a successful result
func Success[T any](data T, metadata ResultMetadata) StandardResult[T] {
	return StandardResult[T]{
		OK:       true,
		State:    ResultStateSuccess,
		Data:     &data,
		Metadata: metadata,
	}
}

// Failure creates a failed result
func Failure[T any](state ResultState, error ErrorDetails, metadata ResultMetadata) StandardResult[T] {
	return StandardResult[T]{
		OK:       false,
		State:    state,
		Error:    &error,
		Metadata: metadata,
	}
}

// VoidSuccess creates a successful result with no data
func VoidSuccess(metadata ResultMetadata) StandardResult[struct{}] {
	return StandardResult[struct{}]{
		OK:       true,
		State:    ResultStateSuccess,
		Metadata: metadata,
	}
}

// ResultBuilder helps construct results with timing
type ResultBuilder struct {
	TraceID     string
	StartTime   time.Time
	ReasonCodes []string
}

// NewResultBuilder creates a new result builder
func NewResultBuilder(traceID string) *ResultBuilder {
	return &ResultBuilder{
		TraceID:   traceID,
		StartTime: time.Now(),
	}
}

// WithReason adds a reason code
func (b *ResultBuilder) WithReason(reason string) *ResultBuilder {
	b.ReasonCodes = append(b.ReasonCodes, reason)
	return b
}

// SuccessAny creates a success result using any payload.
// Note: methods on non-generic types cannot declare type parameters.
func (b *ResultBuilder) SuccessAny(data any) StandardResult[any] {
	return Success(data, ResultMetadata{
		TraceID:     b.TraceID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		DurationMs:  time.Since(b.StartTime).Milliseconds(),
		ReasonCodes: b.ReasonCodes,
	})
}

// Failure creates a failure result
func (b *ResultBuilder) Failure(state ResultState, code, message string) StandardResult[struct{}] {
	return Failure[struct{}](state, ErrorDetails{
		Code:    code,
		Message: message,
	}, ResultMetadata{
		TraceID:     b.TraceID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		DurationMs:  time.Since(b.StartTime).Milliseconds(),
		ReasonCodes: b.ReasonCodes,
	})
}

// TryCatch runs an operation and wraps it in a StandardResult
func TryCatch[T any](operation func() (T, error), traceID string) StandardResult[T] {
	builder := NewResultBuilder(traceID)

	data, err := operation()
	if err != nil {
		return Failure[T](ResultStateInternalError, ErrorDetails{
			Code:    "OPERATION_FAILED",
			Message: err.Error(),
		}, builder.Metadata())
	}

	return Success(data, builder.Metadata())
}

// Metadata returns the current metadata for the builder
func (b *ResultBuilder) Metadata() ResultMetadata {
	return ResultMetadata{
		TraceID:     b.TraceID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		DurationMs:  time.Since(b.StartTime).Milliseconds(),
		ReasonCodes: b.ReasonCodes,
	}
}

// GenerateTraceID generates a trace ID
func GenerateTraceID() string {
	return fmt.Sprintf("tr_%d_%d", time.Now().UnixMilli(), time.Now().Nanosecond())
}

// GenerateTraceIDWithPrefix generates a trace ID with a prefix
func GenerateTraceIDWithPrefix(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), time.Now().Nanosecond())
}

# Self-Observability Guide

MEL includes comprehensive internal self-observability features that allow operators to monitor the health, freshness, and SLO compliance of internal components.

## Overview

Self-observability is critical for understanding MEL's internal operational state. Unlike external observability (monitoring mesh devices), self-observability tracks MEL's own components:

- **Internal Health**: Real-time health status of MEL's internal components
- **Freshness Tracking**: Ensuring background jobs and pipelines are running on schedule
- **SLO Monitoring**: Tracking service level objectives for internal operations
- **Internal Metrics**: Pipeline latencies, queue depths, and resource usage

## Internal Health Model

MEL tracks health for the following internal components:

| Component | Description |
|-----------|-------------|
| `ingest` | Message ingestion pipeline |
| `classify` | Message classification/intelligence |
| `alert` | Alert generation and delivery |
| `control` | Control plane operations |
| `retention` | Data retention/pruning |
| `backup` | Database backup operations |

### Health States

- **healthy**: Component operating normally
- **degraded**: Component working but with reduced performance (>1% error rate)
- **failing**: Component not functioning properly (>10% error rate)
- **unknown**: No data collected yet

### Health Transitions

Health automatically transitions based on error rates:

```
Unknown → Healthy (no errors)
Healthy → Degraded (error rate > 1%)
Degraded → Failing (error rate > 10%)
Failing → Degraded (error rate < 10%)
```

## Freshness Semantics

Freshness tracking ensures background jobs complete within expected intervals.

### Default Freshness Intervals

| Component | Expected Interval | Stale Threshold |
|-----------|------------------|-----------------|
| `ingest` | 10 seconds | 60 seconds |
| `classify` | 30 seconds | 2 minutes |
| `alert` | 60 seconds | 5 minutes |
| `control` | 30 seconds | 2 minutes |
| `retention` | 5 minutes | 10 minutes |
| `backup` | 1 hour | 24 hours |

### Freshness States

- **Fresh**: Last update within stale threshold
- **Stale**: No update within stale threshold
- **Never Updated**: Component has not reported yet

### Troubleshooting Stale Components

If a component shows as stale:

1. Check MEL service logs for errors
2. Verify background goroutines are running
3. Check system resources (memory, CPU)
4. Review recent incidents in `/api/v1/incidents`

## SLO Definitions

MEL tracks the following SLOs:

### message_ingest_latency

- **Target**: 95% of messages ingested within threshold
- **Window**: 24 hours
- **Metric**: `ingest_latency_p99` (ms)

### alert_freshness

- **Target**: 99% of alert cycles completing on time
- **Window**: 24 hours
- **Metric**: `alert_cycle_success` (%)

### control_success_rate

- **Target**: 99.5% success rate
- **Window**: 24 hours
- **Metric**: `control_operation_success` (%)

### retention_compliance

- **Target**: 100% (must always complete)
- **Window**: 24 hours
- **Metric**: `retention_run_success` (%)

### backup_success

- **Target**: 99% success rate
- **Window**: 24 hours
- **Metric**: `backup_operation_success` (%)

### SLO Status Values

- **healthy**: Within budget
- **at_risk**: 50-99% of error budget used
- **breached**: 100%+ of error budget used

## Internal Metrics

### Pipeline Latency

- `ingest_to_classify_p99`: 99th percentile latency
- `classify_to_alert_p99`: Classification to alert generation
- `alert_to_action_p99`: Alert to action execution

### Queue Depths

Current queue size for each pipeline stage. High queue depths may indicate processing bottlenecks.

### Error Rates

Percentage of failed operations per component over time.

### Resource Usage

- `memory_used_bytes`: Current heap memory allocation
- `goroutines`: Number of running goroutines
- `num_gc`: Number of garbage collections

## Correlation IDs

MEL generates correlation IDs for end-to-end tracing of events through the pipeline.

### Correlation ID Format

```
[a-f0-9]{32}  // 128-bit UUID hex encoding
```

### Propagation

Correlation IDs are propagated through:

1. **Ingest**: Created when message is first received
2. **Classify**: Carried through classification
3. **Alert**: Included in alert generation
4. **Action**: Used in control plane actions

### Usage

```go
// Create new correlation ID
corr := selfobs.NewCorrelationID("ingest")

// Add to context
ctx := selfobs.ContextWithCorrelationID(context.Background(), corr)

// Extract from context
if corr, ok := selfobs.FromContext(ctx); ok {
    log.Printf("Correlation ID: %s", corr.ID)
}
```

## API Endpoints

### GET /api/v1/health/internal

Returns internal component health status.

```json
{
  "overall_health": "healthy",
  "components": [
    {
      "name": "ingest",
      "health": "healthy",
      "last_success": "2026-03-21T01:00:00Z",
      "last_failure": "0001-01-01T00:00:00Z",
      "error_count": 0,
      "success_count": 1000,
      "error_rate": 0
    }
  ]
}
```

### GET /api/v1/health/freshness

Returns freshness status for all components.

```json
{
  "markers": [
    {
      "component": "ingest",
      "last_update": "2026-03-21T01:00:00Z",
      "age_seconds": 5.0,
      "is_fresh": true,
      "is_stale": false,
      "expected_interval": 10,
      "stale_threshold": 60
    }
  ],
  "stale_components": []
}
```

### GET /api/v1/health/slo

Returns SLO compliance status.

```json
{
  "slos": [
    {
      "name": "message_ingest_latency",
      "description": "P99 latency from message receipt to successful ingest",
      "current_value": 98.5,
      "target": 95.0,
      "status": "healthy",
      "budget_used": 0,
      "unit": "ms",
      "window": "24h0m0s"
    }
  ]
}
```

### GET /api/v1/metrics/internal

Returns internal metrics snapshot.

```json
{
  "timestamp": "2026-03-21T01:00:00Z",
  "pipeline_latency": {
    "ingest_to_classify_p99": 50,
    "classify_to_alert_p99": 100,
    "alert_to_action_p99": 25
  },
  "queue_depths": {
    "ingest": 10,
    "classify": 5,
    "alert": 0
  },
  "error_rates": {
    "ingest": 0.1,
    "classify": 0.05
  },
  "resource_usage": {
    "memory_used_bytes": 10485760,
    "goroutines": 25,
    "num_gc": 100
  }
}
```

## CLI Commands

### mel health internal

Show internal component health:

```bash
mel health internal
```

Output:
```
Overall Health: healthy

Components:
  ✓ ingest - Error Rate: 0.0%
  ✓ classify - Error Rate: 0.1%
  ✓ alert - Error Rate: 0.0%
```

### mel health freshness

Show freshness status:

```bash
mel health freshness
```

Output:
```
✓ All components fresh

Component Freshness:
  ✓ ingest - 5s ago (threshold: 60s)
  ✓ classify - 15s ago (threshold: 120s)
```

### mel health slo

Show SLO status:

```bash
mel health slo
```

Output:
```
SLO Status:
  ✓ message_ingest_latency: 98.5% (target: 95.0%) [healthy]
  ✓ alert_freshness: 99.5% (target: 99.0%) [healthy]
  ✓ control_success_rate: 100.0% (target: 99.5%) [healthy]
```

### mel health metrics

Show internal metrics:

```bash
mel health metrics
```

Output:
```
Pipeline Latency (P99):
  Ingest → Classify: 50ms
  Classify → Alert: 100ms
  Alert → Action: 25ms

Queue Depths:
  ingest: 10
  classify: 5
  alert: 0

Error Rates:
  ingest: 0.10%
  classify: 0.05%

Resource Usage:
  Memory: 10485760 bytes
  Goroutines: 25
```

## Support Bundle

Self-observability data is automatically included in support bundles generated by `/api/v1/diagnostics`.

## Troubleshooting

### Component Stuck in Unknown

1. Ensure MEL has been running for at least one freshness interval
2. Check that the component has been invoked at least once

### High Error Rates

1. Review MEL logs for specific error messages
2. Check transport connectivity
3. Verify database accessibility

### Stale Components

1. Check if MEL service is responsive
2. Review background job status
3. Check for resource exhaustion
4. Review recent incidents

### SLO Breaches

SLO breaches indicate that error budgets have been exhausted. This typically requires investigation:

1. Identify the failing component
2. Review error patterns
3. Check system resources
4. Consider adjusting SLO targets if legitimate

## Integration Points

Components should call `MarkFresh()` after completing their work cycle:

```go
// After successful batch ingest
selfobs.MarkFresh("ingest")
selfobs.GetGlobalRegistry().RecordSuccess("ingest")

// After alert run
selfobs.MarkFresh("alert")
selfobs.RecordSLOSuccess("alert_cycle_success")

// After control operation
selfobs.GetGlobalRegistry().RecordSuccess("control")
```

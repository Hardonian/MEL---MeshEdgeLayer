package status

import (
	"testing"

	"github.com/mel-project/mel/internal/transport"
)

func TestBuildPanelReflectsHistoricalOnlyState(t *testing.T) {
	snap := Snapshot{
		GeneratedAt: "2026-03-19T00:00:00Z",
		Transports: []TransportReport{{
			Name:              "mqtt",
			Type:              "mqtt",
			EffectiveState:    transport.StateHistoricalOnly,
			PersistedMessages: 4,
			Detail:            "historical ingest exists (4 stored messages); current runtime state is not proven live",
		}},
	}
	panel := BuildPanel(snap)
	if panel.OperatorState != "degraded" {
		t.Fatalf("expected degraded operator state, got %+v", panel)
	}
	if panel.Summary == "" || len(panel.ShortCommands) == 0 || len(panel.DeviceMenu) == 0 {
		t.Fatalf("expected compact panel metadata, got %+v", panel)
	}
}

func TestBuildPanelReflectsLiveReadyState(t *testing.T) {
	snap := Snapshot{
		GeneratedAt: "2026-03-19T00:00:00Z",
		Transports: []TransportReport{{
			Name:           "direct",
			Type:           "tcp",
			EffectiveState: transport.StateIngesting,
			TotalMessages:  2,
			LastIngestAt:   "2026-03-19T00:00:00Z",
			Detail:         "live ingest confirmed by SQLite writes",
		}},
	}
	panel := BuildPanel(snap)
	if panel.OperatorState != "ready" {
		t.Fatalf("expected ready operator state, got %+v", panel)
	}
	if got := panel.Transports[0].Messages; got != 2 {
		t.Fatalf("expected message count 2, got %d", got)
	}
}

func TestCollectBuildsRetryAndDeadLetterEvidence(t *testing.T) {
	snap := Snapshot{
		GeneratedAt: "2026-03-19T00:00:00Z",
		Transports: []TransportReport{{
			Name:                "mqtt",
			Type:                "mqtt",
			EffectiveState:      transport.StateConfiguredOffline,
			ReconnectAttempts:   3,
			ConsecutiveTimeouts: 2,
			DeadLetters:         1,
			RetryStatus:         "backoff armed after 3 reconnect attempts",
			Detail:              "broker disconnected; waiting to retry",
		}},
	}
	panel := BuildPanel(snap)
	if panel.OperatorState != "degraded" {
		t.Fatalf("expected degraded operator state, got %+v", panel)
	}
	if panel.Transports[0].Detail == "" {
		t.Fatalf("expected detail to remain populated, got %+v", panel.Transports[0])
	}
}

package operatorlang

import (
	"strings"
	"testing"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
)

func TestTargetSummary_NodeAndTransport(t *testing.T) {
	s := TargetSummary("restart_transport", "mqtt-uplink", "", "!abcd1234")
	if s == "" {
		t.Fatal("empty summary")
	}
	if !containsAll(s, []string{"node", "!abcd1234", "mqtt-uplink"}) {
		t.Fatalf("unexpected summary: %q", s)
	}
}

func TestIncidentIDFromMetadata(t *testing.T) {
	id := IncidentIDFromMetadata(map[string]any{"incident_id": " inc-1 "})
	if id != "inc-1" {
		t.Fatalf("got %q", id)
	}
}

func TestLifecycleOperatorLabels_PendingApproval(t *testing.T) {
	q, a, e := lifecycleOperatorLabels(db.ControlActionRecord{
		LifecycleState: control.LifecyclePendingApproval,
		ExecutionMode:  control.ExecutionModeApprovalRequired,
		Result:         control.ResultPendingApproval,
	})
	if q != "Approver inbox" || a != "Awaiting approver" || e != "Not started" {
		t.Fatalf("got %q %q %q", q, a, e)
	}
}

func TestLifecycleOperatorLabels_ApprovedWaitingExecutor(t *testing.T) {
	q, a, e := lifecycleOperatorLabels(db.ControlActionRecord{
		LifecycleState: control.LifecyclePending,
		ExecutionMode:  control.ExecutionModeApprovalRequired,
		Result:         control.ResultApproved,
	})
	if q != "Executor queue" || a != "Approved" || e != "Waiting for executor" {
		t.Fatalf("got %q %q %q", q, a, e)
	}
}

func containsAll(s string, parts []string) bool {
	for _, p := range parts {
		if p == "" {
			continue
		}
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

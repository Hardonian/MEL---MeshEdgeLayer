package service

import (
	"context"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/transport"
)

func TestIncidentEscalation(t *testing.T) {
	tc := config.TransportConfig{Name: "mqtt-primary", Type: "mqtt", Enabled: true}
	app := newTestApp(t, tc)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.startWorkers(ctx)
	
	// Create a severe observation (retry threshold exceeded is an error level)
	obs := transport.NewObservation(tc.Name, tc.Type, "topic", transport.ReasonRetryThresholdExceeded, []byte("locked"), false, "access denied by broker", map[string]any{"code": 403, "final": true})
	
	// Publish to Bus
	app.Bus.Publish(events.Event{Type: "transport.observation", Data: obs})
	
	// Poll for outcome
	found := false
	for i := 0; i < 40; i++ {
		time.Sleep(100 * time.Millisecond)
		incidents, _ := app.DB.RecentIncidents(10)
		for _, inc := range incidents {
			if inc.ResourceID == tc.Name && inc.Category == "transport" && inc.State == "open" {
				found = true
				break
			}
		}
		if found { break }
	}

	if !found {
		incidents, _ := app.DB.RecentIncidents(100)
		t.Errorf("No incident found in DB! Found %d incidents.", len(incidents))
		for i, inc := range incidents {
			t.Logf("INC[%d]: %+v", i, inc)
		}
		audit, _ := app.DB.QueryRows("SELECT category, message FROM audit_logs;")
		t.Logf("Audit logs: %+v", audit)
		
		// Also check if any messages made it to the ingest worker? No, it's observation.
		// Let's check if observationWorker is even running.
	}
}

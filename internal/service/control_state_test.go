package service

import (
	"testing"
	"time"
)

func TestTransportControlState_suppressAndClear(t *testing.T) {
	st := newTransportControlState()
	until := time.Now().UTC().Add(5 * time.Minute)
	st.setSuppressNode(42, until)
	if !st.shouldDropSuppressed(42, time.Now().UTC()) {
		t.Fatal("expected suppression active for node 42")
	}
	if st.shouldDropSuppressed(99, time.Now().UTC()) {
		t.Fatal("expected other nodes not suppressed")
	}
	st.clearIngestActuators()
	if st.shouldDropSuppressed(42, time.Now().UTC()) {
		t.Fatal("expected suppression cleared")
	}
}

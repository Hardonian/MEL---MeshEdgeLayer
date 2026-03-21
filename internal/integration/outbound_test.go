package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeliverWebhookRetries(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		b, _ := io.ReadAll(r.Body)
		var got Event
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatal(err)
		}
		if got.SchemaVersion != "mel.integration.v1" {
			t.Fatalf("schema: %+v", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{}
	res := c.DeliverWebhook(context.Background(), srv.URL, Event{
		SchemaVersion: "mel.integration.v1",
		EventType:     "test",
		Timestamp:     "2026-01-01T00:00:00Z",
		Source:        "test",
		Summary:       "hi",
	})
	if !res.Success || res.Attempts != 3 {
		t.Fatalf("unexpected result: %+v attempts=%d", res, attempts)
	}
	if attempts < 2 {
		t.Fatalf("expected retry, attempts=%d", attempts)
	}
}

package meshintel

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func TestRollupRecentMessagesHistograms(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "sig.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Anchor to wall clock so the 24h rollup window always includes these rows.
	base := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	for i := 0; i < 20; i++ {
		_, err := d.InsertMessage(map[string]any{
			"transport_name": "t",
			"packet_id":      int64(i),
			"dedupe_hash":    fmt.Sprintf("h%d", i),
			"raw_hex":        "00",
			"rx_time":        base,
			"from_node":      int64(1),
			"to_node":        int64(2),
			"hop_limit":      int64(i % 4),
			"relay_node":     int64(99),
			"portnum":        int64(1 + i%3),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sig, err := RollupRecentMessages(d, 24*time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	if sig.TotalMessages < 20 {
		t.Fatalf("total messages: %d", sig.TotalMessages)
	}
	if len(sig.HopBuckets) == 0 {
		t.Fatal("expected hop buckets")
	}
	if len(sig.PortnumBuckets) == 0 {
		t.Fatal("expected portnum buckets")
	}
	if sig.RebroadcastPathProxy < 0.9 {
		t.Fatalf("rebroadcast proxy: %v", sig.RebroadcastPathProxy)
	}
}

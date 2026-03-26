package meshintel

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/db"
)

// TransportLikelyConnectedFromRuntime uses persisted transport_runtime_status when no live transport handles exist (CLI).
func TransportLikelyConnectedFromRuntime(d *db.DB) bool {
	if d == nil {
		return false
	}
	rows, err := d.QueryRows(`SELECT enabled, runtime_state FROM transport_runtime_status;`)
	if err != nil {
		return false
	}
	for _, row := range rows {
		if dbAsInt64(row["enabled"]) == 0 {
			continue
		}
		st := strings.ToLower(strings.TrimSpace(fmt.Sprint(row["runtime_state"])))
		if st == "live" || st == "idle" {
			return true
		}
	}
	return false
}

// RollupRecentMessages aggregates messages table rows for routing-pressure proxies.
func RollupRecentMessages(d *db.DB, window time.Duration, transportConnected bool) (MessageSignals, error) {
	sig := MessageSignals{
		WindowDescription:  fmt.Sprintf("last_%s", window),
		TransportConnected: transportConnected,
	}
	if d == nil {
		return sig, nil
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	cutoff := time.Now().UTC().Add(-window).Format(time.RFC3339)

	rows, err := d.QueryRows(fmt.Sprintf(`
SELECT
  COUNT(*) AS total,
  SUM(CASE WHEN relay_node IS NOT NULL AND relay_node != 0 THEN 1 ELSE 0 END) AS with_relay,
  SUM(CASE WHEN hop_limit IS NOT NULL AND hop_limit > 0 THEN 1 ELSE 0 END) AS with_hop,
  AVG(CASE WHEN hop_limit IS NOT NULL AND hop_limit >= 0 THEN hop_limit ELSE NULL END) AS avg_hop,
  MAX(CASE WHEN hop_limit IS NOT NULL THEN hop_limit ELSE 0 END) AS max_hop,
  COUNT(DISTINCT from_node) AS distinct_from
FROM messages
WHERE rx_time >= '%s';
`, cutoff))
	if err != nil {
		return sig, err
	}
	if len(rows) > 0 {
		r := rows[0]
		sig.TotalMessages = dbAsInt64(r["total"])
		sig.MessagesWithRelay = dbAsInt64(r["with_relay"])
		sig.MessagesWithHop = dbAsInt64(r["with_hop"])
		sig.AvgHopLimit = dbAsFloat(r["avg_hop"])
		sig.MaxHopLimit = int(dbAsInt64(r["max_hop"]))
		sig.DistinctFromNodes = int(dbAsInt64(r["distinct_from"]))
	}

	// Duplicate relay hotspot: max share of packets sharing the same relay_node among relayed packets
	rows2, err := d.QueryRows(fmt.Sprintf(`
SELECT relay_node, COUNT(*) AS c
FROM messages
WHERE rx_time >= '%s' AND relay_node IS NOT NULL AND relay_node != 0
GROUP BY relay_node
ORDER BY c DESC
LIMIT 1;
`, cutoff))
	if err != nil {
		return sig, err
	}
	if len(rows2) > 0 && sig.MessagesWithRelay > 0 {
		top := dbAsInt64(rows2[0]["c"])
		sig.DuplicateRelayHotspot = float64(top) / float64(sig.MessagesWithRelay)
		sig.RelayMaxShare = sig.DuplicateRelayHotspot
	}

	rowsRelayN, err := d.QueryRows(fmt.Sprintf(`
SELECT COUNT(DISTINCT relay_node) AS n FROM messages
WHERE rx_time >= '%s' AND relay_node IS NOT NULL AND relay_node != 0;
`, cutoff))
	if err != nil {
		return sig, err
	}
	if len(rowsRelayN) > 0 {
		sig.DistinctRelayNodes = int(dbAsInt64(rowsRelayN[0]["n"]))
	}

	if sig.TotalMessages > 0 {
		sig.RebroadcastPathProxy = float64(sig.MessagesWithRelay) / float64(sig.TotalMessages)
	}

	rowsHop, err := d.QueryRows(fmt.Sprintf(`
SELECT hop_limit, COUNT(*) AS c FROM messages
WHERE rx_time >= '%s' AND hop_limit IS NOT NULL
GROUP BY hop_limit ORDER BY c DESC LIMIT 12;
`, cutoff))
	if err != nil {
		return sig, err
	}
	for _, row := range rowsHop {
		sig.HopBuckets = append(sig.HopBuckets, HistogramBucket{
			Key:   fmt.Sprintf("hop_%d", int(dbAsInt64(row["hop_limit"]))),
			Count: dbAsInt64(row["c"]),
		})
	}

	rowsPort, err := d.QueryRows(fmt.Sprintf(`
SELECT portnum, COUNT(*) AS c FROM messages
WHERE rx_time >= '%s' AND portnum IS NOT NULL
GROUP BY portnum ORDER BY c DESC LIMIT 10;
`, cutoff))
	if err != nil {
		return sig, err
	}
	for _, row := range rowsPort {
		sig.PortnumBuckets = append(sig.PortnumBuckets, HistogramBucket{
			Key:   fmt.Sprintf("port_%d", int(dbAsInt64(row["portnum"]))),
			Count: dbAsInt64(row["c"]),
		})
	}

	return sig, nil
}

func dbAsInt64(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case nil:
		return 0
	default:
		var n int64
		_, _ = fmt.Sscan(fmt.Sprint(v), &n)
		return n
	}
}

func dbAsFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case nil:
		return 0
	default:
		var f float64
		_, _ = fmt.Sscan(fmt.Sprint(v), &f)
		return f
	}
}

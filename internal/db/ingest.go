package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mel-project/mel/internal/selfobs"
)

type IngestRecord struct {
	Message        map[string]any
	Node           map[string]any
	TelemetryType  string
	TelemetryValue any
	ObservedAt     string
}

func (d *DB) PersistIngest(record IngestRecord) (bool, error) {
	message := record.Message
	if strings.TrimSpace(asString(message["transport_name"])) == "" || strings.TrimSpace(asString(message["dedupe_hash"])) == "" || strings.TrimSpace(asString(message["raw_hex"])) == "" {
		return false, fmt.Errorf("ingest record missing required message identity fields")
	}
	payloadJSON, _ := json.Marshal(message["payload_json"])
	telemetryJSON, _ := json.Marshal(record.TelemetryValue)
	telemetrySQL := ""
	if record.TelemetryType != "" {
		telemetrySQL = fmt.Sprintf(`INSERT INTO telemetry_samples(node_num,sample_type,value_json,observed_at)
SELECT %d,'%s','%s','%s' FROM _mel_ingest_state WHERE message_inserted > 0;`,
			asInt(record.Node["node_num"]), esc(record.TelemetryType), esc(string(telemetryJSON)), esc(record.ObservedAt))
	}
	rows, err := d.QueryRows(fmt.Sprintf(`BEGIN IMMEDIATE;
CREATE TEMP TABLE IF NOT EXISTS _mel_ingest_state(message_inserted INTEGER NOT NULL);
DELETE FROM _mel_ingest_state;
INSERT OR IGNORE INTO messages(transport_name,packet_id,dedupe_hash,channel_id,gateway_id,from_node,to_node,portnum,payload_text,payload_json,raw_hex,rx_time,hop_limit,relay_node)
VALUES('%s',%d,'%s','%s','%s',%d,%d,%d,'%s','%s','%s','%s',%d,%d);
INSERT INTO _mel_ingest_state(message_inserted) VALUES (changes());
INSERT INTO nodes(node_num,node_id,long_name,short_name,last_seen,last_gateway_id,last_snr,last_rssi,lat_redacted,lon_redacted,altitude,updated_at)
SELECT %d,'%s','%s','%s','%s','%s',%f,%d,%f,%f,%d,'%s'
FROM _mel_ingest_state WHERE message_inserted > 0
ON CONFLICT(node_num) DO UPDATE SET node_id=excluded.node_id,long_name=excluded.long_name,short_name=excluded.short_name,last_seen=excluded.last_seen,last_gateway_id=excluded.last_gateway_id,last_snr=excluded.last_snr,last_rssi=excluded.last_rssi,lat_redacted=excluded.lat_redacted,lon_redacted=excluded.lon_redacted,altitude=excluded.altitude,updated_at=excluded.updated_at;
%s
COMMIT;
SELECT COALESCE(MAX(message_inserted),0) AS message_inserted FROM _mel_ingest_state;`,
		esc(asString(message["transport_name"])), asInt(message["packet_id"]), esc(asString(message["dedupe_hash"])), esc(asString(message["channel_id"])), esc(asString(message["gateway_id"])), asInt(message["from_node"]), asInt(message["to_node"]), asInt(message["portnum"]), esc(asString(message["payload_text"])), esc(string(payloadJSON)), esc(asString(message["raw_hex"])), esc(asString(message["rx_time"])), asInt(message["hop_limit"]), asInt(message["relay_node"]),
		asInt(record.Node["node_num"]), esc(asString(record.Node["node_id"])), esc(asString(record.Node["long_name"])), esc(asString(record.Node["short_name"])), esc(asString(record.Node["last_seen"])), esc(asString(record.Node["last_gateway_id"])), asFloat(record.Node["last_snr"]), asInt(record.Node["last_rssi"]), asFloat(record.Node["lat_redacted"]), asFloat(record.Node["lon_redacted"]), asInt(record.Node["altitude"]), esc(record.ObservedAt), telemetrySQL))
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	if asInt(rows[0]["message_inserted"]) > 0 {
		selfobs.MarkFresh("ingest")
	}
	return asInt(rows[0]["message_inserted"]) > 0, nil
}

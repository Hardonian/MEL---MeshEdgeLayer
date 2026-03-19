package db

import (
	"fmt"
	"sort"
	"strings"
)

func (d *DB) RecentTransportNodeAttribution(start, end string, perTransportLimit int) (map[string][]string, error) {
	if d == nil {
		return map[string][]string{}, nil
	}
	if perTransportLimit <= 0 {
		perTransportLimit = 5
	}
	clauses := []string{"m.transport_name != ''", "m.from_node > 0"}
	if strings.TrimSpace(start) != "" {
		clauses = append(clauses, fmt.Sprintf("m.rx_time >= '%s'", esc(start)))
	}
	if strings.TrimSpace(end) != "" {
		clauses = append(clauses, fmt.Sprintf("m.rx_time <= '%s'", esc(end)))
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT m.transport_name,
       COALESCE(NULLIF(n.node_id,''), CAST(m.from_node AS TEXT)) AS attributed_node_id,
       COUNT(*) AS message_count
FROM messages m
LEFT JOIN nodes n ON n.node_num = m.from_node
WHERE %s
GROUP BY m.transport_name, attributed_node_id
ORDER BY m.transport_name, message_count DESC, attributed_node_id ASC;`, strings.Join(clauses, " AND ")))
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, row := range rows {
		name := asString(row["transport_name"])
		if len(out[name]) >= perTransportLimit {
			continue
		}
		nodeID := asString(row["attributed_node_id"])
		if strings.TrimSpace(nodeID) == "" {
			continue
		}
		out[name] = append(out[name], nodeID)
	}
	for name := range out {
		sort.Strings(out[name])
	}
	return out, nil
}

package topology

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/db"
)

// Store provides topology persistence operations backed by SQLite.
type Store struct {
	DB *db.DB
}

// NewStore creates a topology store.
func NewStore(d *db.DB) *Store {
	return &Store{DB: d}
}

// ListNodes returns all nodes with topology-enriched fields. Bounded by limit.
func (s *Store) ListNodes(limit int) ([]Node, error) {
	if limit <= 0 || limit > 10000 {
		limit = 500
	}
	rows, err := s.DB.QueryRows(fmt.Sprintf(`SELECT node_num, COALESCE(node_id,'') AS node_id, COALESCE(long_name,'') AS long_name, COALESCE(short_name,'') AS short_name,
		COALESCE(role,'') AS role, COALESCE(hardware_model,'') AS hardware_model,
		COALESCE(first_seen_at,'') AS first_seen_at, COALESCE(last_seen,'') AS last_seen,
		COALESCE(last_direct_seen_at,'') AS last_direct_seen_at, COALESCE(last_broker_seen_at,'') AS last_broker_seen_at,
		COALESCE(last_gateway_id,'') AS last_gateway_id,
		COALESCE(trust_class,'unknown') AS trust_class, COALESCE(location_state,'unknown') AS location_state,
		COALESCE(mobility_state,'unknown') AS mobility_state, COALESCE(health_state,'unknown') AS health_state,
		COALESCE(health_score,0) AS health_score, COALESCE(health_factors_json,'') AS health_factors_json,
		COALESCE(lat_redacted,0) AS lat_redacted, COALESCE(lon_redacted,0) AS lon_redacted,
		COALESCE(altitude,0) AS altitude, COALESCE(last_snr,0) AS last_snr, COALESCE(last_rssi,0) AS last_rssi,
		COALESCE(stale,0) AS stale, COALESCE(quarantined,0) AS quarantined, COALESCE(quarantine_reason,'') AS quarantine_reason,
		COALESCE(source_connector_id,'') AS source_connector_id
		FROM nodes ORDER BY last_seen DESC LIMIT %d;`, limit))
	if err != nil {
		return nil, err
	}
	return rowsToNodes(rows), nil
}

// GetNode returns a single node by node_num.
func (s *Store) GetNode(nodeNum int64) (Node, bool, error) {
	rows, err := s.DB.QueryRows(fmt.Sprintf(`SELECT node_num, COALESCE(node_id,'') AS node_id, COALESCE(long_name,'') AS long_name, COALESCE(short_name,'') AS short_name,
		COALESCE(role,'') AS role, COALESCE(hardware_model,'') AS hardware_model,
		COALESCE(first_seen_at,'') AS first_seen_at, COALESCE(last_seen,'') AS last_seen,
		COALESCE(last_direct_seen_at,'') AS last_direct_seen_at, COALESCE(last_broker_seen_at,'') AS last_broker_seen_at,
		COALESCE(last_gateway_id,'') AS last_gateway_id,
		COALESCE(trust_class,'unknown') AS trust_class, COALESCE(location_state,'unknown') AS location_state,
		COALESCE(mobility_state,'unknown') AS mobility_state, COALESCE(health_state,'unknown') AS health_state,
		COALESCE(health_score,0) AS health_score, COALESCE(health_factors_json,'') AS health_factors_json,
		COALESCE(lat_redacted,0) AS lat_redacted, COALESCE(lon_redacted,0) AS lon_redacted,
		COALESCE(altitude,0) AS altitude, COALESCE(last_snr,0) AS last_snr, COALESCE(last_rssi,0) AS last_rssi,
		COALESCE(stale,0) AS stale, COALESCE(quarantined,0) AS quarantined, COALESCE(quarantine_reason,'') AS quarantine_reason,
		COALESCE(source_connector_id,'') AS source_connector_id
		FROM nodes WHERE node_num=%d LIMIT 1;`, nodeNum))
	if err != nil {
		return Node{}, false, err
	}
	if len(rows) == 0 {
		return Node{}, false, nil
	}
	nodes := rowsToNodes(rows)
	return nodes[0], true, nil
}

// GetLink returns a link by edge_id.
func (s *Store) GetLink(edgeID string) (Link, bool, error) {
	if strings.TrimSpace(edgeID) == "" {
		return Link{}, false, nil
	}
	rows, err := s.DB.QueryRows(fmt.Sprintf(`SELECT edge_id, src_node_num, dst_node_num, observed, directional,
		COALESCE(transport_path,'') AS transport_path, first_observed_at, last_observed_at,
		quality_score, reliability, intermittence_count, source_trust_level,
		COALESCE(source_connector_id,'') AS source_connector_id,
		stale, contradiction, COALESCE(contradiction_detail,'') AS contradiction_detail,
		relay_dependent, COALESCE(quality_factors_json,'') AS quality_factors_json, observation_count
		FROM topology_links WHERE edge_id='%s' LIMIT 1;`, db.EscString(edgeID)))
	if err != nil {
		return Link{}, false, err
	}
	if len(rows) == 0 {
		return Link{}, false, nil
	}
	links := rowsToLinks(rows)
	return links[0], true, nil
}

// ListLinks returns topology links, bounded by limit.
func (s *Store) ListLinks(limit int) ([]Link, error) {
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}
	rows, err := s.DB.QueryRows(fmt.Sprintf(`SELECT edge_id, src_node_num, dst_node_num, observed, directional,
		COALESCE(transport_path,'') AS transport_path, first_observed_at, last_observed_at,
		quality_score, reliability, intermittence_count, source_trust_level,
		COALESCE(source_connector_id,'') AS source_connector_id,
		stale, contradiction, COALESCE(contradiction_detail,'') AS contradiction_detail,
		relay_dependent, COALESCE(quality_factors_json,'') AS quality_factors_json, observation_count
		FROM topology_links ORDER BY last_observed_at DESC LIMIT %d;`, limit))
	if err != nil {
		return nil, err
	}
	return rowsToLinks(rows), nil
}

// LinksForNode returns links connected to a specific node.
func (s *Store) LinksForNode(nodeNum int64) ([]Link, error) {
	rows, err := s.DB.QueryRows(fmt.Sprintf(`SELECT edge_id, src_node_num, dst_node_num, observed, directional,
		COALESCE(transport_path,'') AS transport_path, first_observed_at, last_observed_at,
		quality_score, reliability, intermittence_count, source_trust_level,
		COALESCE(source_connector_id,'') AS source_connector_id,
		stale, contradiction, COALESCE(contradiction_detail,'') AS contradiction_detail,
		relay_dependent, COALESCE(quality_factors_json,'') AS quality_factors_json, observation_count
		FROM topology_links WHERE src_node_num=%d OR dst_node_num=%d
		ORDER BY last_observed_at DESC LIMIT 500;`, nodeNum, nodeNum))
	if err != nil {
		return nil, err
	}
	return rowsToLinks(rows), nil
}

// UpsertLink creates or updates a topology link.
func (s *Store) UpsertLink(l Link) error {
	factorsJSON := "[]"
	if len(l.QualityFactors) > 0 {
		b, _ := json.Marshal(l.QualityFactors)
		factorsJSON = string(b)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	observed := 0
	if l.Observed {
		observed = 1
	}
	directional := 0
	if l.Directional {
		directional = 1
	}
	stale := 0
	if l.Stale {
		stale = 1
	}
	contradiction := 0
	if l.Contradiction {
		contradiction = 1
	}
	relayDep := 0
	if l.RelayDependent {
		relayDep = 1
	}
	sql := fmt.Sprintf(`INSERT INTO topology_links(edge_id, src_node_num, dst_node_num, observed, directional, transport_path, first_observed_at, last_observed_at, quality_score, reliability, intermittence_count, source_trust_level, source_connector_id, stale, contradiction, contradiction_detail, relay_dependent, quality_factors_json, observation_count, updated_at)
		VALUES('%s',%d,%d,%d,%d,'%s','%s','%s',%f,%f,%d,%f,'%s',%d,%d,'%s',%d,'%s',%d,'%s')
		ON CONFLICT(src_node_num, dst_node_num, transport_path) DO UPDATE SET
		observed=excluded.observed, last_observed_at=excluded.last_observed_at, quality_score=excluded.quality_score,
		reliability=excluded.reliability, intermittence_count=excluded.intermittence_count,
		source_trust_level=excluded.source_trust_level, stale=excluded.stale,
		contradiction=excluded.contradiction, contradiction_detail=excluded.contradiction_detail,
		relay_dependent=excluded.relay_dependent, quality_factors_json=excluded.quality_factors_json,
		observation_count=topology_links.observation_count+1, updated_at=excluded.updated_at;`,
		db.EscString(l.EdgeID), l.SrcNodeNum, l.DstNodeNum, observed, directional,
		db.EscString(l.TransportPath), db.EscString(l.FirstObservedAt), db.EscString(l.LastObservedAt),
		l.QualityScore, l.Reliability, l.IntermittenceCount, l.SourceTrustLevel,
		db.EscString(l.SourceConnectorID), stale, contradiction,
		db.EscString(l.ContradictionDetail), relayDep,
		db.EscString(factorsJSON), l.ObservationCount, now)
	return s.DB.Exec(sql)
}

// UpdateNodeHealth updates health scoring fields for a node.
func (s *Store) UpdateNodeHealth(nodeNum int64, score float64, state HealthState, factors []ScoreFactor, staleFlag bool) error {
	factorsJSON := "[]"
	if len(factors) > 0 {
		b, _ := json.Marshal(factors)
		factorsJSON = string(b)
	}
	staleInt := 0
	if staleFlag {
		staleInt = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	sql := fmt.Sprintf(`UPDATE nodes SET health_score=%f, health_state='%s', health_factors_json='%s', stale=%d, updated_at='%s' WHERE node_num=%d;`,
		score, db.EscString(string(state)), db.EscString(factorsJSON), staleInt, now, nodeNum)
	return s.DB.Exec(sql)
}

// RecordNodeScoreHistory inserts a score history entry.
func (s *Store) RecordNodeScoreHistory(nodeNum int64, score float64, state HealthState, factors []ScoreFactor) error {
	factorsJSON := "[]"
	if len(factors) > 0 {
		b, _ := json.Marshal(factors)
		factorsJSON = string(b)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	sql := fmt.Sprintf(`INSERT INTO node_score_history(node_num, health_score, health_state, factors_json, recorded_at) VALUES(%d,%f,'%s','%s','%s');`,
		nodeNum, score, db.EscString(string(state)), db.EscString(factorsJSON), now)
	return s.DB.Exec(sql)
}

// PruneSnapshots keeps only the newest limit rows by created_at.
func (s *Store) PruneSnapshots(limit int) error {
	if s == nil || s.DB == nil || limit <= 0 {
		return nil
	}
	sql := fmt.Sprintf(`DELETE FROM topology_snapshots WHERE rowid NOT IN (
  SELECT rowid FROM topology_snapshots ORDER BY created_at DESC LIMIT %d
);`, limit)
	return s.DB.Exec(sql)
}

// SaveSnapshot persists a topology snapshot.
func (s *Store) SaveSnapshot(snap TopologySnapshot) error {
	sourceJSON, _ := json.Marshal(snap.SourceCoverage)
	confJSON, _ := json.Marshal(snap.ConfidenceSummary)
	explJSON, _ := json.Marshal(snap.Explanation)
	regionJSON, _ := json.Marshal(snap.RegionSummary)
	sql := fmt.Sprintf(`INSERT INTO topology_snapshots(snapshot_id, created_at, node_count, edge_count, direct_edge_count, inferred_edge_count, healthy_nodes, degraded_nodes, stale_nodes, isolated_nodes, graph_hash, source_coverage_json, confidence_summary_json, explanation_json, region_summary_json)
		VALUES('%s','%s',%d,%d,%d,%d,%d,%d,%d,%d,'%s','%s','%s','%s','%s');`,
		db.EscString(snap.SnapshotID), db.EscString(snap.CreatedAt),
		snap.NodeCount, snap.EdgeCount, snap.DirectEdgeCount, snap.InferredEdgeCount,
		snap.HealthyNodes, snap.DegradedNodes, snap.StaleNodes, snap.IsolatedNodes,
		db.EscString(snap.GraphHash), db.EscString(string(sourceJSON)),
		db.EscString(string(confJSON)), db.EscString(string(explJSON)),
		db.EscString(string(regionJSON)))
	return s.DB.Exec(sql)
}

// RecentSnapshots returns the most recent topology snapshots.
func (s *Store) RecentSnapshots(limit int) ([]TopologySnapshot, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.DB.QueryRows(fmt.Sprintf(`SELECT snapshot_id, created_at, node_count, edge_count, direct_edge_count, inferred_edge_count, healthy_nodes, degraded_nodes, stale_nodes, isolated_nodes, COALESCE(graph_hash,'') AS graph_hash, COALESCE(region_summary_json,'[]') AS region_summary_json FROM topology_snapshots ORDER BY created_at DESC LIMIT %d;`, limit))
	if err != nil {
		return nil, err
	}
	var snaps []TopologySnapshot
	for _, row := range rows {
		snap := TopologySnapshot{
			SnapshotID:        asString(row["snapshot_id"]),
			CreatedAt:         asString(row["created_at"]),
			NodeCount:         int(asInt(row["node_count"])),
			EdgeCount:         int(asInt(row["edge_count"])),
			DirectEdgeCount:   int(asInt(row["direct_edge_count"])),
			InferredEdgeCount: int(asInt(row["inferred_edge_count"])),
			HealthyNodes:      int(asInt(row["healthy_nodes"])),
			DegradedNodes:     int(asInt(row["degraded_nodes"])),
			StaleNodes:        int(asInt(row["stale_nodes"])),
			IsolatedNodes:     int(asInt(row["isolated_nodes"])),
			GraphHash:         asString(row["graph_hash"]),
		}
		_ = json.Unmarshal([]byte(asString(row["region_summary_json"])), &snap.RegionSummary)
		snaps = append(snaps, snap)
	}
	return snaps, nil
}

// InsertObservation records a raw node observation.
func (s *Store) InsertObservation(obs NodeObservation) error {
	viaMQTT := 0
	if obs.ViaMQTT {
		viaMQTT = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	dedupeKey := fmt.Sprintf("%d-%s-%s", obs.NodeNum, obs.ConnectorID, obs.ObservedAt)
	sql := fmt.Sprintf(`INSERT OR IGNORE INTO node_observations(node_num, connector_id, source_type, trust_level, observed_at, snr, rssi, lat, lon, altitude, hop_count, via_mqtt, gateway_id, dedupe_key, created_at)
		VALUES(%d,'%s','%s',%f,'%s',%f,%d,%f,%f,%d,%d,%d,'%s','%s','%s');`,
		obs.NodeNum, db.EscString(obs.ConnectorID), db.EscString(obs.SourceType),
		obs.TrustLevel, db.EscString(obs.ObservedAt),
		obs.SNR, obs.RSSI, obs.Lat, obs.Lon, obs.Altitude, obs.HopCount,
		viaMQTT, db.EscString(obs.GatewayID), db.EscString(dedupeKey), now)
	return s.DB.Exec(sql)
}

// RecentObservations returns recent observations for a node.
func (s *Store) RecentObservations(nodeNum int64, limit int) ([]NodeObservation, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.DB.QueryRows(fmt.Sprintf(`SELECT node_num, connector_id, source_type, trust_level, observed_at, COALESCE(snr,0) AS snr, COALESCE(rssi,0) AS rssi, COALESCE(lat,0) AS lat, COALESCE(lon,0) AS lon, COALESCE(altitude,0) AS altitude, COALESCE(hop_count,0) AS hop_count, via_mqtt, COALESCE(gateway_id,'') AS gateway_id FROM node_observations WHERE node_num=%d ORDER BY observed_at DESC LIMIT %d;`, nodeNum, limit))
	if err != nil {
		return nil, err
	}
	var obs []NodeObservation
	for _, row := range rows {
		obs = append(obs, NodeObservation{
			NodeNum:     asInt(row["node_num"]),
			ConnectorID: asString(row["connector_id"]),
			SourceType:  asString(row["source_type"]),
			TrustLevel:  asFloat(row["trust_level"]),
			ObservedAt:  asString(row["observed_at"]),
			SNR:         asFloat(row["snr"]),
			RSSI:        asInt(row["rssi"]),
			Lat:         asFloat(row["lat"]),
			Lon:         asFloat(row["lon"]),
			Altitude:    asInt(row["altitude"]),
			HopCount:    int(asInt(row["hop_count"])),
			ViaMQTT:     asInt(row["via_mqtt"]) == 1,
			GatewayID:   asString(row["gateway_id"]),
		})
	}
	return obs, nil
}

// UpsertBookmark creates or updates a node bookmark.
func (s *Store) UpsertBookmark(bm Bookmark) error {
	now := time.Now().UTC().Format(time.RFC3339)
	active := 0
	if bm.Active {
		active = 1
	}
	sql := fmt.Sprintf(`INSERT INTO node_bookmarks(node_num, bookmark_type, label, notes, actor_id, created_at, updated_at, active)
		VALUES(%d,'%s','%s','%s','%s','%s','%s',%d)
		ON CONFLICT(node_num, bookmark_type) DO UPDATE SET label=excluded.label, notes=excluded.notes, actor_id=excluded.actor_id, updated_at=excluded.updated_at, active=excluded.active;`,
		bm.NodeNum, db.EscString(bm.BookmarkType), db.EscString(bm.Label),
		db.EscString(bm.Notes), db.EscString(bm.ActorID), now, now, active)
	return s.DB.Exec(sql)
}

// ListBookmarks returns active bookmarks, optionally filtered by type.
func (s *Store) ListBookmarks(bookmarkType string, limit int) ([]Bookmark, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	where := "active=1"
	if bookmarkType != "" {
		where += fmt.Sprintf(" AND bookmark_type='%s'", db.EscString(bookmarkType))
	}
	rows, err := s.DB.QueryRows(fmt.Sprintf(`SELECT id, node_num, bookmark_type, COALESCE(label,'') AS label, COALESCE(notes,'') AS notes, actor_id, created_at, updated_at, active FROM node_bookmarks WHERE %s ORDER BY updated_at DESC LIMIT %d;`, where, limit))
	if err != nil {
		return nil, err
	}
	var bookmarks []Bookmark
	for _, row := range rows {
		bookmarks = append(bookmarks, Bookmark{
			ID:           asInt(row["id"]),
			NodeNum:      asInt(row["node_num"]),
			BookmarkType: asString(row["bookmark_type"]),
			Label:        asString(row["label"]),
			Notes:        asString(row["notes"]),
			ActorID:      asString(row["actor_id"]),
			CreatedAt:    asString(row["created_at"]),
			UpdatedAt:    asString(row["updated_at"]),
			Active:       asInt(row["active"]) == 1,
		})
	}
	return bookmarks, nil
}

// BookmarksForNode returns bookmarks for a specific node.
func (s *Store) BookmarksForNode(nodeNum int64) ([]Bookmark, error) {
	rows, err := s.DB.QueryRows(fmt.Sprintf(`SELECT id, node_num, bookmark_type, COALESCE(label,'') AS label, COALESCE(notes,'') AS notes, actor_id, created_at, updated_at, active FROM node_bookmarks WHERE node_num=%d AND active=1 ORDER BY updated_at DESC;`, nodeNum))
	if err != nil {
		return nil, err
	}
	var bookmarks []Bookmark
	for _, row := range rows {
		bookmarks = append(bookmarks, Bookmark{
			ID:           asInt(row["id"]),
			NodeNum:      asInt(row["node_num"]),
			BookmarkType: asString(row["bookmark_type"]),
			Label:        asString(row["label"]),
			Notes:        asString(row["notes"]),
			ActorID:      asString(row["actor_id"]),
			CreatedAt:    asString(row["created_at"]),
			UpdatedAt:    asString(row["updated_at"]),
			Active:       asInt(row["active"]) == 1,
		})
	}
	return bookmarks, nil
}

// DeleteBookmark deactivates a bookmark.
func (s *Store) DeleteBookmark(nodeNum int64, bookmarkType string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return s.DB.Exec(fmt.Sprintf(`UPDATE node_bookmarks SET active=0, updated_at='%s' WHERE node_num=%d AND bookmark_type='%s';`, now, nodeNum, db.EscString(bookmarkType)))
}

// UpsertSourceTrust creates or updates a source trust record.
func (s *Store) UpsertSourceTrust(st SourceTrust) error {
	now := time.Now().UTC().Format(time.RFC3339)
	sql := fmt.Sprintf(`INSERT INTO source_trust(connector_id, connector_name, connector_type, trust_class, trust_level, first_seen_at, last_seen_at, observation_count, contradiction_count, stale_count, operator_notes, updated_at)
		VALUES('%s','%s','%s','%s',%f,'%s','%s',%d,%d,%d,'%s','%s')
		ON CONFLICT(connector_id) DO UPDATE SET connector_name=excluded.connector_name, trust_class=excluded.trust_class, trust_level=excluded.trust_level, last_seen_at=excluded.last_seen_at, observation_count=source_trust.observation_count+1, updated_at=excluded.updated_at;`,
		db.EscString(st.ConnectorID), db.EscString(st.ConnectorName),
		db.EscString(st.ConnectorType), db.EscString(string(st.TrustClass)),
		st.TrustLevel, db.EscString(st.FirstSeenAt), db.EscString(now),
		st.ObservationCount, st.ContradictionCount, st.StaleCount,
		db.EscString(st.OperatorNotes), now)
	return s.DB.Exec(sql)
}

// ListSourceTrust returns all source trust records.
func (s *Store) ListSourceTrust() ([]SourceTrust, error) {
	rows, err := s.DB.QueryRows(`SELECT connector_id, connector_name, connector_type, trust_class, trust_level, first_seen_at, COALESCE(last_seen_at,'') AS last_seen_at, observation_count, contradiction_count, stale_count, COALESCE(operator_notes,'') AS operator_notes FROM source_trust ORDER BY trust_level DESC;`)
	if err != nil {
		return nil, err
	}
	var trusts []SourceTrust
	for _, row := range rows {
		trusts = append(trusts, SourceTrust{
			ConnectorID:        asString(row["connector_id"]),
			ConnectorName:      asString(row["connector_name"]),
			ConnectorType:      asString(row["connector_type"]),
			TrustClass:         TrustClass(asString(row["trust_class"])),
			TrustLevel:         asFloat(row["trust_level"]),
			FirstSeenAt:        asString(row["first_seen_at"]),
			LastSeenAt:         asString(row["last_seen_at"]),
			ObservationCount:   asInt(row["observation_count"]),
			ContradictionCount: asInt(row["contradiction_count"]),
			StaleCount:         asInt(row["stale_count"]),
			OperatorNotes:      asString(row["operator_notes"]),
		})
	}
	return trusts, nil
}

// GetRecoveryState returns the current recovery state.
func (s *Store) GetRecoveryState() (RecoveryState, error) {
	rows, err := s.DB.QueryRows(`SELECT COALESCE(last_clean_shutdown_at,'') AS last_clean_shutdown_at, COALESCE(last_startup_at,'') AS last_startup_at, unclean_shutdown, COALESCE(recovered_jobs_json,'[]') AS recovered_jobs_json, COALESCE(pending_actions_json,'[]') AS pending_actions_json, startup_mode FROM recovery_state WHERE id=1;`)
	if err != nil {
		return RecoveryState{StartupMode: "unknown"}, err
	}
	if len(rows) == 0 {
		return RecoveryState{StartupMode: "normal"}, nil
	}
	row := rows[0]
	rs := RecoveryState{
		LastCleanShutdownAt: asString(row["last_clean_shutdown_at"]),
		LastStartupAt:       asString(row["last_startup_at"]),
		UncleanShutdown:     asInt(row["unclean_shutdown"]) == 1,
		StartupMode:         asString(row["startup_mode"]),
	}
	_ = json.Unmarshal([]byte(asString(row["recovered_jobs_json"])), &rs.RecoveredJobs)
	_ = json.Unmarshal([]byte(asString(row["pending_actions_json"])), &rs.PendingActions)
	return rs, nil
}

// MarkStartup records that MEL has started, detecting unclean shutdown.
func (s *Store) MarkStartup() (uncleanShutdown bool, err error) {
	now := time.Now().UTC().Format(time.RFC3339)
	// Check if last shutdown was clean
	rs, err := s.GetRecoveryState()
	if err != nil {
		return false, err
	}
	// If last_startup_at > last_clean_shutdown_at, previous shutdown was unclean
	unclean := false
	if rs.LastStartupAt != "" && (rs.LastCleanShutdownAt == "" || rs.LastStartupAt > rs.LastCleanShutdownAt) {
		unclean = true
	}
	mode := "normal"
	if unclean {
		mode = "recovery"
	}
	sql := fmt.Sprintf(`UPDATE recovery_state SET last_startup_at='%s', unclean_shutdown=%d, startup_mode='%s', updated_at='%s' WHERE id=1;`, now, boolToInt(unclean), mode, now)
	return unclean, s.DB.Exec(sql)
}

// MarkCleanShutdown records a clean shutdown.
func (s *Store) MarkCleanShutdown() error {
	now := time.Now().UTC().Format(time.RFC3339)
	return s.DB.Exec(fmt.Sprintf(`UPDATE recovery_state SET last_clean_shutdown_at='%s', unclean_shutdown=0, startup_mode='normal', updated_at='%s' WHERE id=1;`, now, now))
}

// MarkStaleness marks nodes and links as stale based on thresholds.
func (s *Store) MarkStaleness(thresholds StaleThresholds) error {
	cutoff := time.Now().UTC().Add(-thresholds.NodeStaleDuration).Format(time.RFC3339)
	sql := fmt.Sprintf(`UPDATE nodes SET stale=1, health_state='stale' WHERE last_seen < '%s' AND stale=0;`, db.EscString(cutoff))
	if err := s.DB.Exec(sql); err != nil {
		return err
	}
	linkCutoff := time.Now().UTC().Add(-thresholds.LinkStaleDuration).Format(time.RFC3339)
	sql = fmt.Sprintf(`UPDATE topology_links SET stale=1 WHERE last_observed_at < '%s' AND stale=0;`, db.EscString(linkCutoff))
	return s.DB.Exec(sql)
}

// helper converters
func rowsToNodes(rows []map[string]any) []Node {
	nodes := make([]Node, 0, len(rows))
	for _, row := range rows {
		n := Node{
			NodeNum:          asInt(row["node_num"]),
			NodeID:           asString(row["node_id"]),
			LongName:         asString(row["long_name"]),
			ShortName:        asString(row["short_name"]),
			Role:             asString(row["role"]),
			HardwareModel:    asString(row["hardware_model"]),
			FirstSeenAt:      asString(row["first_seen_at"]),
			LastSeenAt:       asString(row["last_seen"]),
			LastDirectSeenAt: asString(row["last_direct_seen_at"]),
			LastBrokerSeenAt: asString(row["last_broker_seen_at"]),
			LastGatewayID:    asString(row["last_gateway_id"]),
			TrustClass:       TrustClass(asString(row["trust_class"])),
			LocationState:    LocationState(asString(row["location_state"])),
			MobilityState:    MobilityState(asString(row["mobility_state"])),
			HealthState:      HealthState(asString(row["health_state"])),
			HealthScore:      asFloat(row["health_score"]),
			LatRedacted:      asFloat(row["lat_redacted"]),
			LonRedacted:      asFloat(row["lon_redacted"]),
			Altitude:         asInt(row["altitude"]),
			LastSNR:          asFloat(row["last_snr"]),
			LastRSSI:         asInt(row["last_rssi"]),
			Stale:            asInt(row["stale"]) == 1,
			Quarantined:      asInt(row["quarantined"]) == 1,
			QuarantineReason: asString(row["quarantine_reason"]),
			SourceConnector:  asString(row["source_connector_id"]),
		}
		factorsJSON := asString(row["health_factors_json"])
		if factorsJSON != "" {
			_ = json.Unmarshal([]byte(factorsJSON), &n.HealthFactors)
		}
		nodes = append(nodes, n)
	}
	return nodes
}

func rowsToLinks(rows []map[string]any) []Link {
	links := make([]Link, 0, len(rows))
	for _, row := range rows {
		l := Link{
			EdgeID:              asString(row["edge_id"]),
			SrcNodeNum:          asInt(row["src_node_num"]),
			DstNodeNum:          asInt(row["dst_node_num"]),
			Observed:            asInt(row["observed"]) == 1,
			Directional:         asInt(row["directional"]) == 1,
			TransportPath:       asString(row["transport_path"]),
			FirstObservedAt:     asString(row["first_observed_at"]),
			LastObservedAt:      asString(row["last_observed_at"]),
			QualityScore:        asFloat(row["quality_score"]),
			Reliability:         asFloat(row["reliability"]),
			IntermittenceCount:  asInt(row["intermittence_count"]),
			SourceTrustLevel:    asFloat(row["source_trust_level"]),
			SourceConnectorID:   asString(row["source_connector_id"]),
			Stale:               asInt(row["stale"]) == 1,
			Contradiction:       asInt(row["contradiction"]) == 1,
			ContradictionDetail: asString(row["contradiction_detail"]),
			RelayDependent:      asInt(row["relay_dependent"]) == 1,
			ObservationCount:    asInt(row["observation_count"]),
		}
		factorsJSON := asString(row["quality_factors_json"])
		if factorsJSON != "" {
			_ = json.Unmarshal([]byte(factorsJSON), &l.QualityFactors)
		}
		links = append(links, l)
	}
	return links
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func asInt(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case string:
		var i int64
		fmt.Sscan(x, &i)
		return i
	}
	var i int64
	fmt.Sscan(fmt.Sprint(v), &i)
	return i
}

func asFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		var f float64
		fmt.Sscan(x, &f)
		return f
	}
	var f float64
	fmt.Sscan(fmt.Sprint(v), &f)
	return f
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// unused but keeping for interface completeness
var _ = strings.TrimSpace

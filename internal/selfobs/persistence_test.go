package selfobs

import (
	"fmt"
	"testing"
	"time"
)

var testStoreCounter int

// setupTestStore creates an in-memory persistence store for testing
// Uses a unique database name per call to ensure isolation between tests
func setupTestStore(t *testing.T) *PersistenceStore {
	t.Helper()
	// Use a unique database name for isolation
	// Each test gets its own in-memory database
	testStoreCounter++
	dbName := fmt.Sprintf("file:memdb%d?mode=memory&cache=shared", testStoreCounter)
	store, err := NewPersistenceStore(dbName, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	})
	return store
}

// TestPersistenceStoreCreation tests store creation and initialization
func TestPersistenceStoreCreation(t *testing.T) {
	t.Run("in_memory_database", func(t *testing.T) {
		store, err := NewPersistenceStore("file:memdb_test1?mode=memory&cache=shared", 7*24*time.Hour)
		if err != nil {
			t.Fatalf("Failed to create in-memory store: %v", err)
		}
		defer store.Close()

		if store.db == nil {
			t.Fatal("Expected database to be initialized")
		}
	})

	t.Run("table_creation", func(t *testing.T) {
		store := setupTestStore(t)

		tables := []string{
			"healing_audit",
			"heat_map_data",
			"health_history",
			"llm_audit",
		}

		for _, table := range tables {
			var count int
			query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
			err := store.db.QueryRow(query, table).Scan(&count)
			if err != nil {
				t.Errorf("Failed to check table %s: %v", table, err)
				continue
			}
			if count != 1 {
				t.Errorf("Expected table %s to exist", table)
			}
		}

		indexes := []string{
			"idx_healing_component",
			"idx_healing_timestamp",
			"idx_heat_component",
			"idx_heat_timestamp",
			"idx_health_component",
			"idx_health_timestamp",
			"idx_llm_timestamp",
			"idx_llm_input_hash",
		}

		for _, index := range indexes {
			var count int
			query := "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?"
			err := store.db.QueryRow(query, index).Scan(&count)
			if err != nil {
				t.Errorf("Failed to check index %s: %v", index, err)
				continue
			}
			if count != 1 {
				t.Errorf("Expected index %s to exist", index)
			}
		}
	})

	t.Run("invalid_path", func(t *testing.T) {
		_, err := NewPersistenceStore("", 7*24*time.Hour)
		if err == nil {
			t.Fatal("Expected error for empty database path")
		}
	})
}

// TestSaveAndGetHealingAction tests saving and retrieving healing actions
func TestSaveAndGetHealingAction(t *testing.T) {
	store := setupTestStore(t)
	now := time.Now()

	t.Run("save_healing_action", func(t *testing.T) {
		action := HealingAction{
			Timestamp:     now,
			Component:     "test-component",
			ActionType:    ActionRetry,
			Success:       true,
			ErrorMessage:  "",
			RetryCount:    1,
			StrategyName:  "retry",
			TransportType: "http",
		}

		err := store.SaveHealingAction(action)
		if err != nil {
			t.Fatalf("Failed to save healing action: %v", err)
		}
	})

	t.Run("get_healing_actions_by_component", func(t *testing.T) {
		action := HealingAction{
			Timestamp:     now.Add(-time.Hour),
			Component:     "test-component",
			ActionType:    ActionRetry,
			Success:       false,
			ErrorMessage:  "connection timeout",
			RetryCount:    2,
			StrategyName:  "retry",
			TransportType: "http",
		}
		err := store.SaveHealingAction(action)
		if err != nil {
			t.Fatalf("Failed to save healing action: %v", err)
		}

		actions, err := store.GetHealingActions("test-component", now.Add(-2*time.Hour))
		if err != nil {
			t.Fatalf("Failed to get healing actions: %v", err)
		}

		if len(actions) < 1 {
			t.Fatalf("Expected at least 1 healing action, got %d", len(actions))
		}

		found := false
		for _, a := range actions {
			if a.Component == "test-component" && a.ErrorMessage == "connection timeout" {
				found = true
				if a.RetryCount != 2 {
					t.Errorf("Expected retry count 2, got %d", a.RetryCount)
				}
				break
			}
		}
		if !found {
			t.Error("Expected to find healing action with error message")
		}
	})

	t.Run("get_all_healing_actions", func(t *testing.T) {
		action1 := HealingAction{
			Timestamp:  now,
			Component:  "component-a",
			ActionType: ActionReconnect,
			Success:    true,
		}
		action2 := HealingAction{
			Timestamp:  now,
			Component:  "component-b",
			ActionType: ActionReset,
			Success:    false,
		}

		if err := store.SaveHealingAction(action1); err != nil {
			t.Fatalf("Failed to save action1: %v", err)
		}
		if err := store.SaveHealingAction(action2); err != nil {
			t.Fatalf("Failed to save action2: %v", err)
		}

		actions, err := store.GetAllHealingActions(now.Add(-time.Hour))
		if err != nil {
			t.Fatalf("Failed to get all healing actions: %v", err)
		}

		foundA, foundB := false, false
		for _, a := range actions {
			if a.Component == "component-a" {
				foundA = true
			}
			if a.Component == "component-b" {
				foundB = true
			}
		}

		if !foundA {
			t.Error("Expected to find component-a")
		}
		if !foundB {
			t.Error("Expected to find component-b")
		}
	})
}

// TestHealingActionStats tests statistics calculation for healing actions
func TestHealingActionStats(t *testing.T) {
	store := setupTestStore(t)
	now := time.Now()

	t.Run("stats_calculation", func(t *testing.T) {
		actions := []HealingAction{
			{Timestamp: now, Component: "stats-component", ActionType: ActionRetry, Success: true, RetryCount: 0},
			{Timestamp: now, Component: "stats-component", ActionType: ActionRetry, Success: true, RetryCount: 1},
			{Timestamp: now, Component: "stats-component", ActionType: ActionRetry, Success: false, RetryCount: 2},
			{Timestamp: now, Component: "stats-component", ActionType: ActionRetry, Success: false, RetryCount: 3},
		}

		for _, action := range actions {
			if err := store.SaveHealingAction(action); err != nil {
				t.Fatalf("Failed to save healing action: %v", err)
			}
		}

		stats, err := store.GetHealingActionStats("stats-component", time.Hour)
		if err != nil {
			t.Fatalf("Failed to get healing action stats: %v", err)
		}

		if stats.TotalActions != 4 {
			t.Errorf("Expected 4 total actions, got %d", stats.TotalActions)
		}
	})

	t.Run("success_rate_computation", func(t *testing.T) {
		actions := []HealingAction{
			{Timestamp: now, Component: "success-component", ActionType: ActionRetry, Success: true, RetryCount: 0},
			{Timestamp: now, Component: "success-component", ActionType: ActionRetry, Success: true, RetryCount: 0},
			{Timestamp: now, Component: "success-component", ActionType: ActionRetry, Success: false, RetryCount: 0},
			{Timestamp: now, Component: "success-component", ActionType: ActionRetry, Success: true, RetryCount: 0},
		}

		for _, action := range actions {
			if err := store.SaveHealingAction(action); err != nil {
				t.Fatalf("Failed to save healing action: %v", err)
			}
		}

		stats, err := store.GetHealingActionStats("success-component", time.Hour)
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		expectedRate := 0.75 // 3 out of 4 successful
		if stats.SuccessRate != expectedRate {
			t.Errorf("Expected success rate %.2f, got %.2f", expectedRate, stats.SuccessRate)
		}
	})

	t.Run("average_retries", func(t *testing.T) {
		actions := []HealingAction{
			{Timestamp: now, Component: "retry-component", ActionType: ActionRetry, Success: true, RetryCount: 0},
			{Timestamp: now, Component: "retry-component", ActionType: ActionRetry, Success: true, RetryCount: 2},
			{Timestamp: now, Component: "retry-component", ActionType: ActionRetry, Success: true, RetryCount: 4},
		}

		for _, action := range actions {
			if err := store.SaveHealingAction(action); err != nil {
				t.Fatalf("Failed to save healing action: %v", err)
			}
		}

		stats, err := store.GetHealingActionStats("retry-component", time.Hour)
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		expectedAvg := 2.0 // (0 + 2 + 4) / 3
		if stats.AvgRetries != expectedAvg {
			t.Errorf("Expected average retries %.2f, got %.2f", expectedAvg, stats.AvgRetries)
		}
	})
}

// TestSaveAndGetHeatData tests saving and retrieving heat data
func TestSaveAndGetHeatData(t *testing.T) {
	store := setupTestStore(t)
	now := time.Now()

	t.Run("save_heat_data", func(t *testing.T) {
		heat := ComponentHeat{
			Timestamp:         now,
			Component:         "heat-component",
			FailureCount:      int64(10),
			RetryCount:        int64(5),
			StateChangeCount:  int64(3),
			RecoveryAttempts:  int64(8),
			RecoverySuccesses: int64(6),
			HeatScore:         75,
		}

		err := store.SaveHeatData("heat-component", heat)
		if err != nil {
			t.Fatalf("Failed to save heat data: %v", err)
		}
	})

	t.Run("get_heat_history", func(t *testing.T) {
		heats := []ComponentHeat{
			{Timestamp: now.Add(-30 * time.Minute), Component: "history-component", FailureCount: int64(5), HeatScore: 50},
			{Timestamp: now.Add(-15 * time.Minute), Component: "history-component", FailureCount: int64(10), HeatScore: 75},
			{Timestamp: now, Component: "history-component", FailureCount: int64(15), HeatScore: 90},
		}

		for _, heat := range heats {
			if err := store.SaveHeatData("history-component", heat); err != nil {
				t.Fatalf("Failed to save heat data: %v", err)
			}
		}

		history, err := store.GetHeatData("history-component", now.Add(-time.Hour))
		if err != nil {
			t.Fatalf("Failed to get heat data: %v", err)
		}

		if len(history) != 3 {
			t.Errorf("Expected 3 heat records, got %d", len(history))
		}

		for _, h := range history {
			if h.Component != "history-component" {
				t.Errorf("Expected component 'history-component', got '%s'", h.Component)
			}
		}
	})

	t.Run("get_hotspots_from_db", func(t *testing.T) {
		// Create a fresh store for isolation
		store := setupTestStore(t)
		now := time.Now()

		heats := []ComponentHeat{
			{Timestamp: now, Component: "hotspot-unique-1", FailureCount: int64(10), HeatScore: 95},
			{Timestamp: now, Component: "hotspot-unique-2", FailureCount: int64(8), HeatScore: 85},
			{Timestamp: now, Component: "cool-spot-unique", FailureCount: int64(2), HeatScore: 25},
		}

		for _, heat := range heats {
			if err := store.SaveHeatData(heat.Component, heat); err != nil {
				t.Fatalf("Failed to save heat data: %v", err)
			}
		}

		hotspots, err := store.GetHotspotsFromDB(80, time.Hour)
		if err != nil {
			t.Fatalf("Failed to get hotspots: %v", err)
		}

		if len(hotspots) != 2 {
			t.Errorf("Expected 2 hotspots, got %d", len(hotspots))
		}

		for _, h := range hotspots {
			if h.HeatScore < 80 {
				t.Errorf("Expected heat score >= 80, got %d", h.HeatScore)
			}
		}
	})
}

// TestHeatTrend tests heat trend calculation
func TestHeatTrend(t *testing.T) {
	store := setupTestStore(t)
	now := time.Now()

	t.Run("trend_calculation", func(t *testing.T) {
		heats := []ComponentHeat{
			{Timestamp: now.Add(-30 * time.Minute), Component: "trend-component", FailureCount: int64(5), HeatScore: 20},
			{Timestamp: now.Add(-20 * time.Minute), Component: "trend-component", FailureCount: int64(10), HeatScore: 40},
			{Timestamp: now.Add(-10 * time.Minute), Component: "trend-component", FailureCount: int64(15), HeatScore: 60},
			{Timestamp: now, Component: "trend-component", FailureCount: int64(20), HeatScore: 80},
		}

		for _, heat := range heats {
			if err := store.SaveHeatData("trend-component", heat); err != nil {
				t.Fatalf("Failed to save heat data: %v", err)
			}
		}

		trend, err := store.GetHeatTrend("trend-component", time.Hour)
		if err != nil {
			t.Fatalf("Failed to get heat trend: %v", err)
		}

		if len(trend) != 4 {
			t.Errorf("Expected 4 trend points, got %d", len(trend))
		}

		for i := 1; i < len(trend); i++ {
			if trend[i].Timestamp.Before(trend[i-1].Timestamp) {
				t.Error("Expected trend points to be sorted by timestamp")
			}
		}

		expectedScores := []int{20, 40, 60, 80}
		for i, point := range trend {
			if point.HeatScore != expectedScores[i] {
				t.Errorf("Expected heat score %d at index %d, got %d", expectedScores[i], i, point.HeatScore)
			}
		}
	})

	t.Run("time_series_data", func(t *testing.T) {
		baseTime := now.Add(-24 * time.Hour)
		for i := 0; i < 24; i++ {
			heat := ComponentHeat{
				Timestamp:    baseTime.Add(time.Duration(i) * time.Hour),
				Component:    "series-component-unique",
				FailureCount: int64(i),
				HeatScore:    i * 4, // 0, 4, 8, ..., 92
			}
			if err := store.SaveHeatData("series-component-unique", heat); err != nil {
				t.Fatalf("Failed to save heat data: %v", err)
			}
		}

		trend, err := store.GetHeatTrend("series-component-unique", 25*time.Hour)
		if err != nil {
			t.Fatalf("Failed to get heat trend: %v", err)
		}

		if len(trend) != 24 {
			t.Errorf("Expected 24 trend points, got %d", len(trend))
		}

		for i, point := range trend {
			expectedScore := i * 4
			if point.HeatScore != expectedScore {
				t.Errorf("Expected heat score %d at index %d, got %d", expectedScore, i, point.HeatScore)
			}
		}
	})
}

// TestSaveAndGetHealthState tests saving and retrieving health states
func TestSaveAndGetHealthState(t *testing.T) {
	store := setupTestStore(t)
	now := time.Now()

	t.Run("save_health_state", func(t *testing.T) {
		comp := &InternalComponent{
			Name:         "health-component",
			Health:       HealthHealthy,
			ErrorCount:   0,
			SuccessCount: 100,
			TotalOps:     100,
		}
		comp.LastSuccess = now

		err := store.SaveHealthState(comp)
		if err != nil {
			t.Fatalf("Failed to save health state: %v", err)
		}
	})

	t.Run("get_health_history", func(t *testing.T) {
		comps := []*InternalComponent{
			{Name: "history-health", Health: HealthHealthy, ErrorCount: 0, SuccessCount: 100, TotalOps: 100},
			{Name: "history-health", Health: HealthDegraded, ErrorCount: 10, SuccessCount: 90, TotalOps: 100},
			{Name: "history-health", Health: HealthFailing, ErrorCount: 50, SuccessCount: 50, TotalOps: 100},
		}

		for _, comp := range comps {
			if err := store.SaveHealthState(comp); err != nil {
				t.Fatalf("Failed to save health state: %v", err)
			}
		}

		history, err := store.GetHealthHistory("history-health", now.Add(-time.Hour))
		if err != nil {
			t.Fatalf("Failed to get health history: %v", err)
		}

		if len(history) != 3 {
			t.Errorf("Expected 3 health history entries, got %d", len(history))
		}

		for _, h := range history {
			if h.Component != "history-health" {
				t.Errorf("Expected component 'history-health', got '%s'", h.Component)
			}
		}
	})

	t.Run("get_health_trend", func(t *testing.T) {
		comps := []*InternalComponent{
			{Name: "trend-health", Health: HealthHealthy, ErrorCount: 0, SuccessCount: 100, TotalOps: 100},
			{Name: "trend-health", Health: HealthDegraded, ErrorCount: 20, SuccessCount: 80, TotalOps: 100},
			{Name: "trend-health", Health: HealthFailing, ErrorCount: 60, SuccessCount: 40, TotalOps: 100},
		}

		for _, comp := range comps {
			if err := store.SaveHealthState(comp); err != nil {
				t.Fatalf("Failed to save health state: %v", err)
			}
		}

		trend, err := store.GetHealthTrend("trend-health", time.Hour)
		if err != nil {
			t.Fatalf("Failed to get health trend: %v", err)
		}

		if trend.Component != "trend-health" {
			t.Errorf("Expected component 'trend-health', got '%s'", trend.Component)
		}

		if len(trend.Points) != 3 {
			t.Errorf("Expected 3 trend points, got %d", len(trend.Points))
		}

		for i := 1; i < len(trend.Points); i++ {
			if trend.Points[i].Timestamp.Before(trend.Points[i-1].Timestamp) {
				t.Error("Expected trend points to be sorted by timestamp")
			}
		}
	})
}

// TestSaveAndGetLLMAudit tests saving and retrieving LLM audit entries
func TestSaveAndGetLLMAudit(t *testing.T) {
	store := setupTestStore(t)
	now := time.Now()

	t.Run("save_llm_audit", func(t *testing.T) {
		entry := LLMAuditEntry{
			Timestamp:     now,
			OperationType: OpClassify,
			InputHash:     "abc123hash",
			ResponseHash:  "def456hash",
			Success:       true,
			LatencyMs:     150,
			ErrorMessage:  "",
			ProviderType:  "test",
		}

		err := store.SaveLLMAudit(entry)
		if err != nil {
			t.Fatalf("Failed to save LLM audit: %v", err)
		}
	})

	t.Run("get_audit_log", func(t *testing.T) {
		entries := []LLMAuditEntry{
			{Timestamp: now.Add(-30 * time.Minute), OperationType: OpExplain, InputHash: "hash2", ResponseHash: "resp2", Success: false, LatencyMs: 500, ErrorMessage: "timeout", ProviderType: "test"},
			{Timestamp: now.Add(-20 * time.Minute), OperationType: OpSuggest, InputHash: "hash3", ResponseHash: "resp3", Success: true, LatencyMs: 200, ProviderType: "test"},
		}

		for _, entry := range entries {
			if err := store.SaveLLMAudit(entry); err != nil {
				t.Fatalf("Failed to save LLM audit: %v", err)
			}
		}

		auditLog, err := store.GetLLMAudit(now.Add(-time.Hour))
		if err != nil {
			t.Fatalf("Failed to get LLM audit: %v", err)
		}

		// Should have at least the 2 we just added
		if len(auditLog) < 2 {
			t.Errorf("Expected at least 2 audit entries, got %d", len(auditLog))
		}

		foundExplain := false
		for _, entry := range auditLog {
			if entry.OperationType == OpExplain && entry.ErrorMessage == "timeout" {
				foundExplain = true
				if entry.LatencyMs != 500 {
					t.Errorf("Expected latency 500ms, got %d", entry.LatencyMs)
				}
			}
		}
		if !foundExplain {
			t.Error("Expected to find explain operation with timeout error")
		}
	})

	t.Run("get_llm_stats", func(t *testing.T) {
		entries := []LLMAuditEntry{
			{Timestamp: now, OperationType: OpClassify, InputHash: "h1", Success: true, LatencyMs: 100, ProviderType: "test"},
			{Timestamp: now, OperationType: OpClassify, InputHash: "h2", Success: true, LatencyMs: 200, ProviderType: "test"},
			{Timestamp: now, OperationType: OpExplain, InputHash: "h3", Success: false, LatencyMs: 5000, ProviderType: "test"},
			{Timestamp: now, OperationType: OpSuggest, InputHash: "h4", Success: true, LatencyMs: 300, ProviderType: "test"},
		}

		for _, entry := range entries {
			if err := store.SaveLLMAudit(entry); err != nil {
				t.Fatalf("Failed to save LLM audit: %v", err)
			}
		}

		stats, err := store.GetLLMStats(time.Hour)
		if err != nil {
			t.Fatalf("Failed to get LLM stats: %v", err)
		}

		// Stats should include at least the 4 we just added
		if stats.TotalCalls < 4 {
			t.Errorf("Expected at least 4 total calls, got %d", stats.TotalCalls)
		}

		// Just verify success rate is reasonable (between 0 and 1)
		if stats.SuccessRate < 0 || stats.SuccessRate > 1 {
			t.Errorf("Expected success rate between 0 and 1, got %.2f", stats.SuccessRate)
		}
	})
}

// TestPersistencePruneOldData tests data pruning functionality
func TestPersistencePruneOldData(t *testing.T) {
	store := setupTestStore(t)
	now := time.Now()

	t.Run("prune_audit_trail", func(t *testing.T) {
		oldAction := HealingAction{
			Timestamp:  now.Add(-48 * time.Hour),
			Component:  "prune-audit",
			ActionType: ActionRetry,
			Success:    true,
		}
		newAction := HealingAction{
			Timestamp:  now,
			Component:  "prune-audit",
			ActionType: ActionRetry,
			Success:    true,
		}

		if err := store.SaveHealingAction(oldAction); err != nil {
			t.Fatalf("Failed to save old action: %v", err)
		}
		if err := store.SaveHealingAction(newAction); err != nil {
			t.Fatalf("Failed to save new action: %v", err)
		}

		err := store.PruneAuditTrail(now.Add(-24 * time.Hour))
		if err != nil {
			t.Fatalf("Failed to prune audit trail: %v", err)
		}

		actions, err := store.GetHealingActions("prune-audit", now.Add(-72*time.Hour))
		if err != nil {
			t.Fatalf("Failed to get healing actions: %v", err)
		}

		if len(actions) != 1 {
			t.Errorf("Expected 1 action after pruning, got %d", len(actions))
		}

		if len(actions) > 0 && actions[0].Timestamp.Before(now.Add(-24*time.Hour)) {
			t.Error("Expected only recent action to remain")
		}
	})

	t.Run("prune_heat_data", func(t *testing.T) {
		oldHeat := ComponentHeat{
			Timestamp:    now.Add(-48 * time.Hour),
			Component:    "prune-heat",
			FailureCount: int64(5),
			HeatScore:    50,
		}
		newHeat := ComponentHeat{
			Timestamp:    now,
			Component:    "prune-heat",
			FailureCount: int64(10),
			HeatScore:    75,
		}

		if err := store.SaveHeatData("prune-heat", oldHeat); err != nil {
			t.Fatalf("Failed to save old heat: %v", err)
		}
		if err := store.SaveHeatData("prune-heat", newHeat); err != nil {
			t.Fatalf("Failed to save new heat: %v", err)
		}

		err := store.PruneHeatData(now.Add(-24 * time.Hour))
		if err != nil {
			t.Fatalf("Failed to prune heat data: %v", err)
		}

		heats, err := store.GetHeatData("prune-heat", now.Add(-72*time.Hour))
		if err != nil {
			t.Fatalf("Failed to get heat data: %v", err)
		}

		if len(heats) != 1 {
			t.Errorf("Expected 1 heat record after pruning, got %d", len(heats))
		}
	})

	t.Run("prune_health_history", func(t *testing.T) {
		oldComp := &InternalComponent{
			Name:   "prune-health",
			Health: HealthHealthy,
		}
		newComp := &InternalComponent{
			Name:   "prune-health",
			Health: HealthDegraded,
		}

		store.mu.Lock()
		_, err := store.db.Exec(
			"INSERT INTO health_history (timestamp, component, health_state) VALUES (?, ?, ?)",
			now.Add(-48*time.Hour), oldComp.Name, string(oldComp.Health),
		)
		store.mu.Unlock()
		if err != nil {
			t.Fatalf("Failed to insert old health state: %v", err)
		}

		if err := store.SaveHealthState(newComp); err != nil {
			t.Fatalf("Failed to save new health state: %v", err)
		}

		err = store.PruneHealthHistory(now.Add(-24 * time.Hour))
		if err != nil {
			t.Fatalf("Failed to prune health history: %v", err)
		}

		history, err := store.GetHealthHistory("prune-health", now.Add(-72*time.Hour))
		if err != nil {
			t.Fatalf("Failed to get health history: %v", err)
		}

		if len(history) != 1 {
			t.Errorf("Expected 1 health record after pruning, got %d", len(history))
		}
	})

	t.Run("prune_llm_audit", func(t *testing.T) {
		oldEntry := LLMAuditEntry{
			Timestamp:     now.Add(-48 * time.Hour),
			OperationType: OpClassify,
			InputHash:     "old-hash",
			Success:       true,
			ProviderType:  "test",
		}
		newEntry := LLMAuditEntry{
			Timestamp:     now,
			OperationType: OpExplain,
			InputHash:     "new-hash",
			Success:       true,
			ProviderType:  "test",
		}

		if err := store.SaveLLMAudit(oldEntry); err != nil {
			t.Fatalf("Failed to save old audit entry: %v", err)
		}
		if err := store.SaveLLMAudit(newEntry); err != nil {
			t.Fatalf("Failed to save new audit entry: %v", err)
		}

		err := store.PruneLLMAudit(now.Add(-24 * time.Hour))
		if err != nil {
			t.Fatalf("Failed to prune LLM audit: %v", err)
		}

		auditLog, err := store.GetLLMAudit(now.Add(-72 * time.Hour))
		if err != nil {
			t.Fatalf("Failed to get LLM audit: %v", err)
		}

		if len(auditLog) != 1 {
			t.Errorf("Expected 1 audit entry after pruning, got %d", len(auditLog))
		}
	})

	t.Run("prune_all_old_data", func(t *testing.T) {
		// Use unique component names to avoid conflicts
		action := HealingAction{
			Timestamp:  now.Add(-48 * time.Hour),
			Component:  "prune-all-test",
			ActionType: ActionRetry,
			Success:    true,
		}
		if err := store.SaveHealingAction(action); err != nil {
			t.Fatalf("Failed to save action: %v", err)
		}

		heat := ComponentHeat{
			Timestamp:    now.Add(-48 * time.Hour),
			Component:    "prune-all-test",
			FailureCount: int64(5),
			HeatScore:    50,
		}
		if err := store.SaveHeatData("prune-all-test", heat); err != nil {
			t.Fatalf("Failed to save heat: %v", err)
		}

		entry := LLMAuditEntry{
			Timestamp:     now.Add(-48 * time.Hour),
			OperationType: OpClassify,
			InputHash:     "all-hash-test",
			Success:       true,
			ProviderType:  "test",
		}
		if err := store.SaveLLMAudit(entry); err != nil {
			t.Fatalf("Failed to save audit: %v", err)
		}

		err := store.PruneOldData(now.Add(-24 * time.Hour))
		if err != nil {
			t.Fatalf("Failed to prune old data: %v", err)
		}

		actions, _ := store.GetHealingActions("prune-all-test", now.Add(-72*time.Hour))
		heats, _ := store.GetHeatData("prune-all-test", now.Add(-72*time.Hour))
		auditLog, _ := store.GetLLMAudit(now.Add(-72 * time.Hour))

		// Filter audit log for our specific entry
		var foundAudit int
		for _, entry := range auditLog {
			if entry.InputHash == "all-hash-test" {
				foundAudit++
			}
		}

		if len(actions) != 0 {
			t.Errorf("Expected 0 actions after pruning, got %d", len(actions))
		}
		if len(heats) != 0 {
			t.Errorf("Expected 0 heat records after pruning, got %d", len(heats))
		}
		if foundAudit != 0 {
			t.Errorf("Expected 0 matching audit entries after pruning, got %d", foundAudit)
		}
	})
}

// TestRetentionPolicy tests automatic retention policy behavior
func TestRetentionPolicy(t *testing.T) {
	t.Run("retention_policy_set", func(t *testing.T) {
		store, err := NewPersistenceStore("file:memdb_test2?mode=memory&cache=shared", 30*24*time.Hour)
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.Close()

		if store.retentionPolicy != 30*24*time.Hour {
			t.Errorf("Expected retention policy 720h, got %v", store.retentionPolicy)
		}
	})

	t.Run("edge_cases", func(t *testing.T) {
		store := setupTestStore(t)
		now := time.Now()

		err := store.PruneOldData(now.Add(24 * time.Hour))
		if err != nil {
			t.Fatalf("Failed to prune with future date: %v", err)
		}

		err = store.PruneOldData(now.Add(-100 * 365 * 24 * time.Hour))
		if err != nil {
			t.Fatalf("Failed to prune with very old date: %v", err)
		}

		heat := ComponentHeat{
			Timestamp:    now,
			Component:    "edge-case",
			FailureCount: int64(0),
			HeatScore:    0,
		}
		if err := store.SaveHeatData("edge-case", heat); err != nil {
			t.Fatalf("Failed to save heat with zero score: %v", err)
		}

		action := HealingAction{
			Timestamp:    now,
			Component:    "edge-case",
			ActionType:   ActionRetry,
			Success:      false,
			ErrorMessage: "",
		}
		if err := store.SaveHealingAction(action); err != nil {
			t.Fatalf("Failed to save action with empty error message: %v", err)
		}
	})
}

// TestConcurrency tests concurrent access to the persistence store
func TestConcurrency(t *testing.T) {
	t.Run("concurrent_writes", func(t *testing.T) {
		store := setupTestStore(t)
		now := time.Now()
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(idx int) {
				defer func() { done <- true }()

				action := HealingAction{
					Timestamp:  now,
					Component:  "concurrent-component",
					ActionType: ActionRetry,
					Success:    idx%2 == 0,
					RetryCount: idx,
				}

				if err := store.SaveHealingAction(action); err != nil {
					t.Errorf("Failed to save healing action: %v", err)
				}

				heat := ComponentHeat{
					Timestamp:    now,
					Component:    "concurrent-component",
					FailureCount: int64(idx),
					HeatScore:    idx * 10,
				}

				if err := store.SaveHeatData("concurrent-component", heat); err != nil {
					t.Errorf("Failed to save heat data: %v", err)
				}
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		actions, err := store.GetHealingActions("concurrent-component", now.Add(-time.Hour))
		if err != nil {
			t.Fatalf("Failed to get healing actions: %v", err)
		}

		if len(actions) != 10 {
			t.Errorf("Expected 10 actions, got %d", len(actions))
		}
	})

	t.Run("concurrent_reads", func(t *testing.T) {
		// Create a fresh store for this subtest
		store := setupTestStore(t)
		now := time.Now()

		for i := 0; i < 5; i++ {
			action := HealingAction{
				Timestamp:  now,
				Component:  "read-component",
				ActionType: ActionRetry,
				Success:    true,
			}
			if err := store.SaveHealingAction(action); err != nil {
				t.Fatalf("Failed to save healing action: %v", err)
			}
		}

		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				actions, err := store.GetHealingActions("read-component", now.Add(-time.Hour))
				if err != nil {
					t.Errorf("Failed to get healing actions: %v", err)
					return
				}

				if len(actions) != 5 {
					t.Errorf("Expected 5 actions, got %d", len(actions))
				}

				_, err = store.GetHealingActionStats("read-component", time.Hour)
				if err != nil {
					t.Errorf("Failed to get stats: %v", err)
				}
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent_read_write", func(t *testing.T) {
		store := setupTestStore(t)
		now := time.Now()
		done := make(chan bool, 20)

		for i := 0; i < 10; i++ {
			go func(idx int) {
				defer func() { done <- true }()

				action := HealingAction{
					Timestamp:  now,
					Component:  "rw-component",
					ActionType: ActionRetry,
					Success:    true,
				}
				store.SaveHealingAction(action)
			}(i)
		}

		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				store.GetHealingActions("rw-component", now.Add(-time.Hour))
				store.GetAllHealingActions(now.Add(-time.Hour))
			}()
		}

		for i := 0; i < 20; i++ {
			<-done
		}
	})
}

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/selfobs"
)

// healthInternalCmd shows internal component health
func healthInternalCmd(args []string) {
	cfg, _ := loadCfg(args)
	if cfg.Bind.API == "" {
		fmt.Println("Error: API server not configured")
		os.Exit(1)
	}
	
	resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/health/internal", cfg.Bind.API))
	if err != nil {
		fmt.Printf("Error connecting to API: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("Error decoding response: %v\n",", err)
		os.Exit(1)
	}
	
	fmt.Printf("Overall Health: %s\n\n", data["overall_health"])
	fmt.Println("Components:")
	components := data["components"].([]any)
	for _, c := range components {
		comp := c.(map[string]any)
		health := comp["health"].(string)
		icon := "✓"
		switch health {
		case "healthy":
			icon = "✓"
		case "degraded":
			icon = "⚠"
		case "failing":
			icon = "✗"
		case "unknown":
			icon = "?"
		}
		fmt.Printf("  %s %s - Error Rate: %.1f%%\n", icon, comp["name"], comp["error_rate"])
	}
}

// healthFreshnessCmd shows freshness status
func healthFreshnessCmd(args []string) {
	cfg, _ := loadCfg(args)
	if cfg.Bind.API == "" {
		fmt.Println("Error: API server not configured")
		os.Exit(1)
	}
	
	resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/health/freshness", cfg.Bind.API))
	if err != nil {
		fmt.Printf("Error connecting to API: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		os.Exit(1)
	}
	
	stale := data["stale_components"].([]any)
	if len(stale) > 0 {
		fmt.Printf("⚠ STALE COMPONENTS: %s\n\n", strings.Join(stale, ", "))
	} else {
		fmt.Println("✓ All components fresh\n")
	}
	
	fmt.Println("Component Freshness:")
	markers := data["markers"].([]any)
	for _, m := range markers {
		marker := m.(map[string]any)
		status := "✓"
		if marker["is_stale"].(bool) {
			status = "✗"
		}
		age := marker["age_seconds"].(float64)
		fmt.Printf("  %s %s - %.1fs ago (threshold: %.0fs)\n", status, marker["component"], age, marker["stale_threshold"])
	}
}

// healthSLOCmd shows SLO status
func healthSLOCmd(args []string) {
	cfg, _ := loadCfg(args)
	if cfg.Bind.API == "" {
		fmt.Println("Error: API server not configured")
		os.Exit(1)
	}
	
	resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/health/slo", cfg.Bind.API))
	if err != nil {
		fmt.Printf("Error connecting to API: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("SLO Status:")
	slos := data["slos"].([]any)
	for _, s := range slos {
		slo := s.(map[string]any)
		status := slo["status"].(string)
		icon := "✓"
		switch status {
		case "healthy":
			icon = "✓"
		case "at_risk":
			icon = "⚠"
		case "breached":
			icon = "✗"
		default:
			icon = "?"
		}
		fmt.Printf("  %s %s: %.1f%% (target: %.1f%%) [%s]\n", icon, slo["name"], slo["current_value"], slo["target"], slo["status"])
	}
}

// healthMetricsCmd shows internal metrics
func healthMetricsCmd(args []string) {
	cfg, _ := loadCfg(args)
	if cfg.Bind.API == "" {
		fmt.Println("Error: API server not configured")
		os.Exit(1)
	}
	
	resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/metrics/internal", cfg.Bind.API))
	if err != nil {
		fmt.Printf("Error connecting to API: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Pipeline Latency (P99):")
	latency := data["pipeline_latency"].(map[string]any)
	fmt.Printf("  Ingest → Classify: %dms\n", int(latency["ingest_to_classify_p99"].(float64)))
	fmt.Printf("  Classify → Alert: %dms\n", int(latency["classify_to_alert_p99"].(float64)))
	fmt.Printf("  Alert → Action: %dms\n", int(latency["alert_to_action_p99"].(float64)))
	
	fmt.Println("\nQueue Depths:")
	queues := data["queue_depths"].(map[string]any)
	for q, d := range queues {
		fmt.Printf("  %s: %d\n", q, int(d.(float64)))
	}
	
	fmt.Println("\nError Rates:")
	errors := data["error_rates"].(map[string]any)
	for c, r := range errors {
		fmt.Printf("  %s: %.2f%%\n", c, r.(float64))
	}
	
	fmt.Println("\nResource Usage:")
	resources := data["resource_usage"].(map[string]any)
	fmt.Printf("  Memory: %d bytes\n", int(resources["memory_used_bytes"].(float64)))
	fmt.Printf("  Goroutines: %d\n", int(resources["goroutines"].(float64)))
}

// selfobs package-level functions for local CLI mode (when API not available)

// printLocalHealth prints health status using local selfobs package
func printLocalHealth() {
	registry := selfobs.GetGlobalRegistry()
	components := registry.GetAllComponents()
	
	fmt.Printf("Overall Health: %s\n\n", registry.GetOverallHealth())
	fmt.Println("Components:")
	for _, comp := range components {
		icon := "✓"
		switch string(comp.Health) {
		case "healthy":
			icon = "✓"
		case "degraded":
			icon = "⚠"
		case "failing":
			icon = "✗"
		case "unknown":
			icon = "?"
		}
		fmt.Printf("  %s %s - Error Rate: %.1f%%\n", icon, comp.Name, comp.ErrorRate())
	}
}

// printLocalFreshness prints freshness using local selfobs package
func printLocalFreshness() {
	tracker := selfobs.GetGlobalFreshnessTracker()
	markers := tracker.GetAllMarkers()
	stale := tracker.GetStaleComponents()
	
	if len(stale) > 0 {
		fmt.Println("⚠ STALE COMPONENTS:")
		for _, m := range stale {
			fmt.Printf("  ✗ %s - %s\n", m.Component, time.Since(m.LastUpdate).Round(time.Second))
		}
	} else {
		fmt.Println("✓ All components fresh")
	}
	
	fmt.Println("\nComponent Freshness:")
	for _, marker := range markers {
		status := "✓"
		if marker.IsStale() {
			status = "✗"
		}
		age := time.Since(marker.LastUpdate).Round(time.Second)
		fmt.Printf("  %s %s - %s ago (threshold: %s)\n", status, marker.Component, age, marker.StaleThreshold)
	}
}

// printLocalSLO prints SLO status using local selfobs package
func printLocalSLO() {
	tracker := selfobs.GetGlobalSLOTracker()
	statuses := tracker.GetAllSLOStatuses()
	
	fmt.Println("SLO Status:")
	for _, status := range statuses {
		icon := "✓"
		switch status.Status {
		case "healthy":
			icon = "✓"
		case "at_risk":
			icon = "⚠"
		case "breached":
			icon = "✗"
		default:
			icon = "?"
		}
		def := tracker.GetSLODefinition(status.Name)
		unit := "%"
		if def != nil {
			unit = def.Unit
		}
		fmt.Printf("  %s %s: %.1f%s (target: %.1f%s) [%s]\n", icon, status.Name, status.CurrentValue, unit, status.Target, unit, status.Status)
	}
}

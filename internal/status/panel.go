package status

import (
	"strings"

	"github.com/mel-project/mel/internal/transport"
)

type Panel struct {
	GeneratedAt   string        `json:"generated_at"`
	OperatorState string        `json:"operator_state"`
	Summary       string        `json:"summary"`
	ShortCommands []string      `json:"short_commands"`
	WebHints      []string      `json:"web_hints"`
	OperatorMenu  []OperatorAction `json:"operator_menu"`
	Transports    []PanelMetric    `json:"transports"`
}

type PanelMetric struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	State      string `json:"state"`
	Messages   uint64 `json:"messages"`
	LastIngest string `json:"last_ingest,omitempty"`
	Detail     string `json:"detail"`
	Score      *int   `json:"score,omitempty"`
}

type OperatorAction struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Action string `json:"action"`
}

func BuildPanel(snap Snapshot) Panel {
	panel := Panel{
		GeneratedAt:   snap.GeneratedAt,
		OperatorState: operatorState(snap),
		Summary:       panelSummary(snap),
		ShortCommands: []string{"S=Status", "T=Transports", "N=Nodes", "R=Replay", "D=Doctor"},
		WebHints: []string{
			"Open /api/v1/status for full transport truth.",
			"Open /api/v1/panel for compact operator state.",
			"Use the Web UI to verify live ingest versus historical-only evidence.",
		},
		OperatorMenu: []OperatorAction{
			{Key: "A", Label: "State", Action: "Show operator state and overall ingest truth"},
			{Key: "B", Label: "Link", Action: "Cycle transport states and last errors"},
			{Key: "C", Label: "Msgs", Action: "Show persisted and runtime message counters"},
			{Key: "D", Label: "Retry", Action: "Show reconnect attempts and offline guidance"},
		},
		Transports: make([]PanelMetric, 0, len(snap.Transports)),
	}
	for _, tr := range snap.Transports {
		label := strings.ToUpper(firstRune(tr.Type))
		if label == "" {
			label = "?"
		}
		metric := PanelMetric{
			Name:       tr.Name,
			Label:      label,
			State:      tr.EffectiveState,
			Messages:   maxUint64(tr.TotalMessages, tr.PersistedMessages),
			LastIngest: tr.LastIngestAt,
			Detail:     tr.Detail,
		}
		if tr.Health.LastEvaluatedAt != "" {
			metric.Score = &tr.Health.Score
		}
		panel.Transports = append(panel.Transports, metric)
	}
	return panel
}

func operatorState(snap Snapshot) string {
	if len(snap.Transports) == 0 {
		return "idle"
	}
	hasLive := false
	hasDegraded := false
	for _, tr := range snap.Transports {
		switch tr.EffectiveState {
		case transport.StateIngesting:
			hasLive = true
		case transport.StateAttempting, transport.StateConfiguredOffline, transport.StateConnectedNoIngest, transport.StateHistoricalOnly, transport.StateError:
			hasDegraded = true
		}
	}
	if hasLive && !hasDegraded {
		return "ready"
	}
	if hasLive || hasDegraded {
		return "degraded"
	}
	return "idle"
}

func panelSummary(snap Snapshot) string {
	if len(snap.Transports) == 0 {
		return "No transports configured; MEL is explicitly idle."
	}
	for _, tr := range snap.Transports {
		if tr.EffectiveState == transport.StateIngesting {
			return "Live ingest is confirmed by SQLite writes."
		}
	}
	for _, tr := range snap.Transports {
		if tr.EffectiveState == transport.StateHistoricalOnly {
			return "Historical evidence exists, but current live ingest is not proven."
		}
	}
	return "Configured transports are present, but no live ingest has been proven yet."
}

func firstRune(v string) string {
	if v == "" {
		return ""
	}
	return string([]rune(v)[0])
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

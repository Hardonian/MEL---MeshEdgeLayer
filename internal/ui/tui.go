package ui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/status"
)

// Model is the Bubble Tea model for the operator TUI.
type Model struct {
	cfg        config.Config
	db         *db.DB
	snap       status.Snapshot
	err        error
	width      int
	height     int
	activeTab  int
	lastUpdate time.Time
	polling    bool
	stats      runtime.MemStats
	confirming bool
}

const (
	tabOverview = iota
	tabMesh
	tabAlerts
	tabControl
	tabLogs
	tabDiagnostics
)

var tabNames = []string{"OVERVIEW", "MESH", "ALERTS", "CONTROL", "LOGS", "DIAGS"}

func New(cfg config.Config, d *db.DB) *Model {
	return &Model{
		cfg:     cfg,
		db:      d,
		polling: true,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.refresh(), m.tick())
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				return m, m.vacuum()
			case "n", "N", "esc":
				m.confirming = false
				return m, nil
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right":
			m.activeTab = (m.activeTab + 1) % len(tabNames)
		case "left":
			m.activeTab = (m.activeTab - 1 + len(tabNames)) % len(tabNames)
		case "r":
			return m, m.refresh()
		case "1", "2", "3", "4", "5", "6":
			m.activeTab = int(msg.Runes[0] - '1')
		case "p":
			m.polling = !m.polling
		case "v":
			if m.activeTab == tabDiagnostics {
				m.confirming = true
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		if m.polling {
			return m, tea.Batch(m.refresh(), m.tick(), m.statTick())
		}
		return m, tea.Batch(m.tick(), m.statTick())
	case snapMsg:
		m.snap = status.Snapshot(msg)
		m.lastUpdate = time.Now()
		m.err = nil
	case statMsg:
		m.stats = runtime.MemStats(msg)
	case errMsg:
		m.err = error(msg)
	}
	return m, nil
}

type tickMsg struct{}
type statMsg runtime.MemStats
type snapMsg status.Snapshot
type errMsg error

func (m *Model) tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *Model) statTick() tea.Cmd {
	return func() tea.Msg {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		return statMsg(stats)
	}
}

func (m *Model) vacuum() tea.Cmd {
	return func() tea.Msg {
		if m.db == nil { return nil }
		if err := m.db.Vacuum(); err != nil {
			return errMsg(err)
		}
		return m.refresh()()
	}
}

func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg {
		snap, err := status.Collect(m.cfg, m.db, nil)
		if err != nil {
			return errMsg(err)
		}
		return snapMsg(snap)
	}
}

func (m *Model) View() string {
	if m.width < 60 || m.height < 15 {
		return "Terminal too small. Please enlarge to at least 60x15."
	}

	if m.confirming {
		return m.renderConfirmation()
	}

	header := m.renderHeader()
	tabs := m.renderTabs()
	content := m.renderContent()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, footer)
}

func (m *Model) renderConfirmation() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#EBCB8B")).
		Padding(1, 4).
		Width(50).
		Align(lipgloss.Center)

	text := "VACUUM DATABASE?\n\nThis will re-index and shrink the SQLite file.\nThis is safe but may briefly lock the DB.\n\n[Y] Confirm   [N] Cancel"
	
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, style.Render(text))
}

func (m *Model) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#2E3440")).
		Padding(0, 1)

	statusText := "READY"
	statusColor := "#A3BE8C"
	if m.err != nil {
		statusText = "DB ERROR"
		statusColor = "#BF616A"
	} else if len(m.snap.ActiveTransportAlerts) > 0 {
		statusText = "ALERTS ACTIVE"
		statusColor = "#EBCB8B"
	}

	statusStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(statusColor)).
		Padding(0, 1)

	pollingText := "LIVE"
	if !m.polling {
		pollingText = "PAUSED"
	}

	headerText := titleStyle.Render("MEL OPERATOR v0.1") + " " + statusStyle.Render(statusText)
	if !m.lastUpdate.IsZero() && time.Since(m.lastUpdate) > 5*time.Second {
		headerText += " " + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#BF616A")).Render("STALE")
	}
	rightText := lipgloss.NewStyle().Foreground(lipgloss.Color("#4C566A")).Render(fmt.Sprintf("POLLING: %s", pollingText))

	return lipgloss.JoinHorizontal(lipgloss.Center, headerText, lipgloss.PlaceHorizontal(m.width-lipgloss.Width(headerText)-lipgloss.Width(rightText), lipgloss.Right, ""), rightText)
}

func (m *Model) renderTabs() string {
	var tabs []string
	for i, name := range tabNames {
		style := lipgloss.NewStyle().Padding(0, 1)
		if i == m.activeTab {
			style = style.Bold(true).Foreground(lipgloss.Color("#88C0D0")).Underline(true)
		} else {
			style = style.Foreground(lipgloss.Color("#4C566A"))
		}
		tabs = append(tabs, style.Render(fmt.Sprintf("%d %s", i+1, name)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m *Model) renderFooter() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4C566A")).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), true, false, false, false)

	msgCount := m.snap.Messages
	lastIngest := m.snap.LastSuccessfulIngest
	if lastIngest == "" {
		lastIngest = "none"
	}

	helpText := "TAB/←→ Nav • R Ref • P Pause • Q Quit"
	if m.activeTab == tabDiagnostics {
		helpText += " • V Vacuum"
	}
	statsText := fmt.Sprintf("MSGS: %d | LAST: %s", msgCount, lastIngest)

	return style.Width(m.width).Render(fmt.Sprintf("%s %s", helpText, lipgloss.PlaceHorizontal(m.width-lipgloss.Width(helpText)-lipgloss.Width(statsText)-2, lipgloss.Right, statsText)))
}

func (m *Model) renderContent() string {
	contentArea := lipgloss.NewStyle().
		Padding(1).
		Height(m.height - 6).
		Width(m.width - 2)

	if m.err != nil {
		return contentArea.Render(fmt.Sprintf("SYSTEM ERROR:\n\n%v", m.err))
	}

	switch m.activeTab {
	case tabOverview:
		return contentArea.Render(m.viewOverview())
	case tabMesh:
		return contentArea.Render(m.viewMesh())
	case tabAlerts:
		return contentArea.Render(m.viewAlerts())
	case tabControl:
		return contentArea.Render(m.viewControl())
	case tabLogs:
		return contentArea.Render(m.viewLogs())
	case tabDiagnostics:
		return contentArea.Render(m.viewDiagnostics())
	default:
		return contentArea.Render("View not implemented.")
	}
}

func (m *Model) viewOverview() string {
	panel := status.BuildPanel(m.snap)
	
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EBCB8B")).Render("SUMMARY") + "\n")
	sb.WriteString(panel.Summary + "\n\n")
	
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0")).Render("ACTIVE TRANSPORTS") + "\n")
	if len(m.snap.Transports) == 0 {
		sb.WriteString("No transports enabled.\n")
	}
	for _, tr := range m.snap.Transports {
		stateStyle := lipgloss.NewStyle()
		if tr.EffectiveState == "ingesting" || tr.EffectiveState == "live" {
			stateStyle = stateStyle.Foreground(lipgloss.Color("#A3BE8C"))
		} else if tr.EffectiveState == "error" || tr.EffectiveState == "failed" {
			stateStyle = stateStyle.Foreground(lipgloss.Color("#BF616A"))
		}
		
		sb.WriteString(fmt.Sprintf("%-16s %s %s msgs=%d", tr.Name, lipgloss.NewStyle().Foreground(lipgloss.Color("#4C566A")).Render("("+tr.Type+")"), stateStyle.Render(strings.ToUpper(tr.EffectiveState)), tr.PersistedMessages))
		if tr.LastIngestAt != "" {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#4C566A")).Render(" last="+tr.LastIngestAt))
		}
		sb.WriteString("\n")
		if tr.LastError != "" {
			sb.WriteString("    " + lipgloss.NewStyle().Foreground(lipgloss.Color("#D08770")).Render("ERROR: "+tr.LastError) + "\n")
		}
	}
	
	return sb.String()
}

func (m *Model) viewMesh() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0")).Render("MESH TOPOLOGY") + "\n")
	sb.WriteString(fmt.Sprintf("Observed Nodes: %d\n", m.snap.Nodes))
	sb.WriteString(fmt.Sprintf("Schema Version: %s\n\n", m.snap.SchemaVersion))

	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0")).Render("TRANSPORT CAPACITY") + "\n")
	for _, tr := range m.snap.Transports {
		sb.WriteString(fmt.Sprintf("%-16s ", tr.Name))
		caps := []string{}
		if tr.Capabilities.IngestSupported { caps = append(caps, "INGEST") }
		if tr.Capabilities.SendSupported { caps = append(caps, "SEND") }
		if tr.Capabilities.NodeFetchSupported { caps = append(caps, "INVENTORY") }
		if tr.Capabilities.ConfigApplySupported { caps = append(caps, "CONFIG") }
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#4C566A")).Render(strings.Join(caps, " | ")) + "\n")
	}

	return sb.String()
}

func (m *Model) viewAlerts() string {
	if len(m.snap.ActiveTransportAlerts) == 0 {
		return "No active transport alerts."
	}
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#BF616A")).Render("ACTIVE ALERTS") + "\n\n")
	for _, alert := range m.snap.ActiveTransportAlerts {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render(alert.TransportName) + " — " + alert.Reason + "\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")).Render(alert.Summary) + "\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#4C566A")).Render(fmt.Sprintf("First triggered: %s", alert.FirstTriggeredAt)) + "\n\n")
	}
	return sb.String()
}

func (m *Model) viewControl() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0")).Render("CONTROL POSTURE") + "\n")
	sb.WriteString("Mode: OBSERVATION (Default)\n\n")
	sb.WriteString("No active guarded actions in the current episode.\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("Guardrail logic is passive until explicitly enabled in config.control.mode."))
    return sb.String()
}

func (m *Model) viewLogs() string {
	var sb strings.Builder
	if len(m.snap.RecentTransportIncidents) == 0 {
		return "No recent transport incidents recorded."
	}
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0")).Render("RECENT INCIDENTS") + "\n\n")
	for _, inc := range m.snap.RecentTransportIncidents {
		icon := " "
		if inc.DeadLetter { icon = "!" }
		sb.WriteString(fmt.Sprintf("[%s] %-16s %s (%d occurrences)\n", icon, inc.TransportName, inc.Reason, inc.Count))
	}
    return sb.String()
}

func (m *Model) viewDiagnostics() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0")).Render("RUNTIME DIAGNOSTICS") + "\n\n")
	
	sb.WriteString(fmt.Sprintf("Uptime:     %s (current run)\n", time.Since(m.lastUpdate).Round(time.Second)))
	sb.WriteString(fmt.Sprintf("Heap Alloc: %0.2f MB\n", float64(m.stats.HeapAlloc)/1024/1024))
	sb.WriteString(fmt.Sprintf("Sys RAM:    %0.2f MB\n", float64(m.stats.Sys)/1024/1024))
	sb.WriteString(fmt.Sprintf("Goroutines: %d\n", runtime.NumGoroutine()))
	
	sb.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0")).Render("SYSTEM FACTS") + "\n")
	sb.WriteString(fmt.Sprintf("Platform:   %s %s\n", runtime.GOOS, runtime.GOARCH))
	if m.db != nil {
		sb.WriteString(fmt.Sprintf("DB File:    %s\n", m.db.Path))
	}
	sb.WriteString(fmt.Sprintf("Refresh:    2.0s (bounded)\n"))

	return sb.String()
}

func Run(cfg config.Config, d *db.DB) error {
	p := tea.NewProgram(New(cfg, d), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

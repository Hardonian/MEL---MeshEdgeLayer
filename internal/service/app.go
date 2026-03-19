package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/meshstate"
	"github.com/mel-project/mel/internal/meshtastic"
	"github.com/mel-project/mel/internal/plugins"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/privacy"
	"github.com/mel-project/mel/internal/retention"
	"github.com/mel-project/mel/internal/transport"
	"github.com/mel-project/mel/internal/web"
)

type App struct {
	Cfg        config.Config
	Log        *logging.Logger
	DB         *db.DB
	Bus        *events.Bus
	State      *meshstate.State
	Web        *web.Server
	Transports []transport.Transport
	Plugins    []plugins.Plugin
}

func New(cfg config.Config) (*App, error) {
	log := logging.New()
	database, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}
	bus := events.New()
	state := meshstate.New()
	app := &App{Cfg: cfg, Log: log, DB: database, Bus: bus, State: state, Plugins: []plugins.Plugin{plugins.UnsafeMQTTPlugin{}}}
	app.Web = web.New(cfg, log, database, state, bus, app.TransportHealth, app.recommendations)
	for _, tc := range cfg.Transports {
		t, err := transport.Build(tc, log, bus)
		if err != nil {
			return nil, err
		}
		app.Transports = append(app.Transports, t)
	}
	return app, nil
}

func (a *App) recommendations() []policy.Recommendation { return policy.Explain(a.Cfg) }
func (a *App) TransportHealth() []transport.Health {
	out := make([]transport.Health, 0, len(a.Transports))
	for _, t := range a.Transports {
		out = append(out, t.Health())
	}
	return out
}

func (a *App) Start(ctx context.Context) error {
	if err := os.MkdirAll(a.Cfg.Storage.DataDir, 0o755); err != nil {
		return err
	}
	if err := retention.Run(a.DB, a.Cfg); err != nil {
		return err
	}
	for _, finding := range privacy.Audit(a.Cfg) {
		_ = a.DB.InsertAuditLog("privacy", finding.Severity, finding.Message, finding)
	}
	if len(enabledTransports(a.Cfg)) == 0 {
		a.Log.Info("no transports enabled; MEL will remain idle", map[string]any{"state": transport.StateConfigured})
		_ = a.DB.InsertAuditLog("transport", "warning", "no transports enabled; MEL will remain explicitly idle", map[string]any{"guidance": "Enable one transport before expecting stored packets."})
	}
	for _, tc := range a.Cfg.Transports {
		state := transport.StateConfigured
		detail := "configured; MEL has not attempted a live connection in this process yet"
		if !tc.Enabled {
			state = transport.StateDisabled
			detail = "disabled by config"
		} else if tc.Type == "serial" || tc.Type == "tcp" || tc.Type == "serialtcp" {
			_ = a.DB.InsertAuditLog("transport", "warning", "direct-node transport is implemented but not hardware-verified in this build context", map[string]any{"transport": tc.Name, "type": tc.Type, "source": tc.SourceLabel()})
		}
		a.persistTransportRuntime(tc, state, detail, "", "")
	}
	for _, t := range a.Transports {
		cfgTransport := findTransport(a.Cfg, t.Name())
		if !cfgTransport.Enabled {
			continue
		}
		go a.runTransport(ctx, t, cfgTransport)
	}
	go a.Web.Start(ctx)
	<-ctx.Done()
	for _, t := range a.Transports {
		_ = t.Close(context.Background())
		cfgTransport := findTransport(a.Cfg, t.Name())
		a.persistTransportRuntime(cfgTransport, transport.StateConfigured, "configured; process stopped", "", "")
	}
	return nil
}

func (a *App) runTransport(ctx context.Context, t transport.Transport, cfgTransport config.TransportConfig) {
	backoffSeconds := a.Cfg.RateLimits.TransportReconnectSeconds
	if cfgTransport.ReconnectSeconds > 0 {
		backoffSeconds = cfgTransport.ReconnectSeconds
	}
	backoff := time.Duration(backoffSeconds) * time.Second
	if backoff <= 0 {
		backoff = 10 * time.Second
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		a.persistTransportRuntime(cfgTransport, transport.StateAttempting, "attempting connection", "", "")
		if err := t.Connect(ctx); err != nil {
			a.recordTransportFailure(cfgTransport, "transport connect failed", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				continue
			}
		}
		a.persistTransportHealth(cfgTransport, t.Health())
		handler := func(topic string, payload []byte) error {
			err := a.ingest(cfgTransport, topic, payload)
			if err != nil {
				a.recordTransportFailure(cfgTransport, "transport ingest failed", err)
			}
			return err
		}
		if err := t.Subscribe(ctx, handler); err != nil && ctx.Err() == nil {
			a.recordTransportFailure(cfgTransport, "transport subscribe failed", err)
		}
		_ = t.Close(context.Background())
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
	}
}

func findTransport(cfg config.Config, name string) config.TransportConfig {
	for _, t := range cfg.Transports {
		if t.Name == name {
			return t
		}
	}
	return config.TransportConfig{}
}

func (a *App) ingest(cfgTransport config.TransportConfig, topic string, payload []byte) error {
	env, err := meshtastic.ParseEnvelope(payload)
	if err != nil {
		return err
	}
	rxTime := time.Unix(int64(env.Packet.RXTime), 0).UTC().Format(time.RFC3339)
	if env.Packet.RXTime == 0 {
		rxTime = time.Now().UTC().Format(time.RFC3339)
	}
	payloadJSON := map[string]any{"node_id": env.Packet.NodeID, "long_name": env.Packet.LongName, "short_name": env.Packet.ShortName, "topic": topic, "channel_id": env.ChannelID, "gateway_id": env.GatewayID, "transport_name": cfgTransport.Name}
	if env.Packet.Lat != nil || env.Packet.Lon != nil || env.Packet.Altitude != 0 {
		payloadJSON["position"] = map[string]any{"lat": meshtastic.RedactCoord(env.Packet.Lat), "lon": meshtastic.RedactCoord(env.Packet.Lon), "altitude": env.Packet.Altitude}
	}
	msg := map[string]any{"transport_name": cfgTransport.Name, "packet_id": int64(env.Packet.ID), "dedupe_hash": meshtastic.DedupeHash(env), "channel_id": env.ChannelID, "gateway_id": env.GatewayID, "from_node": int64(env.Packet.From), "to_node": int64(env.Packet.To), "portnum": int64(env.Packet.PortNum), "payload_text": env.Packet.PayloadText, "payload_json": payloadJSON, "raw_hex": env.RawHex, "rx_time": rxTime, "hop_limit": int64(env.Packet.HopLimit), "relay_node": int64(env.Packet.RelayNode)}
	stored, err := a.DB.InsertMessage(msg)
	if err != nil {
		return err
	}
	if !stored {
		a.Log.Info("duplicate packet ignored", map[string]any{"transport": cfgTransport.Name, "packet_id": env.Packet.ID})
		_ = a.DB.InsertAuditLog("transport", "warning", "duplicate packet ignored; state remains non-ingesting until a new packet stores successfully", map[string]any{"transport": cfgTransport.Name, "packet_id": env.Packet.ID, "dedupe_hash": meshtastic.DedupeHash(env)})
		return nil
	}
	node := map[string]any{"node_num": int64(env.Packet.From), "node_id": env.Packet.NodeID, "long_name": env.Packet.LongName, "short_name": env.Packet.ShortName, "last_seen": rxTime, "last_gateway_id": env.GatewayID, "last_snr": float64(env.Packet.RXSNR), "last_rssi": int64(env.Packet.RXRSSI), "lat_redacted": meshtastic.RedactCoord(env.Packet.Lat), "lon_redacted": meshtastic.RedactCoord(env.Packet.Lon), "altitude": int64(env.Packet.Altitude)}
	if err := a.DB.UpsertNode(node); err != nil {
		return err
	}
	if env.Packet.Lat != nil || env.Packet.Lon != nil || env.Packet.Altitude != 0 {
		if err := a.DB.InsertTelemetrySample(int64(env.Packet.From), "position", map[string]any{"lat_redacted": meshtastic.RedactCoord(env.Packet.Lat), "lon_redacted": meshtastic.RedactCoord(env.Packet.Lon), "altitude": int64(env.Packet.Altitude), "transport_name": cfgTransport.Name}, rxTime); err != nil {
			return err
		}
	}
	a.State.UpsertNode(meshstate.Node{Num: int64(env.Packet.From), ID: env.Packet.NodeID, LongName: env.Packet.LongName, ShortName: env.Packet.ShortName, LastSeen: rxTime, GatewayID: env.GatewayID})
	a.State.IncMessages()
	a.noteStoredMessage(cfgTransport, rxTime)
	summary := strings.TrimSpace(env.Packet.PayloadText)
	if summary == "" {
		summary = fmt.Sprintf("port %d packet", env.Packet.PortNum)
	}
	evt := events.Event{Type: "meshtastic.packet", Data: fmt.Sprintf("%s packet from %d (%s)", cfgTransport.Name, env.Packet.From, summary)}
	a.Bus.Publish(evt)
	for _, p := range a.Plugins {
		if alert := p.Handle(evt); alert != nil {
			_ = a.DB.InsertAuditLog("plugin", "warning", alert.Message, alert)
		}
	}
	return nil
}

func (a *App) noteStoredMessage(tc config.TransportConfig, rxTime string) {
	total, _, err := a.DB.MessageStatsByTransport(tc.Name)
	if err != nil {
		a.recordTransportFailure(tc, "transport message stats update failed", err)
		return
	}
	tr := db.TransportRuntime{
		Name:          tc.Name,
		Type:          tc.Type,
		Source:        tc.SourceLabel(),
		Enabled:       tc.Enabled,
		State:         transport.StateIngesting,
		Detail:        "transport stored at least one packet successfully in this process",
		LastMessageAt: rxTime,
		LastSuccessAt: time.Now().UTC().Format(time.RFC3339),
		TotalMessages: total,
	}
	if err := a.mergeTransportRuntime(tr); err != nil {
		a.Log.Error("transport runtime persist failed", map[string]any{"transport": tc.Name, "error": err.Error()})
	}
}

func (a *App) recordTransportFailure(tc config.TransportConfig, message string, err error) {
	a.Log.Error(message, map[string]any{"transport": tc.Name, "type": tc.Type, "error": err.Error()})
	_ = a.DB.InsertAuditLog("transport", "error", message, map[string]any{"transport": tc.Name, "type": tc.Type, "error": err.Error()})
	a.persistTransportRuntime(tc, transport.StateError, message, err.Error(), "")
}

func (a *App) persistTransportHealth(tc config.TransportConfig, h transport.Health) {
	tr := db.TransportRuntime{
		Name:            tc.Name,
		Type:            tc.Type,
		Source:          tc.SourceLabel(),
		Enabled:         tc.Enabled,
		State:           h.State,
		Detail:          h.Detail,
		LastAttemptAt:   h.LastAttemptAt,
		LastConnectedAt: h.LastConnectedAt,
		LastSuccessAt:   h.LastSuccessAt,
		LastMessageAt:   h.LastPacketAt,
		LastError:       h.LastError,
		TotalMessages:   h.PacketsRead,
	}
	if err := a.mergeTransportRuntime(tr); err != nil {
		a.Log.Error("transport runtime persist failed", map[string]any{"transport": tc.Name, "error": err.Error()})
	}
}

func (a *App) persistTransportRuntime(tc config.TransportConfig, state, detail, lastError, lastMessageAt string) {
	tr := db.TransportRuntime{
		Name:          tc.Name,
		Type:          tc.Type,
		Source:        tc.SourceLabel(),
		Enabled:       tc.Enabled,
		State:         state,
		Detail:        detail,
		LastError:     lastError,
		LastMessageAt: lastMessageAt,
	}
	if state == transport.StateAttempting {
		tr.LastAttemptAt = time.Now().UTC().Format(time.RFC3339)
	}
	if state == transport.StateConnectedNoData {
		tr.LastConnectedAt = time.Now().UTC().Format(time.RFC3339)
		tr.LastSuccessAt = tr.LastConnectedAt
	}
	if state == transport.StateIngesting && lastMessageAt != "" {
		tr.LastSuccessAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := a.mergeTransportRuntime(tr); err != nil {
		a.Log.Error("transport runtime persist failed", map[string]any{"transport": tc.Name, "error": err.Error()})
	}
}

func (a *App) mergeTransportRuntime(tr db.TransportRuntime) error {
	existing := map[string]db.TransportRuntime{}
	rows, err := a.DB.TransportRuntimeStatuses()
	if err == nil {
		for _, row := range rows {
			existing[row.Name] = row
		}
	}
	merged := existing[tr.Name]
	if tr.Name != "" {
		merged.Name = tr.Name
	}
	if tr.Type != "" {
		merged.Type = tr.Type
	}
	if tr.Source != "" {
		merged.Source = tr.Source
	}
	merged.Enabled = tr.Enabled
	if tr.State != "" {
		merged.State = tr.State
	}
	if tr.Detail != "" {
		merged.Detail = tr.Detail
	}
	if tr.LastAttemptAt != "" {
		merged.LastAttemptAt = tr.LastAttemptAt
	}
	if tr.LastConnectedAt != "" {
		merged.LastConnectedAt = tr.LastConnectedAt
	}
	if tr.LastSuccessAt != "" {
		merged.LastSuccessAt = tr.LastSuccessAt
	}
	if tr.LastMessageAt != "" {
		merged.LastMessageAt = tr.LastMessageAt
	}
	if tr.LastError != "" {
		merged.LastError = tr.LastError
	} else if tr.State == transport.StateIngesting || tr.State == transport.StateConnectedNoData || tr.State == transport.StateConfigured {
		merged.LastError = ""
	}
	if tr.TotalMessages > merged.TotalMessages {
		merged.TotalMessages = tr.TotalMessages
	}
	return a.DB.UpsertTransportRuntime(merged)
}

func enabledTransports(cfg config.Config) []config.TransportConfig {
	out := make([]config.TransportConfig, 0)
	for _, t := range cfg.Transports {
		if t.Enabled {
			out = append(out, t)
		}
	}
	return out
}

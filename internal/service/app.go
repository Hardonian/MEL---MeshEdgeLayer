package service

import (
	"context"
	"errors"
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
	statuspkg "github.com/mel-project/mel/internal/status"
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

func New(cfg config.Config, debug bool) (*App, error) {
	log := logging.New(cfg.Logging.Level, debug)
	database, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}
	bus := events.New()
	state := meshstate.New()
	app := &App{Cfg: cfg, Log: log, DB: database, Bus: bus, State: state, Plugins: []plugins.Plugin{plugins.UnsafeMQTTPlugin{}}}
	app.Web = web.New(cfg, log, database, state, bus, app.TransportHealth, app.recommendations, app.statusSnapshot)
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

func (a *App) statusSnapshot() (statuspkg.Snapshot, error) {
	return statuspkg.Collect(a.Cfg, a.DB, a.TransportHealth())
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
	if len(enabledTransportConfigs(a.Cfg)) == 0 {
		a.Log.Info("transport_idle", "no transports enabled; MEL will remain idle", map[string]any{"state": transport.StateConfigured})
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
		a.syncTransportRuntime(cfgTransport, t)
		a.Log.Debug("transport_attempt", "attempting transport connect", map[string]any{"transport": t.Name(), "type": cfgTransport.Type, "source": cfgTransport.SourceLabel()})
		if err := t.Connect(ctx); err != nil {
			a.Log.Error("transport_failed", "transport connect failed", map[string]any{"transport": t.Name(), "type": cfgTransport.Type, "error": err.Error()})
			_ = a.DB.InsertAuditLog("transport", "error", "transport connect failed", map[string]any{"transport": t.Name(), "error": err.Error()})
			a.syncTransportRuntime(cfgTransport, t)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				continue
			}
		}
		a.Log.Info("transport_connected", "transport connected", map[string]any{"transport": t.Name(), "type": cfgTransport.Type, "source": cfgTransport.SourceLabel()})
		_ = a.DB.InsertAuditLog("transport", "info", "transport connected", map[string]any{"transport": t.Name(), "type": cfgTransport.Type, "source": cfgTransport.SourceLabel()})
		a.syncTransportRuntime(cfgTransport, t)
		if err := t.Subscribe(ctx, func(topic string, payload []byte) error { return a.ingest(t, topic, payload) }); err != nil && ctx.Err() == nil {
			a.Log.Error("transport_failed", "transport subscribe failed", map[string]any{"transport": t.Name(), "error": err.Error()})
			_ = a.DB.InsertAuditLog("transport", "error", "transport subscribe failed", map[string]any{"transport": t.Name(), "error": err.Error()})
		}
		a.syncTransportRuntime(cfgTransport, t)
		_ = t.Close(context.Background())
		a.syncTransportRuntime(cfgTransport, t)
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

func (a *App) ingest(t transport.Transport, topic string, payload []byte) error {
	env, err := meshtastic.ParseEnvelope(payload)
	if err != nil {
		a.Log.Error("ingest_dropped", "failed to parse packet", map[string]any{"transport": t.Name(), "error": err.Error()})
		t.MarkDrop("failed to parse packet")
		_ = a.DB.InsertDeadLetter(db.DeadLetter{TransportName: t.Name(), Topic: topic, Reason: "parse failure", PayloadHex: fmt.Sprintf("%x", payload), Details: map[string]any{"error": err.Error()}})
		a.syncTransportRuntime(findTransport(a.Cfg, t.Name()), t)
		return err
	}
	rxAt := time.Now().UTC()
	rxTime := time.Unix(int64(env.Packet.RXTime), 0).UTC().Format(time.RFC3339)
	if env.Packet.RXTime == 0 {
		rxTime = rxAt.Format(time.RFC3339)
	}
	messageType, payloadJSON, telemetryType, telemetryValue := buildPayloadEnvelope(t.Name(), topic, env)
	msg := map[string]any{"transport_name": t.Name(), "packet_id": int64(env.Packet.ID), "dedupe_hash": meshtastic.DedupeHash(env), "channel_id": env.ChannelID, "gateway_id": env.GatewayID, "from_node": int64(env.Packet.From), "to_node": int64(env.Packet.To), "portnum": int64(env.Packet.PortNum), "payload_text": env.Packet.PayloadText, "payload_json": payloadJSON, "raw_hex": env.RawHex, "rx_time": rxTime, "hop_limit": int64(env.Packet.HopLimit), "relay_node": int64(env.Packet.RelayNode)}
	inserted, err := a.DB.InsertMessage(msg)
	if err != nil {
		a.Log.Error("db_error", "message insert failed", map[string]any{"transport": t.Name(), "error": err.Error()})
		t.MarkDrop("database write failed")
		_ = a.DB.InsertDeadLetter(db.DeadLetter{TransportName: t.Name(), Topic: topic, Reason: "database write failed", PayloadHex: env.RawHex, Details: map[string]any{"error": err.Error(), "from_node": env.Packet.From, "packet_id": env.Packet.ID}})
		a.syncTransportRuntime(findTransport(a.Cfg, t.Name()), t)
		return err
	}
	if !inserted {
		a.Log.Info("ingest_dropped", "duplicate message ignored", map[string]any{"transport": t.Name(), "dedupe_hash": msg["dedupe_hash"]})
		t.MarkDrop("duplicate packet ignored after dedupe")
		a.syncTransportRuntime(findTransport(a.Cfg, t.Name()), t)
		return nil
	}
	node := map[string]any{"node_num": int64(env.Packet.From), "node_id": env.Packet.NodeID, "long_name": env.Packet.LongName, "short_name": env.Packet.ShortName, "last_seen": rxTime, "last_gateway_id": env.GatewayID, "last_snr": float64(env.Packet.RXSNR), "last_rssi": int64(env.Packet.RXRSSI), "lat_redacted": meshtastic.RedactCoord(env.Packet.Lat), "lon_redacted": meshtastic.RedactCoord(env.Packet.Lon), "altitude": int64(env.Packet.Altitude)}
	if err := a.DB.UpsertNode(node); err != nil {
		a.Log.Error("db_error", "node upsert failed", map[string]any{"transport": t.Name(), "error": err.Error()})
		t.MarkDrop("node upsert failed")
		_ = a.DB.InsertDeadLetter(db.DeadLetter{TransportName: t.Name(), Topic: topic, Reason: "node upsert failed", PayloadHex: env.RawHex, Details: map[string]any{"error": err.Error(), "from_node": env.Packet.From}})
		a.syncTransportRuntime(findTransport(a.Cfg, t.Name()), t)
		return err
	}
	if telemetryType != "" {
		if err := a.DB.InsertTelemetrySample(int64(env.Packet.From), telemetryType, telemetryValue, rxTime); err != nil {
			a.Log.Error("db_error", "telemetry insert failed", map[string]any{"transport": t.Name(), "error": err.Error()})
			t.MarkDrop("telemetry insert failed")
			_ = a.DB.InsertDeadLetter(db.DeadLetter{TransportName: t.Name(), Topic: topic, Reason: "telemetry insert failed", PayloadHex: env.RawHex, Details: map[string]any{"error": err.Error(), "from_node": env.Packet.From, "telemetry_type": telemetryType}})
			a.syncTransportRuntime(findTransport(a.Cfg, t.Name()), t)
			return err
		}
	}
	a.State.UpsertNode(meshstate.Node{Num: int64(env.Packet.From), ID: env.Packet.NodeID, LongName: env.Packet.LongName, ShortName: env.Packet.ShortName, LastSeen: rxTime, GatewayID: env.GatewayID})
	a.State.IncMessages()
	t.MarkIngest(rxAt)
	summary := strings.TrimSpace(env.Packet.PayloadText)
	if summary == "" {
		summary = fmt.Sprintf("%s packet", messageType)
	}
	evt := events.Event{Type: "meshtastic.packet", Data: fmt.Sprintf("%s packet from %d (%s)", t.Name(), env.Packet.From, summary)}
	a.Bus.Publish(evt)
	for _, p := range a.Plugins {
		if alert := p.Handle(evt); alert != nil {
			_ = a.DB.InsertAuditLog("plugin", "warning", alert.Message, alert)
		}
	}
	_ = a.DB.InsertAuditLog("node", "info", "node observed via transport", map[string]any{"transport": t.Name(), "topic": topic, "node_num": env.Packet.From, "node_id": env.Packet.NodeID, "gateway_id": env.GatewayID})
	a.syncTransportRuntime(findTransport(a.Cfg, t.Name()), t)
	a.Log.Info("ingest_received", "message persisted", map[string]any{"transport": t.Name(), "message_type": messageType, "from_node": env.Packet.From, "portnum": env.Packet.PortNum})
	return nil
}

func (a *App) syncTransportRuntime(tc config.TransportConfig, t transport.Transport) {
	if a.DB == nil {
		return
	}
	h := t.Health()
	lastMessageAt := h.LastIngestAt
	if lastMessageAt == "" {
		lastMessageAt = h.LastSuccessAt
	}
	_ = a.DB.UpsertTransportRuntime(db.TransportRuntime{
		Name:            tc.Name,
		Type:            tc.Type,
		Source:          tc.SourceLabel(),
		Enabled:         tc.Enabled,
		State:           h.State,
		Detail:          h.Detail,
		LastAttemptAt:   h.LastAttemptAt,
		LastConnectedAt: h.LastConnectedAt,
		LastSuccessAt:   h.LastSuccessAt,
		LastMessageAt:   lastMessageAt,
		LastError:       h.LastError,
		TotalMessages:   h.TotalMessages,
	})
}

func (a *App) persistTransportRuntime(tc config.TransportConfig, state, detail, lastError, lastMessageAt string) {
	if a.DB == nil {
		return
	}
	_ = a.DB.UpsertTransportRuntime(db.TransportRuntime{
		Name:          tc.Name,
		Type:          tc.Type,
		Source:        tc.SourceLabel(),
		Enabled:       tc.Enabled,
		State:         state,
		Detail:        detail,
		LastError:     lastError,
		LastMessageAt: lastMessageAt,
	})
}

func buildPayloadEnvelope(transportName, topic string, env meshtastic.Envelope) (string, map[string]any, string, map[string]any) {
	messageType := meshtastic.MessageType(env.Packet)
	payloadJSON := map[string]any{
		"node_id":         env.Packet.NodeID,
		"long_name":       env.Packet.LongName,
		"short_name":      env.Packet.ShortName,
		"topic":           topic,
		"channel_id":      env.ChannelID,
		"gateway_id":      env.GatewayID,
		"transport_name":  transportName,
		"message_type":    messageType,
		"raw_payload_hex": env.Packet.PayloadHex(),
	}
	telemetryType := ""
	telemetryValue := map[string]any{}
	switch messageType {
	case "position":
		payloadJSON["position"] = map[string]any{"lat": meshtastic.RedactCoord(env.Packet.Lat), "lon": meshtastic.RedactCoord(env.Packet.Lon), "altitude": env.Packet.Altitude}
		telemetryType = "position"
		telemetryValue = map[string]any{"lat_redacted": meshtastic.RedactCoord(env.Packet.Lat), "lon_redacted": meshtastic.RedactCoord(env.Packet.Lon), "altitude": int64(env.Packet.Altitude), "transport_name": transportName}
	case "node_info":
		payloadJSON["user"] = map[string]any{"node_id": env.Packet.NodeID, "long_name": env.Packet.LongName, "short_name": env.Packet.ShortName}
	case "telemetry":
		payloadJSON["telemetry"] = map[string]any{"parser": "raw", "note": "payload stored as raw bytes because this repo does not vendor the full telemetry protobuf schema"}
		telemetryType = "telemetry_raw"
		telemetryValue = map[string]any{"transport_name": transportName, "raw_payload_hex": env.Packet.PayloadHex(), "portnum": env.Packet.PortNum}
	case "text":
		payloadJSON["text"] = strings.TrimSpace(env.Packet.PayloadText)
	default:
		payloadJSON["unknown"] = true
	}
	return messageType, payloadJSON, telemetryType, telemetryValue
}

var errDuplicateMessage = errors.New("duplicate message ignored")

func enabledTransportConfigs(cfg config.Config) []config.TransportConfig {
	out := make([]config.TransportConfig, 0, len(cfg.Transports))
	for _, t := range cfg.Transports {
		if t.Enabled {
			out = append(out, t)
		}
	}
	return out
}

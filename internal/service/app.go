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
		if err := t.Connect(ctx); err != nil {
			a.Log.Error("transport connect failed", map[string]any{"transport": t.Name(), "type": cfgTransport.Type, "error": err.Error()})
			_ = a.DB.InsertAuditLog("transport", "error", "transport connect failed", map[string]any{"transport": t.Name(), "error": err.Error()})
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				continue
			}
		}
		if err := t.Subscribe(ctx, func(topic string, payload []byte) error { return a.ingest(t.Name(), topic, payload) }); err != nil && ctx.Err() == nil {
			a.Log.Error("transport subscribe failed", map[string]any{"transport": t.Name(), "error": err.Error()})
			_ = a.DB.InsertAuditLog("transport", "error", "transport subscribe failed", map[string]any{"transport": t.Name(), "error": err.Error()})
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

func (a *App) ingest(transportName, topic string, payload []byte) error {
	env, err := meshtastic.ParseEnvelope(payload)
	if err != nil {
		return err
	}
	rxTime := time.Unix(int64(env.Packet.RXTime), 0).UTC().Format(time.RFC3339)
	if env.Packet.RXTime == 0 {
		rxTime = time.Now().UTC().Format(time.RFC3339)
	}
	payloadJSON := map[string]any{"node_id": env.Packet.NodeID, "long_name": env.Packet.LongName, "short_name": env.Packet.ShortName, "topic": topic, "channel_id": env.ChannelID, "gateway_id": env.GatewayID, "transport_name": transportName}
	if env.Packet.Lat != nil || env.Packet.Lon != nil || env.Packet.Altitude != 0 {
		payloadJSON["position"] = map[string]any{"lat": meshtastic.RedactCoord(env.Packet.Lat), "lon": meshtastic.RedactCoord(env.Packet.Lon), "altitude": env.Packet.Altitude}
	}
	msg := map[string]any{"transport_name": transportName, "packet_id": int64(env.Packet.ID), "dedupe_hash": meshtastic.DedupeHash(env), "channel_id": env.ChannelID, "gateway_id": env.GatewayID, "from_node": int64(env.Packet.From), "to_node": int64(env.Packet.To), "portnum": int64(env.Packet.PortNum), "payload_text": env.Packet.PayloadText, "payload_json": payloadJSON, "raw_hex": env.RawHex, "rx_time": rxTime, "hop_limit": int64(env.Packet.HopLimit), "relay_node": int64(env.Packet.RelayNode)}
	if err := a.DB.InsertMessage(msg); err != nil {
		return err
	}
	node := map[string]any{"node_num": int64(env.Packet.From), "node_id": env.Packet.NodeID, "long_name": env.Packet.LongName, "short_name": env.Packet.ShortName, "last_seen": rxTime, "last_gateway_id": env.GatewayID, "last_snr": float64(env.Packet.RXSNR), "last_rssi": int64(env.Packet.RXRSSI), "lat_redacted": meshtastic.RedactCoord(env.Packet.Lat), "lon_redacted": meshtastic.RedactCoord(env.Packet.Lon), "altitude": int64(env.Packet.Altitude)}
	if err := a.DB.UpsertNode(node); err != nil {
		return err
	}
	if env.Packet.Lat != nil || env.Packet.Lon != nil || env.Packet.Altitude != 0 {
		if err := a.DB.InsertTelemetrySample(int64(env.Packet.From), "position", map[string]any{"lat_redacted": meshtastic.RedactCoord(env.Packet.Lat), "lon_redacted": meshtastic.RedactCoord(env.Packet.Lon), "altitude": int64(env.Packet.Altitude), "transport_name": transportName}, rxTime); err != nil {
			return err
		}
	}
	a.State.UpsertNode(meshstate.Node{Num: int64(env.Packet.From), ID: env.Packet.NodeID, LongName: env.Packet.LongName, ShortName: env.Packet.ShortName, LastSeen: rxTime, GatewayID: env.GatewayID})
	a.State.IncMessages()
	summary := strings.TrimSpace(env.Packet.PayloadText)
	if summary == "" {
		summary = fmt.Sprintf("port %d packet", env.Packet.PortNum)
	}
	evt := events.Event{Type: "meshtastic.packet", Data: fmt.Sprintf("%s packet from %d (%s)", transportName, env.Packet.From, summary)}
	a.Bus.Publish(evt)
	for _, p := range a.Plugins {
		if alert := p.Handle(evt); alert != nil {
			_ = a.DB.InsertAuditLog("plugin", "warning", alert.Message, alert)
		}
	}
	return nil
}

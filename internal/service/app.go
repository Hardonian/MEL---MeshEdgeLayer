package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
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
	Plugin     plugins.Plugin
}

func New(cfg config.Config) (*App, error) {
	log := logging.New()
	database, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}
	bus := events.New()
	state := meshstate.New()
	app := &App{Cfg: cfg, Log: log, DB: database, Bus: bus, State: state, Plugin: plugins.UnsafeMQTTPlugin{}}
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
	for _, t := range a.Transports {
		cfgTransport := findTransport(a.Cfg, t.Name())
		if !cfgTransport.Enabled {
			continue
		}
		if err := t.Connect(ctx); err != nil {
			a.Log.Error("transport connect failed", map[string]any{"transport": t.Name(), "error": err.Error()})
			continue
		}
		if err := t.Subscribe(ctx, func(topic string, payload []byte) error { return a.ingest(t.Name(), topic, payload) }); err != nil {
			a.Log.Error("transport subscribe failed", map[string]any{"transport": t.Name(), "error": err.Error()})
		}
	}
	go a.Web.Start(ctx)
	<-ctx.Done()
	for _, t := range a.Transports {
		_ = t.Close(context.Background())
	}
	return nil
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
	dedupe := sha256.Sum256(append([]byte(topic), payload...))
	rxTime := time.Unix(int64(env.Packet.RXTime), 0).UTC().Format(time.RFC3339)
	if env.Packet.RXTime == 0 {
		rxTime = time.Now().UTC().Format(time.RFC3339)
	}
	msg := map[string]any{"transport_name": transportName, "packet_id": int64(env.Packet.ID), "dedupe_hash": hex.EncodeToString(dedupe[:]), "channel_id": env.ChannelID, "gateway_id": env.GatewayID, "from_node": int64(env.Packet.From), "to_node": int64(env.Packet.To), "portnum": int64(env.Packet.PortNum), "payload_text": env.Packet.PayloadText, "payload_json": map[string]any{"node_id": env.Packet.NodeID, "long_name": env.Packet.LongName, "short_name": env.Packet.ShortName}, "raw_hex": env.RawHex, "rx_time": rxTime, "hop_limit": int64(env.Packet.HopLimit), "relay_node": int64(env.Packet.RelayNode)}
	if err := a.DB.InsertMessage(msg); err != nil {
		return err
	}
	node := map[string]any{"node_num": int64(env.Packet.From), "node_id": env.Packet.NodeID, "long_name": env.Packet.LongName, "short_name": env.Packet.ShortName, "last_seen": rxTime, "last_gateway_id": env.GatewayID, "last_snr": float64(env.Packet.RXSNR), "last_rssi": int64(env.Packet.RXRSSI), "lat_redacted": meshtastic.RedactCoord(env.Packet.Lat), "lon_redacted": meshtastic.RedactCoord(env.Packet.Lon), "altitude": int64(env.Packet.Altitude)}
	if err := a.DB.UpsertNode(node); err != nil {
		return err
	}
	a.State.UpsertNode(meshstate.Node{Num: int64(env.Packet.From), ID: env.Packet.NodeID, LongName: env.Packet.LongName, ShortName: env.Packet.ShortName, LastSeen: rxTime, GatewayID: env.GatewayID})
	a.State.IncMessages()
	a.Bus.Publish(events.Event{Type: "meshtastic.packet", Data: fmt.Sprintf("%s packet from %d", transportName, env.Packet.From)})
	for _, f := range privacy.Audit(a.Cfg) {
		a.Bus.Publish(events.Event{Type: "privacy.audit", Data: f.Message})
	}
	return nil
}

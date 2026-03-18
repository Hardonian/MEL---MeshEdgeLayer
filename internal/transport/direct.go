package transport

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/meshtastic"
)

const (
	directStart1   = 0x94
	directStart2   = 0xC3
	directHeaderSz = 4
	directMaxFrame = 512
)

type Direct struct {
	cfg    config.TransportConfig
	log    *logging.Logger
	bus    *events.Bus
	mu     sync.Mutex
	rw     io.ReadWriteCloser
	health Health
	dial   func(context.Context, config.TransportConfig) (io.ReadWriteCloser, error)
}

func NewDirect(cfg config.TransportConfig, log *logging.Logger, bus *events.Bus) *Direct {
	status := "partial"
	notes := "real direct-node ingest for serial/TCP Meshtastic streams; send/metadata/node fetch remain disabled until proven"
	if cfg.Type == "serial" || cfg.Type == "tcp" || cfg.Type == "serialtcp" {
		status = "supported"
	}
	caps := capabilityDefaults(cfg, true, false, false, false, true, false, status, notes)
	return &Direct{cfg: cfg, log: log, bus: bus, health: Health{Name: cfg.Name, Type: cfg.Type, Source: cfg.SourceLabel(), Detail: "disconnected", Capabilities: caps}, dial: openDirectConnection}
}

func (d *Direct) Name() string                   { return d.cfg.Name }
func (d *Direct) SourceType() string             { return d.cfg.Type }
func (d *Direct) Capabilities() CapabilityMatrix { return d.Health().Capabilities }
func (d *Direct) Health() Health {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.health
}
func (d *Direct) setHealth(update func(*Health)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	update(&d.health)
}

func (d *Direct) Connect(ctx context.Context) error {
	d.setHealth(func(h *Health) {
		h.ReconnectAttempts++
		h.Source = d.cfg.SourceLabel()
	})
	rw, err := d.dial(ctx, d.cfg)
	if err != nil {
		d.setHealth(func(h *Health) {
			h.OK = false
			h.Detail = "connect failed"
			h.LastError = err.Error()
			h.LastDisconnected = time.Now().UTC().Format(time.RFC3339)
		})
		return err
	}
	d.mu.Lock()
	d.rw = rw
	d.mu.Unlock()
	d.setHealth(func(h *Health) {
		h.OK = true
		h.Detail = "connected; waiting for radio packets"
		h.LastConnectedAt = time.Now().UTC().Format(time.RFC3339)
		h.LastError = ""
	})
	return nil
}

func (d *Direct) Close(context.Context) error {
	d.mu.Lock()
	rw := d.rw
	d.rw = nil
	d.mu.Unlock()
	if rw == nil {
		return nil
	}
	d.setHealth(func(h *Health) {
		h.OK = false
		h.Detail = "closed"
		h.LastDisconnected = time.Now().UTC().Format(time.RFC3339)
	})
	return rw.Close()
}

func (d *Direct) Subscribe(ctx context.Context, handler PacketHandler) error {
	d.mu.Lock()
	rw := d.rw
	d.mu.Unlock()
	if rw == nil {
		return errors.New("not connected")
	}
	reader := bufio.NewReader(rw)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		frame, err := readDirectFrame(reader)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				d.markFailure(err, "stream disconnected")
				return err
			}
			d.markFailure(err, "frame read failed")
			return err
		}
		env, err := meshtastic.ParseDirectFromRadio(frame)
		if err != nil {
			d.setHealth(func(h *Health) {
				h.PacketsDropped++
				h.LastError = err.Error()
				h.Detail = "connected; ignoring non-packet radio frame"
			})
			continue
		}
		wrapped := meshtastic.DirectPacketToEnvelope(env.PacketRaw)
		if err := handler(d.cfg.Name, wrapped); err != nil {
			d.setHealth(func(h *Health) {
				h.PacketsDropped++
				h.LastError = err.Error()
				h.Detail = "connected; ingest handler failed"
			})
			d.bus.Publish(events.Event{Type: "transport.error", Data: err.Error()})
			continue
		}
		d.setHealth(func(h *Health) {
			h.OK = true
			h.PacketsRead++
			h.LastPacketAt = time.Now().UTC().Format(time.RFC3339)
			h.Detail = "connected and ingesting live radio packets"
		})
	}
}

func (d *Direct) SendPacket(context.Context, []byte) error {
	return errors.New("direct-node send is disabled in this milestone")
}
func (d *Direct) FetchMetadata(context.Context) (map[string]any, error) {
	return nil, errors.New("metadata fetch is not implemented for direct-node transport")
}
func (d *Direct) FetchNodes(context.Context) ([]map[string]any, error) {
	return nil, errors.New("node fetch is not implemented for direct-node transport")
}

func (d *Direct) markFailure(err error, detail string) {
	d.setHealth(func(h *Health) {
		h.OK = false
		h.LastError = err.Error()
		h.Detail = detail
		h.LastDisconnected = time.Now().UTC().Format(time.RFC3339)
	})
	d.bus.Publish(events.Event{Type: "transport.error", Data: err.Error()})
}

func openDirectConnection(ctx context.Context, cfg config.TransportConfig) (io.ReadWriteCloser, error) {
	switch cfg.Type {
	case "serial":
		device := cfg.SerialDevice
		if device == "" {
			device = cfg.Endpoint
		}
		if device == "" {
			return nil, errors.New("serial transport requires serial_device")
		}
		if err := configureSerial(ctx, device, cfg.SerialBaud); err != nil {
			return nil, err
		}
		return os.OpenFile(device, os.O_RDWR, 0)
	case "tcp", "serialtcp":
		endpoint := cfg.Endpoint
		if endpoint == "" {
			endpoint = net.JoinHostPort(cfg.TCPHost, fmt.Sprint(cfg.TCPPort))
		}
		dialer := &net.Dialer{Timeout: 5 * time.Second}
		return dialer.DialContext(ctx, "tcp", endpoint)
	default:
		return nil, fmt.Errorf("direct-node transport type %s is not supported", cfg.Type)
	}
}

func configureSerial(ctx context.Context, device string, baud int) error {
	if baud <= 0 {
		baud = 115200
	}
	cmd := exec.CommandContext(ctx, "stty", "-F", device, fmt.Sprint(baud), "raw", "-echo")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("serial setup failed for %s: %w: %s", device, err, string(out))
	}
	return nil
}

func readDirectFrame(r *bufio.Reader) ([]byte, error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b != directStart1 {
			continue
		}
		b2, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b2 != directStart2 {
			continue
		}
		header := make([]byte, 2)
		if _, err := io.ReadFull(r, header); err != nil {
			return nil, err
		}
		ln := int(binary.BigEndian.Uint16(header))
		if ln <= 0 || ln > directMaxFrame {
			return nil, fmt.Errorf("invalid direct frame length %d", ln)
		}
		body := make([]byte, ln)
		if _, err := io.ReadFull(r, body); err != nil {
			return nil, err
		}
		return body, nil
	}
}

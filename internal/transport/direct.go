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
	directMaxFrame = 512
)

var errDirectInvalidFrame = errors.New("invalid direct frame")

type deadlineReadWriteCloser interface {
	io.ReadWriteCloser
	SetReadDeadline(time.Time) error
}

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
	notes := "real direct-node ingest for serial/TCP Meshtastic streams with timeout-based deadlock detection; send/metadata/node fetch remain disabled until proven"
	if cfg.Type == "serial" || cfg.Type == "tcp" || cfg.Type == "serialtcp" {
		status = "supported"
	}
	caps := capabilityDefaults(cfg, true, false, false, false, true, false, status, notes)
	state := StateConfiguredNotAttempted
	detail := "configured but not yet attempted"
	if !cfg.Enabled {
		state = StateDisabled
		detail = "disabled by config"
	}
	return &Direct{
		cfg:    cfg,
		log:    log,
		bus:    bus,
		health: Health{Name: cfg.Name, Type: cfg.Type, Source: cfg.SourceLabel(), State: state, Detail: detail, Capabilities: caps},
		dial:   openDirectConnection,
	}
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
	now := time.Now().UTC().Format(time.RFC3339)
	d.setHealth(func(h *Health) {
		h.ReconnectAttempts++
		h.Source = d.cfg.SourceLabel()
		h.State = StateAttempting
		h.Detail = "attempting direct-node connection"
		h.LastAttemptAt = now
	})
	rw, err := d.dial(ctx, d.cfg)
	if err != nil {
		d.setHealth(func(h *Health) {
			h.OK = false
			h.State = StateConfiguredOffline
			h.Detail = "configured transport is offline"
			h.LastError = err.Error()
			h.ErrorCount++
			h.LastDisconnected = time.Now().UTC().Format(time.RFC3339)
		})
		return err
	}
	d.mu.Lock()
	d.rw = rw
	d.mu.Unlock()
	d.setHealth(func(h *Health) {
		h.OK = true
		h.State = StateConnectedNoIngest
		h.Detail = "connected; waiting for direct-node traffic"
		h.LastConnectedAt = time.Now().UTC().Format(time.RFC3339)
		h.LastSuccessAt = h.LastConnectedAt
		h.LastHeartbeatAt = h.LastConnectedAt
		h.LastError = ""
		h.ConsecutiveTimeouts = 0
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
		if h.State != StateDisabled {
			h.State = StateConfiguredOffline
			if h.TotalMessages > 0 {
				h.Detail = "disconnected; historical ingest exists but no live direct connection is active"
			} else {
				h.Detail = "disconnected; waiting to retry"
			}
		}
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
	readTimeout := time.Duration(d.cfg.ReadTimeoutSec) * time.Second
	if readTimeout <= 0 {
		readTimeout = 15 * time.Second
	}
	maxTimeouts := d.cfg.MaxTimeouts
	if maxTimeouts <= 0 {
		maxTimeouts = 3
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		frame, err := readDirectFrameWithTimeout(ctx, rw, reader, readTimeout)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			if isTimeout(err) {
				timeouts := d.noteTimeout("connected; waiting for direct-node heartbeat or payload")
				if timeouts >= uint64(maxTimeouts) {
					d.markFailure(fmt.Errorf("direct transport exceeded %d consecutive read timeouts", maxTimeouts), "direct transport stalled; reconnecting")
					return err
				}
				continue
			}
			if errors.Is(err, errDirectInvalidFrame) {
				d.markDropWithState("connected; malformed direct frame ignored", err)
				continue
			}
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				d.markFailure(err, "stream disconnected; waiting to retry")
				return err
			}
			d.markFailure(err, "frame read failed; waiting to retry")
			return err
		}
		env, err := meshtastic.ParseDirectFromRadio(frame)
		if err != nil {
			d.markDropWithState("connected; ignoring non-packet radio frame", err)
			continue
		}
		wrapped := meshtastic.DirectPacketToEnvelope(env.PacketRaw)
		if err := handler(d.cfg.Name, wrapped); err != nil {
			d.markDropWithState("connected; ingest handler rejected packet", err)
			d.bus.Publish(events.Event{Type: "transport.error", Data: err.Error()})
			continue
		}
		d.setHealth(func(h *Health) {
			h.OK = true
			if h.TotalMessages == 0 {
				h.State = StateConnectedNoIngest
				h.Detail = "connected; packet received but not yet confirmed stored"
			}
			h.LastHeartbeatAt = time.Now().UTC().Format(time.RFC3339)
			h.ConsecutiveTimeouts = 0
			h.LastError = ""
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
func (d *Direct) MarkIngest(at time.Time) {
	d.setHealth(func(h *Health) {
		h.OK = true
		h.State = StateIngesting
		h.TotalMessages++
		h.LastIngestAt = at.UTC().Format(time.RFC3339)
		h.LastHeartbeatAt = h.LastIngestAt
		h.ConsecutiveTimeouts = 0
		h.LastError = ""
		h.Detail = "live ingest confirmed by SQLite writes"
	})
}
func (d *Direct) MarkDrop(reason string) { d.markDropWithState(reason, nil) }

func (d *Direct) noteTimeout(detail string) uint64 {
	var timeouts uint64
	d.setHealth(func(h *Health) {
		h.OK = true
		h.ConsecutiveTimeouts++
		timeouts = h.ConsecutiveTimeouts
		if h.TotalMessages > 0 {
			h.State = StateIngesting
		} else {
			h.State = StateConnectedNoIngest
		}
		h.Detail = detail
	})
	return timeouts
}

func (d *Direct) markDropWithState(reason string, err error) {
	d.setHealth(func(h *Health) {
		h.PacketsDropped++
		h.State = StateError
		h.Detail = reason
		if err != nil {
			h.LastError = err.Error()
			h.ErrorCount++
		}
	})
}

func (d *Direct) markFailure(err error, detail string) {
	d.setHealth(func(h *Health) {
		h.OK = false
		h.State = StateError
		h.LastError = err.Error()
		h.Detail = detail
		h.ErrorCount++
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
	cmd := exec.CommandContext(ctx, "stty", "-F", device, fmt.Sprint(baud), "raw", "-echo", "min", "0", "time", "10")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("serial setup failed for %s: %w: %s", device, err, string(out))
	}
	return nil
}

func readDirectFrame(ctx context.Context, rw io.ReadWriteCloser, r *bufio.Reader) ([]byte, error) {
	return readDirectFrameWithTimeout(ctx, rw, r, time.Second)
}

func readDirectFrameWithTimeout(ctx context.Context, rw io.ReadWriteCloser, r *bufio.Reader, readTimeout time.Duration) ([]byte, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		b, err := readDirectByte(ctx, rw, r, readTimeout)
		if err != nil {
			return nil, err
		}
		if b != directStart1 {
			continue
		}
		b2, err := readDirectByte(ctx, rw, r, readTimeout)
		if err != nil {
			return nil, err
		}
		if b2 != directStart2 {
			continue
		}
		header := make([]byte, 2)
		if err := readDirectFull(ctx, rw, r, header, readTimeout); err != nil {
			return nil, err
		}
		ln := int(binary.BigEndian.Uint16(header))
		if ln <= 0 || ln > directMaxFrame {
			return nil, fmt.Errorf("%w: invalid direct frame length %d", errDirectInvalidFrame, ln)
		}
		body := make([]byte, ln)
		if err := readDirectFull(ctx, rw, r, body, readTimeout); err != nil {
			return nil, err
		}
		return body, nil
	}
}

func readDirectByte(ctx context.Context, rw io.ReadWriteCloser, r *bufio.Reader, readTimeout time.Duration) (byte, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if readTimeout <= 0 {
		readTimeout = time.Second
	}
	if err := setReadDeadline(rw, time.Now().Add(readTimeout)); err != nil {
		return 0, err
	}
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	_ = clearReadDeadline(rw)
	return b, nil
}

func readDirectFull(ctx context.Context, rw io.ReadWriteCloser, r *bufio.Reader, buf []byte, readTimeout time.Duration) error {
	if readTimeout <= 0 {
		readTimeout = time.Second
	}
	for n := 0; n < len(buf); {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := setReadDeadline(rw, time.Now().Add(readTimeout)); err != nil {
			return err
		}
		readN, err := r.Read(buf[n:])
		if readN > 0 {
			n += readN
		}
		if err == nil {
			continue
		}
		return err
	}
	return clearReadDeadline(rw)
}

func setReadDeadline(rw io.ReadWriteCloser, deadline time.Time) error {
	d, ok := rw.(deadlineReadWriteCloser)
	if !ok {
		return nil
	}
	return d.SetReadDeadline(deadline)
}

func clearReadDeadline(rw io.ReadWriteCloser) error {
	d, ok := rw.(deadlineReadWriteCloser)
	if !ok {
		return nil
	}
	return d.SetReadDeadline(time.Time{})
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

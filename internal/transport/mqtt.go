package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
)

type MQTT struct {
	cfg    config.TransportConfig
	log    *logging.Logger
	bus    *events.Bus
	conn   net.Conn
	health Health
	mu     sync.Mutex
}

func NewMQTT(cfg config.TransportConfig, log *logging.Logger, bus *events.Bus) *MQTT {
	caps := capabilityDefaults(cfg, true, false, false, false, true, false, "supported", "real MQTT subscribe path; MEL does not claim publish/config control in this milestone")
	return &MQTT{cfg: cfg, log: log, bus: bus, health: Health{Name: cfg.Name, Type: cfg.Type, Source: cfg.Endpoint, Detail: "disabled", Capabilities: caps}}
}
func (m *MQTT) Name() string                   { return m.cfg.Name }
func (m *MQTT) SourceType() string             { return m.cfg.Type }
func (m *MQTT) Capabilities() CapabilityMatrix { return m.Health().Capabilities }
func (m *MQTT) Health() Health {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.health
}
func (m *MQTT) setHealth(update func(*Health)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	update(&m.health)
}
func (m *MQTT) Connect(ctx context.Context) error {
	m.setHealth(func(h *Health) { h.ReconnectAttempts++; h.Source = m.cfg.Endpoint })
	conn, err := dialWithTimeout(m.cfg.Endpoint)
	if err != nil {
		m.setHealth(func(h *Health) {
			h.OK = false
			h.Detail = "connect failed"
			h.LastError = err.Error()
			h.LastDisconnected = time.Now().UTC().Format(time.RFC3339)
		})
		return err
	}
	m.conn = conn
	pkt := bytes.NewBuffer(nil)
	pkt.WriteByte(0x10)
	payload := bytes.NewBuffer(nil)
	writeString(payload, "MQTT")
	payload.Write([]byte{0x04, 0x02, 0x00, 0x1e})
	writeString(payload, m.cfg.ClientID)
	writeRemaining(pkt, payload.Len())
	pkt.Write(payload.Bytes())
	if _, err := conn.Write(pkt.Bytes()); err != nil {
		_ = conn.Close()
		return err
	}
	ack := make([]byte, 4)
	if _, err := io.ReadFull(conn, ack); err != nil {
		_ = conn.Close()
		return err
	}
	if ack[0] != 0x20 || ack[3] != 0x00 {
		_ = conn.Close()
		return fmt.Errorf("mqtt connack rejected: %x", ack)
	}
	m.setHealth(func(h *Health) {
		h.OK = true
		h.Detail = "connected; waiting for MQTT publishes"
		h.LastConnectedAt = time.Now().UTC().Format(time.RFC3339)
		h.LastError = ""
	})
	return nil
}
func (m *MQTT) Close(context.Context) error {
	if m.conn != nil {
		m.setHealth(func(h *Health) {
			h.OK = false
			h.Detail = "closed"
			h.LastDisconnected = time.Now().UTC().Format(time.RFC3339)
		})
		return m.conn.Close()
	}
	return nil
}
func (m *MQTT) Subscribe(ctx context.Context, handler PacketHandler) error {
	if m.conn == nil {
		return fmt.Errorf("not connected")
	}
	pkt := bytes.NewBuffer(nil)
	payload := bytes.NewBuffer(nil)
	_ = binary.Write(payload, binary.BigEndian, uint16(1))
	writeString(payload, m.cfg.Topic)
	payload.WriteByte(0)
	pkt.WriteByte(0x82)
	writeRemaining(pkt, payload.Len())
	pkt.Write(payload.Bytes())
	if _, err := m.conn.Write(pkt.Bytes()); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		header := make([]byte, 1)
		if _, err := io.ReadFull(m.conn, header); err != nil {
			m.markReadFailure(err)
			return err
		}
		remaining, err := readRemaining(m.conn)
		if err != nil {
			m.markReadFailure(err)
			return err
		}
		body := make([]byte, remaining)
		if _, err := io.ReadFull(m.conn, body); err != nil {
			m.markReadFailure(err)
			return err
		}
		typeNibble := header[0] >> 4
		if typeNibble != 3 {
			continue
		}
		topic, publishPayload, err := parsePublish(body)
		if err != nil {
			m.setHealth(func(h *Health) {
				h.PacketsDropped++
				h.LastError = err.Error()
				h.Detail = "publish parse failed"
			})
			m.bus.Publish(events.Event{Type: "transport.error", Data: err.Error()})
			continue
		}
		if err := handler(topic, publishPayload); err != nil {
			m.setHealth(func(h *Health) {
				h.PacketsDropped++
				h.LastError = err.Error()
				h.Detail = "ingest handler failed"
			})
			m.bus.Publish(events.Event{Type: "transport.error", Data: err.Error()})
			continue
		}
		m.setHealth(func(h *Health) {
			h.PacketsRead++
			h.OK = true
			h.LastPacketAt = time.Now().UTC().Format(time.RFC3339)
			h.Detail = "connected and ingesting MQTT publishes"
		})
	}
}

func (m *MQTT) SendPacket(context.Context, []byte) error {
	return errors.New("mqtt publish is disabled in this milestone")
}
func (m *MQTT) FetchMetadata(context.Context) (map[string]any, error) {
	return nil, errors.New("metadata fetch not supported for mqtt")
}
func (m *MQTT) FetchNodes(context.Context) ([]map[string]any, error) {
	return nil, errors.New("node fetch not supported for mqtt")
}

func (m *MQTT) markReadFailure(err error) {
	m.setHealth(func(h *Health) {
		h.OK = false
		h.LastError = err.Error()
		h.Detail = "stream disconnected"
		h.LastDisconnected = time.Now().UTC().Format(time.RFC3339)
	})
	m.bus.Publish(events.Event{Type: "transport.error", Data: err.Error()})
}

func writeString(buf *bytes.Buffer, s string) {
	_ = binary.Write(buf, binary.BigEndian, uint16(len(s)))
	buf.WriteString(s)
}
func writeRemaining(buf *bytes.Buffer, n int) {
	for {
		d := byte(n % 128)
		n /= 128
		if n > 0 {
			d |= 128
		}
		buf.WriteByte(d)
		if n == 0 {
			break
		}
	}
}
func readRemaining(r io.Reader) (int, error) {
	mult := 1
	val := 0
	for i := 0; i < 4; i++ {
		var b [1]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, err
		}
		val += int(b[0]&127) * mult
		if b[0]&128 == 0 {
			return val, nil
		}
		mult *= 128
	}
	return 0, fmt.Errorf("bad remaining length")
}
func parsePublish(body []byte) (string, []byte, error) {
	if len(body) < 2 {
		return "", nil, fmt.Errorf("short publish")
	}
	ln := int(binary.BigEndian.Uint16(body[:2]))
	if len(body) < 2+ln {
		return "", nil, fmt.Errorf("short topic")
	}
	return string(body[2 : 2+ln]), body[2+ln:], nil
}

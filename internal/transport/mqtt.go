package transport

import (
	"bytes"
	"context"
	"encoding/binary"
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
	return &MQTT{cfg: cfg, log: log, bus: bus, health: Health{Name: cfg.Name, Type: cfg.Type, Detail: "disabled", Capabilities: cfg.Capabilities}}
}
func (m *MQTT) Name() string   { return m.cfg.Name }
func (m *MQTT) Health() Health { m.mu.Lock(); defer m.mu.Unlock(); return m.health }
func (m *MQTT) setHealth(ok bool, detail string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.health.OK = ok
	m.health.Detail = detail
}
func (m *MQTT) Connect(ctx context.Context) error {
	conn, err := dialWithTimeout(m.cfg.Endpoint)
	if err != nil {
		m.setHealth(false, err.Error())
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
		return err
	}
	ack := make([]byte, 4)
	if _, err := io.ReadFull(conn, ack); err != nil {
		return err
	}
	if ack[0] != 0x20 || ack[3] != 0x00 {
		return fmt.Errorf("mqtt connack rejected: %x", ack)
	}
	m.setHealth(true, "connected")
	return nil
}
func (m *MQTT) Close(context.Context) error {
	if m.conn != nil {
		return m.conn.Close()
	}
	return nil
}
func (m *MQTT) Subscribe(ctx context.Context, handler func(string, []byte) error) error {
	if m.conn == nil {
		return fmt.Errorf("not connected")
	}
	pkt := bytes.NewBuffer(nil)
	payload := bytes.NewBuffer(nil)
	binary.Write(payload, binary.BigEndian, uint16(1))
	writeString(payload, m.cfg.Topic)
	payload.WriteByte(0)
	pkt.WriteByte(0x82)
	writeRemaining(pkt, payload.Len())
	pkt.Write(payload.Bytes())
	if _, err := m.conn.Write(pkt.Bytes()); err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			header := make([]byte, 1)
			if _, err := io.ReadFull(m.conn, header); err != nil {
				m.setHealth(false, err.Error())
				return
			}
			remaining, err := readRemaining(m.conn)
			if err != nil {
				m.setHealth(false, err.Error())
				return
			}
			body := make([]byte, remaining)
			if _, err := io.ReadFull(m.conn, body); err != nil {
				m.setHealth(false, err.Error())
				return
			}
			typeNibble := header[0] >> 4
			if typeNibble != 3 {
				continue
			}
			topic, payload, err := parsePublish(body)
			if err != nil {
				m.setHealth(false, err.Error())
				return
			}
			if err := handler(topic, payload); err != nil {
				m.bus.Publish(events.Event{Type: "transport.error", Data: err.Error()})
			}
		}
	}()
	return nil
}
func writeString(buf *bytes.Buffer, s string) {
	binary.Write(buf, binary.BigEndian, uint16(len(s)))
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

var _ = time.Second

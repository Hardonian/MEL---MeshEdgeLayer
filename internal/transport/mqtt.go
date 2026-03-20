package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
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
	qos2   map[uint16]publishPacket
}

func NewMQTT(cfg config.TransportConfig, log *logging.Logger, bus *events.Bus) *MQTT {
	caps := capabilityDefaults(cfg, true, false, false, false, true, false, "supported", "real MQTT subscribe path with QoS acknowledgements, keepalive, and reconnect-aware timeout handling; MEL does not claim publish/config control in this milestone")
	state := StateConfiguredNotAttempted
	detail := "configured but not yet attempted"
	if !cfg.Enabled {
		state = StateDisabled
		detail = "disabled by config"
	}
	return &MQTT{
		cfg:    cfg,
		log:    log,
		bus:    bus,
		health: Health{Name: cfg.Name, Type: cfg.Type, Source: cfg.Endpoint, State: state, Detail: detail, Capabilities: caps},
		qos2:   map[uint16]publishPacket{},
	}
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

func (m *MQTT) ForceState(state, detail, lastError string) {
	m.setHealth(func(h *Health) {
		h.State = state
		h.Detail = detail
		h.LastError = lastError
		if state == StateFailed || state == StateRetrying {
			h.OK = false
			h.LastFailureAt = time.Now().UTC().Format(time.RFC3339)
		} else if state == StateIdle || state == StateLive || state == StateConnecting || state == StateConfigured {
			h.OK = true
		}
	})
}

func (m *MQTT) BeginFailureEpisode(err error) (string, uint64) {
	now := time.Now().UTC()
	var episodeID string
	var failureCount uint64
	m.setHealth(func(h *Health) {
		if h.EpisodeID == "" {
			h.EpisodeID = fmt.Sprintf("%s-%d", m.cfg.Name, now.UnixNano())
		}
		h.FailureCount++
		failureCount = h.FailureCount
		episodeID = h.EpisodeID
		h.LastFailureAt = now.Format(time.RFC3339)
		if err != nil {
			h.LastError = err.Error()
		}
	})
	return episodeID, failureCount
}

func (m *MQTT) CloseFailureEpisode() {
	m.setHealth(func(h *Health) {
		h.FailureCount = 0
		h.EpisodeID = ""
		h.LastFailureAt = ""
	})
}

func (m *MQTT) RecordObservationDrop(count uint64) {
	if count == 0 {
		return
	}
	m.setHealth(func(h *Health) {
		h.ObservationDrops += count
		h.LastObservationDropAt = time.Now().UTC().Format(time.RFC3339)
	})
}

func (m *MQTT) connectedState() string {
	h := m.Health()
	if h.TotalMessages > 0 {
		return StateLive
	}
	return StateIdle
}

func (m *MQTT) Connect(ctx context.Context) error {
	m.setHealth(func(h *Health) {
		h.ReconnectAttempts++
		h.Source = m.cfg.Endpoint
		h.State = StateConnecting
		h.Detail = "attempting MQTT connection"
		h.LastAttemptAt = time.Now().UTC().Format(time.RFC3339)
	})
	conn, err := dialWithTimeout(m.cfg.Endpoint)
	if err != nil {
		m.setHealth(func(h *Health) {
			h.OK = false
			h.State = StateRetrying
			h.Detail = "configured broker is offline"
			h.LastError = err.Error()
			h.ErrorCount++
			h.LastDisconnected = time.Now().UTC().Format(time.RFC3339)
			h.LastFailureAt = h.LastDisconnected
		})
		return err
	}
	m.mu.Lock()
	m.conn = conn
	m.mu.Unlock()
	if err := m.writePacket(buildConnectPacket(m.cfg)); err != nil {
		_ = conn.Close()
		return err
	}
	ack := make([]byte, 4)
	if err := m.readInto(ack); err != nil {
		_ = conn.Close()
		return err
	}
	if ack[0] != 0x20 || ack[3] != 0x00 {
		_ = conn.Close()
		return fmt.Errorf("mqtt connack rejected: %x", ack)
	}
	m.setHealth(func(h *Health) {
		h.OK = true
		h.State = StateIdle
		h.Detail = fmt.Sprintf("connected; waiting for subscribed publishes (qos=%d) to be stored", m.requestedQoS())
		h.LastConnectedAt = time.Now().UTC().Format(time.RFC3339)
		h.LastSuccessAt = h.LastConnectedAt
		h.LastHeartbeatAt = h.LastConnectedAt
		h.LastError = ""
		h.ConsecutiveTimeouts = 0
	})
	return nil
}

func (m *MQTT) Close(context.Context) error {
	m.mu.Lock()
	conn := m.conn
	m.conn = nil
	m.mu.Unlock()
	if conn == nil {
		return nil
	}
	m.setHealth(func(h *Health) {
		h.OK = false
		if h.State != StateDisabled {
			h.State = StateRetrying
			if h.TotalMessages > 0 {
				h.Detail = "broker disconnected; historical ingest remains available"
			} else {
				h.Detail = "broker disconnected; waiting to retry"
			}
		}
		h.LastDisconnected = time.Now().UTC().Format(time.RFC3339)
	})
	return conn.Close()
}

func (m *MQTT) Subscribe(ctx context.Context, handler PacketHandler) error {
	if m.conn == nil {
		return fmt.Errorf("not connected")
	}
	packetID := uint16(1)
	if err := m.writePacket(buildSubscribePacket(packetID, m.cfg.Topic, m.requestedQoS())); err != nil {
		m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, m.cfg.Topic, ReasonSubscribeFailure, nil, true, err.Error(), map[string]any{"endpoint": m.cfg.Endpoint, "phase": "write_subscribe"}))
		return err
	}
	if err := m.expectSubAck(packetID); err != nil {
		m.markReadFailure(err, ReasonSubscribeFailure)
		return err
	}
	pingDone := make(chan struct{})
	go m.keepAliveLoop(ctx, pingDone)
	defer close(pingDone)

	maxTimeouts := m.cfg.MaxTimeouts
	if maxTimeouts <= 0 {
		maxTimeouts = 3
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		header := make([]byte, 1)
		if err := m.readInto(header); err != nil {
			if isTimeout(err) {
				timeouts := m.noteTimeout("connected; waiting for broker heartbeat or publish")
				m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, m.cfg.Topic, ReasonTimeoutStall, nil, false, "mqtt stream is stalled waiting for broker activity", map[string]any{
					"endpoint":             m.cfg.Endpoint,
					"consecutive_timeouts": timeouts,
					"max_timeouts":         maxTimeouts,
				}))
				if timeouts >= uint64(maxTimeouts) {
					m.markReadFailure(fmt.Errorf("mqtt transport exceeded %d consecutive read timeouts", maxTimeouts), ReasonTimeoutFailure)
					return err
				}
				continue
			}
			m.markReadFailure(err, ReasonStreamFailure)
			return err
		}
		remaining, err := readRemaining(m.conn)
		if err != nil {
			if isTimeout(err) {
				timeouts := m.noteTimeout("connected; waiting for broker heartbeat or publish")
				m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, m.cfg.Topic, ReasonTimeoutStall, nil, false, "mqtt stream is stalled waiting for broker activity", map[string]any{
					"endpoint":             m.cfg.Endpoint,
					"consecutive_timeouts": timeouts,
					"max_timeouts":         maxTimeouts,
				}))
				continue
			}
			m.markReadFailure(err, ReasonStreamFailure)
			return err
		}
		body := make([]byte, remaining)
		if err := m.readInto(body); err != nil {
			if isTimeout(err) {
				timeouts := m.noteTimeout("connected; waiting for broker heartbeat or publish")
				m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, m.cfg.Topic, ReasonTimeoutStall, nil, false, "mqtt stream is stalled waiting for broker activity", map[string]any{
					"endpoint":             m.cfg.Endpoint,
					"consecutive_timeouts": timeouts,
					"max_timeouts":         maxTimeouts,
				}))
				continue
			}
			m.markReadFailure(err, ReasonStreamFailure)
			return err
		}
		typeNibble := header[0] >> 4
		switch typeNibble {
		case 3:
			publish, err := parsePublish(header[0], body)
			if err != nil {
				m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, m.cfg.Topic, ReasonMalformedPublish, body, false, err.Error(), map[string]any{"endpoint": m.cfg.Endpoint, "header": fmt.Sprintf("0x%02x", header[0])}))
				m.markDropWithState("publish parse failed", err)
				m.bus.Publish(events.Event{Type: "transport.error", Data: err.Error()})
				continue
			}
			if !mqttTopicMatches(m.cfg.Topic, publish.Topic) {
				m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, publish.Topic, ReasonTopicMismatch, publish.Payload, false, "publish topic did not match configured filter", map[string]any{"configured_topic": m.cfg.Topic, "endpoint": m.cfg.Endpoint}))
				m.markDropWithState("publish topic did not match configured filter", fmt.Errorf("unexpected topic %s", publish.Topic))
				continue
			}
			if len(publish.Payload) > MaxMessageSize {
				m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, publish.Topic, ReasonObservationDropped, publish.Payload, false, fmt.Sprintf("payload exceeds maximum allowed size of %d bytes", MaxMessageSize), map[string]any{
					"endpoint":     m.cfg.Endpoint,
					"packet_id":    publish.PacketID,
					"qos":          publish.QoS,
					"payload_size": len(publish.Payload),
					"max_size":     MaxMessageSize,
				}))
				m.markDropWithState("payload size exceeds MaxMessageSize; dropping", fmt.Errorf("payload size %d exceeds max %d", len(publish.Payload), MaxMessageSize))
				continue
			}
			if err := handler(publish.Topic, publish.Payload); err != nil {
				m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, publish.Topic, ReasonHandlerRejection, publish.Payload, true, err.Error(), map[string]any{
					"endpoint":  m.cfg.Endpoint,
					"packet_id": publish.PacketID,
					"qos":       publish.QoS,
					"final":     true,
				}))
				m.markDropWithState("ingest handler rejected publish", err)
				m.bus.Publish(events.Event{Type: "transport.error", Data: err.Error()})
				continue
			}
			if err := m.ackPublish(publish); err != nil {
				m.markReadFailure(err, ReasonStreamFailure)
				return err
			}
			m.setHealth(func(h *Health) {
				h.OK = true
				if h.TotalMessages == 0 {
					h.State = StateIdle
					h.Detail = "publish received; waiting for database confirmation"
				}
				h.LastHeartbeatAt = time.Now().UTC().Format(time.RFC3339)
				h.ConsecutiveTimeouts = 0
				h.LastError = ""
			})
		case 13:
			m.recordHeartbeat("broker heartbeat received")
		case 9:
			m.recordHeartbeat("broker subscription acknowledged")
		case 6:
			if err := m.completeQoS2(body); err != nil {
				m.markReadFailure(err, ReasonStreamFailure)
				return err
			}
		default:
			m.recordHeartbeat(fmt.Sprintf("mqtt control packet type %d received", typeNibble))
		}
	}
}

func (m *MQTT) SendPacket(context.Context, []byte) error {
	err := errors.New("mqtt publish is disabled in this milestone")
	m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, m.cfg.Topic, ReasonRejectedPublish, nil, true, err.Error(), map[string]any{"endpoint": m.cfg.Endpoint, "qos": m.requestedQoS()}))
	return err
}
func (m *MQTT) FetchMetadata(context.Context) (map[string]any, error) {
	err := errors.New("metadata fetch not supported for mqtt")
	m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, m.cfg.Topic, ReasonUnsupportedControlPath, nil, false, err.Error(), map[string]any{"operation": "fetch_metadata", "endpoint": m.cfg.Endpoint}))
	return nil, err
}
func (m *MQTT) FetchNodes(context.Context) ([]map[string]any, error) {
	err := errors.New("node fetch not supported for mqtt")
	m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, m.cfg.Topic, ReasonUnsupportedControlPath, nil, false, err.Error(), map[string]any{"operation": "fetch_nodes", "endpoint": m.cfg.Endpoint}))
	return nil, err
}
func (m *MQTT) MarkIngest(at time.Time) {
	m.setHealth(func(h *Health) {
		h.OK = true
		h.State = StateLive
		h.TotalMessages++
		h.LastIngestAt = at.UTC().Format(time.RFC3339)
		h.LastHeartbeatAt = h.LastIngestAt
		h.ConsecutiveTimeouts = 0
		h.LastError = ""
		h.Detail = "live ingest confirmed by SQLite writes"
	})
	m.CloseFailureEpisode()
}
func (m *MQTT) MarkDrop(reason string) { m.markDropWithState(reason, nil) }

func (m *MQTT) markReadFailure(err error, reason string) {
	episodeID, failureCount := m.BeginFailureEpisode(err)
	state := StateRetrying
	if reason == ReasonTimeoutFailure {
		state = StateFailed
	}
	m.setHealth(func(h *Health) {
		h.OK = false
		h.State = state
		h.LastError = err.Error()
		h.Detail = "stream disconnected; waiting to retry"
		h.ErrorCount++
		h.LastDisconnected = time.Now().UTC().Format(time.RFC3339)
	})
	m.bus.Publish(events.Event{Type: "transport.error", Data: err.Error()})
	m.publishObservation(NewObservation(m.cfg.Name, m.cfg.Type, m.cfg.Topic, reason, nil, false, "stream disconnected; waiting to retry", map[string]any{
		"error":                err.Error(),
		"endpoint":             m.cfg.Endpoint,
		"consecutive_timeouts": m.Health().ConsecutiveTimeouts,
		"failure_count":        failureCount,
		"episode_id":           episodeID,
		"last_error":           err.Error(),
	}))
}

func (m *MQTT) markDropWithState(reason string, err error) {
	state := m.connectedState()
	m.setHealth(func(h *Health) {
		h.PacketsDropped++
		h.State = state
		h.Detail = reason
		if err != nil {
			h.LastError = err.Error()
			h.ErrorCount++
		}
	})
}

func (m *MQTT) noteTimeout(detail string) uint64 {
	var timeouts uint64
	m.setHealth(func(h *Health) {
		h.OK = true
		h.ConsecutiveTimeouts++
		timeouts = h.ConsecutiveTimeouts
		if h.TotalMessages > 0 {
			h.State = StateLive
		} else {
			h.State = StateIdle
		}
		h.Detail = detail
	})
	return timeouts
}

func (m *MQTT) recordHeartbeat(detail string) {
	m.setHealth(func(h *Health) {
		h.OK = true
		if h.TotalMessages > 0 {
			h.State = StateLive
		} else {
			h.State = StateIdle
		}
		h.Detail = detail
		h.LastHeartbeatAt = time.Now().UTC().Format(time.RFC3339)
		h.ConsecutiveTimeouts = 0
		h.LastError = ""
	})
}

func (m *MQTT) publishObservation(obs Observation) {
	if m.bus == nil {
		return
	}
	if obs.TransportName == "" {
		obs.TransportName = m.cfg.Name
	}
	if obs.TransportType == "" {
		obs.TransportType = m.cfg.Type
	}
	if obs.Topic == "" {
		obs.Topic = m.cfg.Topic
	}
	_, dropped := m.bus.Publish(events.Event{Type: "transport.observation", Data: obs})
	if dropped > 0 {
		m.RecordObservationDrop(uint64(dropped))
	}
}

func (m *MQTT) requestedQoS() byte {
	if m.cfg.MQTTQoS < 0 || m.cfg.MQTTQoS > 2 {
		return 1
	}
	return byte(m.cfg.MQTTQoS)
}

func (m *MQTT) keepAliveLoop(ctx context.Context, done <-chan struct{}) {
	interval := time.Duration(m.cfg.MQTTKeepAliveSec) * time.Second / 2
	if interval <= 0 {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if err := m.writePacket([]byte{0xC0, 0x00}); err != nil {
				m.markReadFailure(err, ReasonStreamFailure)
				return
			}
			m.setHealth(func(h *Health) {
				h.OK = true
				if h.TotalMessages > 0 {
					h.State = StateLive
				} else {
					h.State = StateIdle
				}
				h.Detail = "keepalive ping sent; awaiting broker response"
			})
		}
	}
}

func (m *MQTT) expectSubAck(packetID uint16) error {
	header := make([]byte, 1)
	if err := m.readInto(header); err != nil {
		return err
	}
	if header[0]>>4 != 9 {
		return fmt.Errorf("expected suback, got packet type %d", header[0]>>4)
	}
	remaining, err := readRemaining(m.conn)
	if err != nil {
		return err
	}
	body := make([]byte, remaining)
	if err := m.readInto(body); err != nil {
		return err
	}
	if len(body) < 3 {
		return fmt.Errorf("short suback")
	}
	if binary.BigEndian.Uint16(body[:2]) != packetID {
		return fmt.Errorf("unexpected suback packet id %d", binary.BigEndian.Uint16(body[:2]))
	}
	if body[2] == 0x80 {
		return fmt.Errorf("broker rejected subscribe to %s", m.cfg.Topic)
	}
	m.recordHeartbeat("broker subscription acknowledged")
	return nil
}

func (m *MQTT) ackPublish(p publishPacket) error {
	switch p.QoS {
	case 0:
		return nil
	case 1:
		buf := bytes.NewBuffer([]byte{0x40, 0x02})
		_ = binary.Write(buf, binary.BigEndian, p.PacketID)
		return m.writePacket(buf.Bytes())
	case 2:
		m.mu.Lock()
		m.qos2[p.PacketID] = p
		m.mu.Unlock()
		buf := bytes.NewBuffer([]byte{0x50, 0x02})
		_ = binary.Write(buf, binary.BigEndian, p.PacketID)
		return m.writePacket(buf.Bytes())
	default:
		return fmt.Errorf("unsupported publish qos %d", p.QoS)
	}
}

func (m *MQTT) completeQoS2(body []byte) error {
	if len(body) != 2 {
		return fmt.Errorf("invalid pubrel length %d", len(body))
	}
	packetID := binary.BigEndian.Uint16(body)
	m.mu.Lock()
	_, ok := m.qos2[packetID]
	delete(m.qos2, packetID)
	m.mu.Unlock()
	buf := bytes.NewBuffer([]byte{0x70, 0x02})
	_ = binary.Write(buf, binary.BigEndian, packetID)
	if err := m.writePacket(buf.Bytes()); err != nil {
		return err
	}
	if ok {
		m.recordHeartbeat("broker qos2 delivery completed")
		return nil
	}
	m.recordHeartbeat("broker qos2 release received without cached publish; pubcomp sent to clear session")
	return nil
}

func (m *MQTT) writePacket(pkt []byte) error {
	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()
	if conn == nil {
		return net.ErrClosed
	}
	if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout(m.cfg))); err != nil {
		return err
	}
	_, err := conn.Write(pkt)
	return err
}

func (m *MQTT) readInto(buf []byte) error {
	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()
	if conn == nil {
		return net.ErrClosed
	}
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout(m.cfg))); err != nil {
		return err
	}
	_, err := io.ReadFull(conn, buf)
	return err
}

type publishPacket struct {
	Topic    string
	Payload  []byte
	PacketID uint16
	QoS      byte
}

func mqttTopicMatches(filter, topic string) bool {
	if filter == topic {
		return true
	}
	fp := strings.Split(filter, "/")
	tp := strings.Split(topic, "/")
	for i := 0; i < len(fp); i++ {
		if i >= len(tp) {
			return fp[i] == "#" && i == len(fp)-1
		}
		switch fp[i] {
		case "#":
			return i == len(fp)-1
		case "+":
			continue
		default:
			if fp[i] != tp[i] {
				return false
			}
		}
	}
	return len(fp) == len(tp)
}

func buildConnectPacket(cfg config.TransportConfig) []byte {
	pkt := bytes.NewBuffer(nil)
	pkt.WriteByte(0x10)
	payload := bytes.NewBuffer(nil)
	writeString(payload, "MQTT")
	connectFlags := byte(0)
	if cfg.MQTTCleanSession {
		connectFlags |= 0x02
	}
	if cfg.Username != "" {
		connectFlags |= 0x80
	}
	if cfg.Password != "" {
		connectFlags |= 0x40
	}
	payload.Write([]byte{0x04, connectFlags})
	_ = binary.Write(payload, binary.BigEndian, uint16(cfg.MQTTKeepAliveSec))
	writeString(payload, cfg.ClientID)
	if cfg.Username != "" {
		writeString(payload, cfg.Username)
	}
	if cfg.Password != "" {
		writeString(payload, cfg.Password)
	}
	writeRemaining(pkt, payload.Len())
	pkt.Write(payload.Bytes())
	return pkt.Bytes()
}

func buildSubscribePacket(packetID uint16, topic string, qos byte) []byte {
	pkt := bytes.NewBuffer(nil)
	payload := bytes.NewBuffer(nil)
	_ = binary.Write(payload, binary.BigEndian, packetID)
	writeString(payload, topic)
	payload.WriteByte(qos)
	pkt.WriteByte(0x82)
	writeRemaining(pkt, payload.Len())
	pkt.Write(payload.Bytes())
	return pkt.Bytes()
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
func parsePublish(header byte, body []byte) (publishPacket, error) {
	if len(body) < 2 {
		return publishPacket{}, fmt.Errorf("short publish")
	}
	ln := int(binary.BigEndian.Uint16(body[:2]))
	if len(body) < 2+ln {
		return publishPacket{}, fmt.Errorf("short topic")
	}
	out := publishPacket{Topic: string(body[2 : 2+ln]), QoS: (header >> 1) & 0x03}
	offset := 2 + ln
	if out.QoS > 0 {
		if len(body) < offset+2 {
			return publishPacket{}, fmt.Errorf("missing packet id")
		}
		out.PacketID = binary.BigEndian.Uint16(body[offset : offset+2])
		offset += 2
	}
	out.Payload = body[offset:]
	return out, nil
}

func readTimeout(cfg config.TransportConfig) time.Duration {
	if cfg.ReadTimeoutSec <= 0 {
		return 15 * time.Second
	}
	return time.Duration(cfg.ReadTimeoutSec) * time.Second
}

func writeTimeout(cfg config.TransportConfig) time.Duration {
	if cfg.WriteTimeoutSec <= 0 {
		return 5 * time.Second
	}
	return time.Duration(cfg.WriteTimeoutSec) * time.Second
}

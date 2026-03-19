package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
)

func TestMQTTSubscribe(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	cfg := config.TransportConfig{Name: "test", Type: "mqtt", Endpoint: ln.Addr().String(), Topic: "msh/test", ClientID: "mel-test", MQTTQoS: 1, MQTTKeepAliveSec: 30, ReadTimeoutSec: 1, WriteTimeoutSec: 1, MaxTimeouts: 3}
	m := NewMQTT(cfg, logging.New("debug", true), events.New())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	acks := make(chan []byte, 2)
	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		buf := make([]byte, 256)
		_, _ = conn.Read(buf) // CONNECT
		_, _ = conn.Write([]byte{0x20, 0x02, 0x00, 0x00})
		_, _ = conn.Read(buf) // SUBSCRIBE
		_, _ = conn.Write([]byte{0x90, 0x03, 0x00, 0x01, 0x01})
		payload := []byte{0x01, 0x02}
		body := bytes.NewBuffer(nil)
		_ = binary.Write(body, binary.BigEndian, uint16(len(cfg.Topic)))
		body.WriteString(cfg.Topic)
		_ = binary.Write(body, binary.BigEndian, uint16(7))
		body.Write(payload)
		pkt := bytes.NewBuffer([]byte{0x32, byte(body.Len())})
		pkt.Write(body.Bytes())
		_, _ = conn.Write(pkt.Bytes())
		ack := make([]byte, 4)
		_, _ = io.ReadFull(conn, ack)
		acks <- ack
	}()
	if err := m.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	got := make(chan []byte, 1)
	go func() {
		_ = m.Subscribe(ctx, func(topic string, payload []byte) error {
			m.MarkIngest(time.Now())
			got <- payload
			cancel()
			return nil
		})
	}()
	select {
	case p := <-got:
		if len(p) != 2 {
			t.Fatal("payload missing")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	select {
	case ack := <-acks:
		if !bytes.Equal(ack, []byte{0x40, 0x02, 0x00, 0x07}) {
			t.Fatalf("unexpected puback: %x", ack)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected puback")
	}
}

func TestMQTTSubscribeDetectsTimeouts(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	cfg := config.TransportConfig{Name: "test", Type: "mqtt", Endpoint: ln.Addr().String(), Topic: "msh/test", ClientID: "mel-test", MQTTQoS: 1, MQTTKeepAliveSec: 1, ReadTimeoutSec: 1, WriteTimeoutSec: 1, MaxTimeouts: 2}
	m := NewMQTT(cfg, logging.New("debug", true), events.New())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		buf := make([]byte, 256)
		_, _ = conn.Read(buf) // CONNECT
		_, _ = conn.Write([]byte{0x20, 0x02, 0x00, 0x00})
		_, _ = conn.Read(buf) // SUBSCRIBE
		_, _ = conn.Write([]byte{0x90, 0x03, 0x00, 0x01, 0x01})
		time.Sleep(5 * time.Second)
	}()
	if err := m.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	err = m.Subscribe(ctx, func(string, []byte) error { return nil })
	if err == nil {
		t.Fatal("expected subscribe error after repeated timeouts")
	}
	h := m.Health()
	if h.State != StateError || h.ConsecutiveTimeouts < 2 {
		t.Fatalf("expected timeout-driven error state, got %+v", h)
	}
}

package transport

import (
	"bytes"
	"context"
	"encoding/binary"
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
	cfg := config.TransportConfig{Name: "test", Type: "mqtt", Endpoint: ln.Addr().String(), Topic: "msh/test", ClientID: "mel-test"}
	m := NewMQTT(cfg, logging.New(), events.New())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		buf := make([]byte, 128)
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte{0x20, 0x02, 0x00, 0x00})
		_, _ = conn.Read(buf)
		payload := []byte{0x01, 0x02}
		body := bytes.NewBuffer(nil)
		_ = binary.Write(body, binary.BigEndian, uint16(len(cfg.Topic)))
		body.WriteString(cfg.Topic)
		body.Write(payload)
		pkt := bytes.NewBuffer([]byte{0x30, byte(body.Len())})
		pkt.Write(body.Bytes())
		_, _ = conn.Write(pkt.Bytes())
		time.Sleep(200 * time.Millisecond)
		_ = conn.Close()
	}()
	if err := m.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	got := make(chan []byte, 1)
	go func() {
		_ = m.Subscribe(ctx, func(topic string, payload []byte) error { got <- payload; cancel(); return nil })
	}()
	select {
	case p := <-got:
		if len(p) != 2 {
			t.Fatal("payload missing")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
	"net"
	"testing"
	"time"
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
		conn.Read(buf)
		conn.Write([]byte{0x20, 0x02, 0x00, 0x00})
		conn.Read(buf)
		payload := []byte{0x01, 0x02}
		body := bytes.NewBuffer(nil)
		binary.Write(body, binary.BigEndian, uint16(len(cfg.Topic)))
		body.WriteString(cfg.Topic)
		body.Write(payload)
		pkt := bytes.NewBuffer([]byte{0x30, byte(body.Len())})
		pkt.Write(body.Bytes())
		conn.Write(pkt.Bytes())
		time.Sleep(500 * time.Millisecond)
	}()
	if err := m.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	got := make(chan []byte, 1)
	if err := m.Subscribe(ctx, func(topic string, payload []byte) error { got <- payload; return nil }); err != nil {
		t.Fatal(err)
	}
	select {
	case p := <-got:
		if len(p) != 2 {
			t.Fatal("payload missing")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

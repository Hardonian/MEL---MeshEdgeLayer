package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
)

type rwc struct {
	io.Reader
	io.Writer
}

func (r *rwc) Close() error { return nil }

func TestReadDirectFrame(t *testing.T) {
	payload := []byte{0x0a, 0x01, 0x01}
	buf := bytes.NewBuffer([]byte{0x00, directStart1, directStart2, 0x00, byte(len(payload))})
	buf.Write(payload)
	got, err := readDirectFrame(bufio.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("unexpected payload %x", got)
	}
}

func TestDirectSubscribe(t *testing.T) {
	packet := testPacket()
	fromRadio := append([]byte{directStart1, directStart2, 0x00, byte(len(packet))}, packet...)
	reader := bytes.NewBuffer(fromRadio)
	transport := NewDirect(config.TransportConfig{Name: "direct", Type: "tcp", Endpoint: "127.0.0.1:4403"}, logging.New(), events.New())
	transport.dial = func(context.Context, config.TransportConfig) (io.ReadWriteCloser, error) {
		return &rwc{Reader: reader, Writer: io.Discard}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	got := make(chan []byte, 1)
	go func() {
		_ = transport.Subscribe(ctx, func(topic string, payload []byte) error { got <- payload; cancel(); return nil })
	}()
	select {
	case payload := <-got:
		if len(payload) == 0 {
			t.Fatal("expected wrapped envelope payload")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for direct payload")
	}
	h := transport.Health()
	if h.PacketsRead != 1 || !h.OK {
		t.Fatalf("unexpected health: %+v", h)
	}
}

func testPacket() []byte {
	user := protoMsg(protoBytes(1, []byte("!abcd")), protoBytes(2, []byte("Node")), protoBytes(3, []byte("N")))
	data := protoMsg(protoVarint(1, 4), protoBytes(2, user))
	meshPacket := protoMsg(protoFixed32(1, 42), protoFixed32(2, 255), protoBytes(4, data), protoFixed32(6, 99), protoFixed32(7, uint32(time.Now().Unix())), protoVarint(9, 3))
	return protoMsg(protoBytes(1, meshPacket))
}
func protoMsg(parts ...[]byte) []byte { return bytes.Join(parts, nil) }
func protoTag(field int, wt int) []byte {
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, uint64(field<<3|wt))
	return b[:n]
}
func protoVarint(field int, v uint64) []byte {
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, v)
	return append(protoTag(field, 0), b[:n]...)
}
func protoFixed32(field int, v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return append(protoTag(field, 5), b...)
}
func protoBytes(field int, v []byte) []byte {
	ln := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(ln, uint64(len(v)))
	out := append(protoTag(field, 2), ln[:n]...)
	return append(out, v...)
}

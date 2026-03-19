package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
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

type timeoutReader struct {
	buf bytes.Buffer
}

func (t *timeoutReader) Read(p []byte) (int, error) {
	if t.buf.Len() == 0 {
		return 0, timeoutErr{}
	}
	return t.buf.Read(p)
}
func (t *timeoutReader) Write(p []byte) (int, error) { return len(p), nil }
func (t *timeoutReader) Close() error                { return nil }
func (t *timeoutReader) SetReadDeadline(time.Time) error {
	return nil
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestReadDirectFrame(t *testing.T) {
	payload := []byte{0x0a, 0x01, 0x01}
	buf := bytes.NewBuffer([]byte{0x00, directStart1, directStart2, 0x00, byte(len(payload))})
	buf.Write(payload)
	got, err := readDirectFrame(context.Background(), &rwc{Reader: buf, Writer: io.Discard}, bufio.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("unexpected payload %x", got)
	}
}

func TestReadDirectFrameInvalidLength(t *testing.T) {
	buf := bytes.NewBuffer([]byte{directStart1, directStart2, 0x02, 0x01})
	_, err := readDirectFrame(context.Background(), &rwc{Reader: buf, Writer: io.Discard}, bufio.NewReader(buf))
	if !errors.Is(err, errDirectInvalidFrame) {
		t.Fatalf("expected invalid frame error, got %v", err)
	}
}

func TestDirectSubscribe(t *testing.T) {
	packet := testPacket()
	fromRadio := append([]byte{directStart1, directStart2, 0x00, byte(len(packet))}, packet...)
	reader := bytes.NewBuffer(fromRadio)
	transport := NewDirect(config.TransportConfig{Name: "direct", Type: "tcp", Enabled: true, Endpoint: "127.0.0.1:4403"}, logging.New(), events.New())
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
	if h.PacketsRead != 1 || !h.OK || h.State != StateIngesting {
		t.Fatalf("unexpected health: %+v", h)
	}
}

func TestDirectSubscribeInvalidFrameContinues(t *testing.T) {
	packet := testPacket()
	invalid := []byte{directStart1, directStart2, 0x02, 0x01}
	valid := append([]byte{directStart1, directStart2, 0x00, byte(len(packet))}, packet...)
	reader := &timeoutReader{}
	reader.buf.Write(invalid)
	reader.buf.Write(valid)
	transport := NewDirect(config.TransportConfig{Name: "direct", Type: "tcp", Enabled: true, Endpoint: "127.0.0.1:4403"}, logging.New(), events.New())
	transport.dial = func(context.Context, config.TransportConfig) (io.ReadWriteCloser, error) {
		return reader, nil
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
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for payload after malformed frame")
	}
	h := transport.Health()
	if h.PacketsRead != 1 || h.PacketsDropped == 0 || h.State != StateIngesting {
		t.Fatalf("unexpected health after malformed frame recovery: %+v", h)
	}
}

func TestDirectSubscribeCancelOnIdleConnection(t *testing.T) {
	transport := NewDirect(config.TransportConfig{Name: "direct", Type: "tcp", Enabled: true, Endpoint: "127.0.0.1:4403"}, logging.New(), events.New())
	transport.dial = func(context.Context, config.TransportConfig) (io.ReadWriteCloser, error) {
		return &timeoutReader{}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := transport.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- transport.Subscribe(ctx, func(string, []byte) error { return nil }) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected clean shutdown, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("subscribe did not exit after context cancellation")
	}
	h := transport.Health()
	if h.State != StateConnectedNoData {
		t.Fatalf("expected idle state to remain truthful, got %+v", h)
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

var _ net.Error = timeoutErr{}

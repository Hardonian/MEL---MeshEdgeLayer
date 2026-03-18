package meshtastic

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func tag(field int, wt int) []byte {
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, uint64(field<<3|wt))
	return b[:n]
}
func fieldVarint(field int, v uint64) []byte {
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, v)
	return append(tag(field, 0), b[:n]...)
}
func fieldFixed32(field int, v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return append(tag(field, 5), b...)
}
func fieldBytes(field int, v []byte) []byte {
	ln := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(ln, uint64(len(v)))
	out := append(tag(field, 2), ln[:n]...)
	return append(out, v...)
}
func msg(parts ...[]byte) []byte { return bytes.Join(parts, nil) }
func TestParseEnvelope(t *testing.T) {
	user := msg(fieldBytes(1, []byte("!abcd")), fieldBytes(2, []byte("Node")), fieldBytes(3, []byte("N")))
	data := msg(fieldVarint(1, 4), fieldBytes(2, user))
	packet := msg(fieldFixed32(1, 42), fieldFixed32(2, 255), fieldBytes(4, data), fieldFixed32(6, 99), fieldFixed32(7, uint32(time.Now().Unix())), fieldVarint(9, 3))
	env := msg(fieldBytes(1, packet), fieldBytes(2, []byte("mel-test")), fieldBytes(3, []byte("!gw")))
	parsed, err := ParseEnvelope(env)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Packet.From != 42 || parsed.Packet.LongName != "Node" {
		t.Fatalf("unexpected parse: %+v", parsed.Packet)
	}
}

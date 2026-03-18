package meshtastic

import (
	"testing"
	"time"
)

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

func TestParseDirectFromRadio(t *testing.T) {
	user := msg(fieldBytes(1, []byte("!abcd")), fieldBytes(2, []byte("Node")), fieldBytes(3, []byte("N")))
	data := msg(fieldVarint(1, 4), fieldBytes(2, user))
	packet := msg(fieldFixed32(1, 42), fieldFixed32(2, 255), fieldBytes(4, data), fieldFixed32(6, 99), fieldFixed32(7, uint32(time.Now().Unix())), fieldVarint(9, 3))
	fromRadio := msg(fieldBytes(1, packet))
	parsed, err := ParseDirectFromRadio(fromRadio)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Packet.From != 42 || parsed.PacketRaw == nil {
		t.Fatalf("unexpected direct parse: %+v", parsed)
	}
}

func TestDedupeHashUsesPacketBytes(t *testing.T) {
	packet := msg(fieldFixed32(1, 42), fieldFixed32(2, 255), fieldFixed32(6, 99), fieldFixed32(7, 1))
	mqttEnv, err := ParseEnvelope(msg(fieldBytes(1, packet), fieldBytes(2, []byte("chan"))))
	if err != nil {
		t.Fatal(err)
	}
	directEnv, err := ParseDirectFromRadio(msg(fieldBytes(1, packet)))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := DedupeHash(mqttEnv), DedupeHash(directEnv); got != want {
		t.Fatalf("expected matching hashes, got %s want %s", got, want)
	}
}

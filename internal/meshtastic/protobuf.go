package meshtastic

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
)

type Envelope struct {
	ChannelID, GatewayID string
	Packet               Packet
	RawHex               string
}
type Packet struct {
	From, To, ID, RXTime, HopLimit, RelayNode uint32
	RXSNR                                     float32
	RXRSSI                                    int32
	PortNum                                   int32
	Payload                                   []byte
	PayloadText                               string
	NodeID                                    string
	LongName                                  string
	ShortName                                 string
	Lat                                       *float64
	Lon                                       *float64
	Altitude                                  int32
}

func ParseEnvelope(raw []byte) (Envelope, error) {
	var env Envelope
	env.RawHex = hex.EncodeToString(raw)
	fields, err := parse(raw)
	if err != nil {
		return env, err
	}
	if v, ok := fields[2]; ok {
		env.ChannelID = string(v[0].Bytes)
	}
	if v, ok := fields[3]; ok {
		env.GatewayID = string(v[0].Bytes)
	}
	if v, ok := fields[1]; ok {
		pkt, err := parsePacket(v[0].Bytes)
		if err != nil {
			return env, err
		}
		env.Packet = pkt
	}
	return env, nil
}

func parsePacket(raw []byte) (Packet, error) {
	var p Packet
	fields, err := parse(raw)
	if err != nil {
		return p, err
	}
	p.From = fields[1][0].Fixed32
	p.To = fields[2][0].Fixed32
	p.ID = fields[6][0].Fixed32
	p.RXTime = fields[7][0].Fixed32
	if len(fields[8]) > 0 {
		p.RXSNR = math.Float32frombits(fields[8][0].Fixed32)
	}
	if len(fields[9]) > 0 {
		p.HopLimit = uint32(fields[9][0].Varint)
	}
	if len(fields[12]) > 0 {
		p.RXRSSI = int32(fields[12][0].Varint)
	}
	if len(fields[19]) > 0 {
		p.RelayNode = uint32(fields[19][0].Varint)
	}
	if len(fields[4]) > 0 {
		dataFields, err := parse(fields[4][0].Bytes)
		if err != nil {
			return p, err
		}
		if len(dataFields[1]) > 0 {
			p.PortNum = int32(dataFields[1][0].Varint)
		}
		if len(dataFields[2]) > 0 {
			p.Payload = dataFields[2][0].Bytes
			p.PayloadText = string(dataFields[2][0].Bytes)
		}
		switch p.PortNum {
		case 1:
			p.PayloadText = string(p.Payload)
		case 3:
			applyPosition(&p, p.Payload)
		case 4:
			applyUser(&p, p.Payload)
		}
	}
	return p, nil
}

type wire struct {
	Varint  uint64
	Fixed32 uint32
	Bytes   []byte
	Type    int
}

func parse(raw []byte) (map[int][]wire, error) {
	out := map[int][]wire{}
	for i := 0; i < len(raw); {
		tag, n := binary.Uvarint(raw[i:])
		if n <= 0 {
			return nil, errors.New("invalid varint tag")
		}
		i += n
		fieldNum := int(tag >> 3)
		wireType := int(tag & 0x7)
		w := wire{Type: wireType}
		switch wireType {
		case 0:
			v, n := binary.Uvarint(raw[i:])
			if n <= 0 {
				return nil, errors.New("invalid varint value")
			}
			i += n
			w.Varint = v
		case 1:
			if i+8 > len(raw) {
				return nil, errors.New("short fixed64")
			}
			i += 8
		case 2:
			ln, n := binary.Uvarint(raw[i:])
			if n <= 0 {
				return nil, errors.New("invalid len")
			}
			i += n
			if i+int(ln) > len(raw) {
				return nil, errors.New("short bytes")
			}
			w.Bytes = append([]byte(nil), raw[i:i+int(ln)]...)
			i += int(ln)
		case 5:
			if i+4 > len(raw) {
				return nil, errors.New("short fixed32")
			}
			w.Fixed32 = binary.LittleEndian.Uint32(raw[i : i+4])
			i += 4
		default:
			return nil, fmt.Errorf("unsupported wire type %d", wireType)
		}
		out[fieldNum] = append(out[fieldNum], w)
	}
	return out, nil
}

func applyUser(p *Packet, raw []byte) {
	fields, err := parse(raw)
	if err != nil {
		return
	}
	if len(fields[1]) > 0 {
		p.NodeID = string(fields[1][0].Bytes)
	}
	if len(fields[2]) > 0 {
		p.LongName = string(fields[2][0].Bytes)
	}
	if len(fields[3]) > 0 {
		p.ShortName = string(fields[3][0].Bytes)
	}
}
func applyPosition(p *Packet, raw []byte) {
	fields, err := parse(raw)
	if err != nil {
		return
	}
	if len(fields[1]) > 0 {
		v := float64(int32(fields[1][0].Fixed32)) / 1e7
		p.Lat = &v
	}
	if len(fields[2]) > 0 {
		v := float64(int32(fields[2][0].Fixed32)) / 1e7
		p.Lon = &v
	}
	if len(fields[3]) > 0 {
		p.Altitude = int32(fields[3][0].Varint)
	}
}

func RedactCoord(v *float64) float64 {
	if v == nil {
		return 0
	}
	return math.Round(*v*100) / 100
}

func TopicEncrypted(topic string) bool {
	topic = strings.ToLower(topic)
	return strings.Contains(topic, "/e/") || strings.Contains(topic, "encrypted")
}

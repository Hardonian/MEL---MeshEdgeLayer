package meshtastic

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
)

const (
	// MaxMessageSize is the maximum allowed protobuf message size (64KB).
	// Messages larger than this are rejected to prevent memory exhaustion.
	MaxMessageSize = 64 * 1024

	// MaxFields is the maximum number of fields allowed per message.
	// This prevents resource exhaustion from malicious messages with excessive fields.
	MaxFields = 1000

	// MaxNestingDepth is the maximum nesting depth for nested protobuf messages.
	// This prevents stack overflow from malicious deeply nested messages.
	MaxNestingDepth = 10

	// MaxAllocBytes is the maximum bytes that can be allocated during a single parse operation.
	// This includes all byte slice allocations from length-delimited fields.
	MaxAllocBytes = 1 * 1024 * 1024 // 1MB
)

var (
	// ErrMessageTooLarge is returned when a message exceeds MaxMessageSize.
	ErrMessageTooLarge = errors.New("protobuf message exceeds maximum size")

	// ErrTooManyFields is returned when a message exceeds MaxFields.
	ErrTooManyFields = errors.New("protobuf message has too many fields")

	// ErrNestingTooDeep is returned when message nesting exceeds MaxNestingDepth.
	ErrNestingTooDeep = errors.New("protobuf message nesting exceeds maximum depth")

	// ErrAllocationExceeded is returned when total allocations exceed MaxAllocBytes.
	ErrAllocationExceeded = errors.New("protobuf parse allocation limit exceeded")
)

// ParseLimits defines security boundaries for protobuf parsing.
// All limits are defensive and prevent resource exhaustion from hostile input.
type ParseLimits struct {
	// MaxMessageSize is the maximum input size in bytes. Default: 64KB
	MaxMessageSize int

	// MaxFields is the maximum number of fields per message. Default: 1000
	MaxFields int

	// MaxNestingDepth is the maximum message nesting depth. Default: 10
	MaxNestingDepth int

	// MaxAllocBytes is the maximum total allocation during parsing. Default: 1MB
	MaxAllocBytes int
}

// DefaultParseLimits returns safe default parsing limits.
func DefaultParseLimits() ParseLimits {
	return ParseLimits{
		MaxMessageSize:  MaxMessageSize,
		MaxFields:       MaxFields,
		MaxNestingDepth: MaxNestingDepth,
		MaxAllocBytes:   MaxAllocBytes,
	}
}

// parseState tracks parsing progress and enforces limits.
type parseState struct {
	limits     ParseLimits
	depth      int
	fieldCount int
	allocBytes int
}

// newParseState creates a new parse state with the given limits.
func newParseState(limits ParseLimits) *parseState {
	return &parseState{
		limits:     limits,
		depth:      0,
		fieldCount: 0,
		allocBytes: 0,
	}
}

// checkMessageSize validates that the input size is within limits.
func (s *parseState) checkMessageSize(size int) error {
	if size > s.limits.MaxMessageSize {
		return fmt.Errorf("%w: %d bytes (max %d)", ErrMessageTooLarge, size, s.limits.MaxMessageSize)
	}
	return nil
}

// enter increments the nesting depth and checks the limit.
func (s *parseState) enter() error {
	s.depth++
	if s.depth > s.limits.MaxNestingDepth {
		return fmt.Errorf("%w: %d (max %d)", ErrNestingTooDeep, s.depth, s.limits.MaxNestingDepth)
	}
	return nil
}

// exit decrements the nesting depth.
func (s *parseState) exit() {
	s.depth--
}

// addField increments the field count and checks the limit.
func (s *parseState) addField() error {
	s.fieldCount++
	if s.fieldCount > s.limits.MaxFields {
		return fmt.Errorf("%w: %d (max %d)", ErrTooManyFields, s.fieldCount, s.limits.MaxFields)
	}
	return nil
}

// alloc tracks a byte slice allocation and checks the limit.
func (s *parseState) alloc(size int) error {
	s.allocBytes += size
	if s.allocBytes > s.limits.MaxAllocBytes {
		return fmt.Errorf("%w: %d bytes (max %d)", ErrAllocationExceeded, s.allocBytes, s.limits.MaxAllocBytes)
	}
	return nil
}

type Envelope struct {
	ChannelID, GatewayID string
	Packet               Packet
	PacketRaw            []byte
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

func (p Packet) PayloadHex() string {
	return hex.EncodeToString(p.Payload)
}

func MessageType(packet Packet) string {
	switch packet.PortNum {
	case 1:
		return "text"
	case 3:
		return "position"
	case 4:
		return "node_info"
	case 67:
		return "telemetry"
	default:
		return "unknown"
	}
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
		env.PacketRaw = append([]byte(nil), v[0].Bytes...)
		pkt, err := parsePacket(v[0].Bytes)
		if err != nil {
			return env, err
		}
		env.Packet = pkt
	}
	return env, nil
}

func DirectPacketToEnvelope(packet []byte) []byte {
	return msg(fieldBytes(1, packet))
}

func ParseDirectFromRadio(raw []byte) (Envelope, error) {
	fields, err := parse(raw)
	if err != nil {
		return Envelope{}, err
	}
	if len(fields[1]) == 0 || len(fields[1][0].Bytes) == 0 {
		return Envelope{}, errors.New("fromradio message did not include a mesh packet")
	}
	return ParseEnvelope(DirectPacketToEnvelope(fields[1][0].Bytes))
}

func DedupeHash(env Envelope) string {
	base := env.PacketRaw
	if len(base) == 0 {
		base = []byte(env.RawHex)
	}
	sum := sha256Bytes(base)
	return hex.EncodeToString(sum)
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

// parseWithLimits parses a protobuf message with configurable security limits.
// It tracks message size, field count, nesting depth, and memory allocation.
// Returns detailed errors when any limit is exceeded.
func parseWithLimits(raw []byte, state *parseState) (map[int][]wire, error) {
	// Check message size limit
	if err := state.checkMessageSize(len(raw)); err != nil {
		return nil, err
	}

	// Check nesting depth limit
	if err := state.enter(); err != nil {
		return nil, err
	}
	defer state.exit()

	out := map[int][]wire{}
	for i := 0; i < len(raw); {
		// Track and check field count
		if err := state.addField(); err != nil {
			return nil, err
		}

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
			// Track and check allocation for this byte slice
			if err := state.alloc(int(ln)); err != nil {
				return nil, err
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

// parse parses a protobuf message using safe default limits.
// This is the legacy API that maintains backward compatibility.
// For configurable limits, use parseWithLimits directly.
func parse(raw []byte) (map[int][]wire, error) {
	state := newParseState(DefaultParseLimits())
	return parseWithLimits(raw, state)
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
func sha256Bytes(raw []byte) []byte {
	sum := sha256.Sum256(raw)
	return sum[:]
}

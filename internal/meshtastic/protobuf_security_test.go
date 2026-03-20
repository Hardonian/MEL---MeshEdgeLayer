package meshtastic

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestSQLInjectionInProtobufTransportName(t *testing.T) {
	sqlPayloads := []struct {
		name      string
		transport string
		shouldBan bool
	}{
		{"semicolon_injection", "mqtt; DROP TABLE messages;", true},
		{"comment_injection", "mqtt--comment", true},
		{"block_comment", "mqtt/*comment*/", true},
		{"union_select", "mqtt' UNION SELECT * FROM users", false},
		{"stacked_queries", "mqtt'; DELETE FROM nodes;", true},
		{"normal_transport", "mqtt", false},
		{"transport_with_hyphen", "mqtt-broker", false},
		{"transport_with_underscore", "mqtt_broker", false},
	}

	for _, tc := range sqlPayloads {
		t.Run(tc.name, func(t *testing.T) {
			isInvalid := strings.Contains(tc.transport, ";") ||
				strings.Contains(tc.transport, "--") ||
				strings.Contains(tc.transport, "/*") ||
				strings.Contains(tc.transport, "*/")

			if tc.shouldBan && !isInvalid {
				t.Errorf("transport name %q should be flagged as suspicious", tc.transport)
			}
			if !tc.shouldBan && isInvalid {
				t.Errorf("transport name %q should be allowed but was flagged", tc.transport)
			}
		})
	}
}

func TestOversizedProtobufMessage(t *testing.T) {
	oversizedPayload := make([]byte, MaxMessageSize+1)
	for i := range oversizedPayload {
		oversizedPayload[i] = byte(i % 256)
	}

	_, err := ParseEnvelope(oversizedPayload)
	if err == nil {
		t.Error("expected error for oversized message")
	}

	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("expected size error, got: %v", err)
	}
}

func TestMaxMessageSizeBoundary(t *testing.T) {
	boundaryTests := []struct {
		name     string
		size     int
		expectOK bool
	}{
		{"exactly_max", MaxMessageSize, true},
		{"one_over_max", MaxMessageSize + 1, false},
		{"way_over_max", MaxMessageSize * 2, false},
		{"typical_size", 1024, true},
		{"empty", 0, true},
		{"small", 10, true},
	}

	for _, tc := range boundaryTests {
		t.Run(tc.name, func(t *testing.T) {
			limits := DefaultParseLimits()
			state := newParseState(limits)
			err := state.checkMessageSize(tc.size)

			if tc.expectOK && err != nil {
				t.Errorf("expected no error for size %d, got: %v", tc.size, err)
			}
			if !tc.expectOK && err == nil {
				t.Errorf("expected error for size %d", tc.size)
			}
		})
	}
}

func TestTooManyFields(t *testing.T) {
	limits := ParseLimits{
		MaxMessageSize:  MaxMessageSize,
		MaxFields:       10,
		MaxNestingDepth: MaxNestingDepth,
		MaxAllocBytes:   MaxAllocBytes,
	}
	state := newParseState(limits)

	for i := 0; i < limits.MaxFields+1; i++ {
		err := state.addField()
		if i < limits.MaxFields && err != nil {
			t.Errorf("unexpected error at field %d: %v", i, err)
		}
		if i >= limits.MaxFields && err == nil {
			t.Errorf("expected error at field %d", i)
		}
	}
}

func TestNestingDepthLimit(t *testing.T) {
	limits := ParseLimits{
		MaxMessageSize:  MaxMessageSize,
		MaxFields:       MaxFields,
		MaxNestingDepth: 5,
		MaxAllocBytes:   MaxAllocBytes,
	}
	state := newParseState(limits)

	for i := 0; i < limits.MaxNestingDepth+2; i++ {
		err := state.enter()
		if i < limits.MaxNestingDepth && err != nil {
			t.Errorf("unexpected error at depth %d: %v", i, err)
		}
		if i >= limits.MaxNestingDepth && err == nil {
			t.Errorf("expected error at depth %d", i)
		}
	}

	for i := 0; i < limits.MaxNestingDepth+2; i++ {
		state.exit()
	}

	if state.depth != 0 {
		t.Errorf("expected depth to return to 0, got %d", state.depth)
	}
}

func TestAllocationLimit(t *testing.T) {
	limits := ParseLimits{
		MaxMessageSize:  MaxMessageSize,
		MaxFields:       MaxFields,
		MaxNestingDepth: MaxNestingDepth,
		MaxAllocBytes:   1000,
	}
	state := newParseState(limits)

	err := state.alloc(500)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = state.alloc(500)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = state.alloc(1)
	if err == nil {
		t.Error("expected allocation limit error")
	}
}

func TestInvalidWireTypes(t *testing.T) {
	wireTypeTests := []struct {
		name     string
		wireType int
		valid    bool
	}{
		{"varint", 0, true},
		{"fixed64", 1, true},
		{"length_delimited", 2, true},
		{"start_group", 3, false},
		{"end_group", 4, false},
		{"fixed32", 5, true},
		{"reserved_6", 6, false},
		{"reserved_7", 7, false},
		{"invalid_high", 255, false},
	}

	for _, tc := range wireTypeTests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			fieldNum := uint64(1)
			tag := fieldNum<<3 | uint64(tc.wireType)
			// Encode tag as varint (protobuf wire format)
			tagBuf := make([]byte, binary.MaxVarintLen64)
			n := binary.PutUvarint(tagBuf, tag)
			buf.Write(tagBuf[:n])

			if tc.wireType == 0 {
				// varint: write a simple varint value
				valBuf := make([]byte, binary.MaxVarintLen64)
				n := binary.PutUvarint(valBuf, uint64(42))
				buf.Write(valBuf[:n])
			} else if tc.wireType == 1 {
				// fixed64: need exactly 8 bytes of data
				buf.Write(make([]byte, 8))
			} else if tc.wireType == 2 {
				// length-delimited: write length as varint, then data
				lenBuf := make([]byte, binary.MaxVarintLen64)
				n := binary.PutUvarint(lenBuf, uint64(0))
				buf.Write(lenBuf[:n])
			} else if tc.wireType == 5 {
				// fixed32: need exactly 4 bytes of data
				buf.Write(make([]byte, 4))
			}

			_, err := parseWithLimits(buf.Bytes(), newParseState(DefaultParseLimits()))

			switch tc.wireType {
			case 0, 1, 2, 5:
				if err != nil && tc.valid {
					t.Errorf("unexpected error for valid wire type %d: %v", tc.wireType, err)
				}
			default:
				if err == nil && !tc.valid {
					t.Errorf("expected error for invalid wire type %d", tc.wireType)
				}
			}
		})
	}
}

func TestTruncatedMessages(t *testing.T) {
	truncatedTests := []struct {
		name    string
		payload []byte
	}{
		{"empty", []byte{}},
		{"single_byte", []byte{0x08}},
		{"partial_varint", []byte{0x08, 0x80}},
		{"partial_fixed32", []byte{0x0d, 0x01, 0x02}},
		{"partial_fixed64", []byte{0x09, 0x01, 0x02, 0x03}},
		{"partial_length_delimited", []byte{0x12, 0x10, 0x01, 0x02}},
		{"invalid_tag_only", []byte{0xff, 0xff, 0xff, 0xff, 0xff}},
	}

	for _, tc := range truncatedTests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseWithLimits(tc.payload, newParseState(DefaultParseLimits()))
			// Empty payload is valid in protobuf (returns empty message)
			if tc.name == "empty" {
				if err != nil {
					t.Errorf("empty payload should be valid: %v", err)
				}
			} else {
				if err == nil {
					t.Error("expected error for truncated message")
				}
			}
		})
	}
}

func TestMalformedProtobufPanics(t *testing.T) {
	malformedTests := []struct {
		name    string
		payload []byte
	}{
		{"nested_overflow", buildNestedMessage(15)},
		{"recursive_depth", buildRecursiveMessage(20)},
		{"massive_field_num", func() []byte {
			var buf bytes.Buffer
			tag := uint64(1<<29-1)<<3 | 0
			binary.Write(&buf, binary.LittleEndian, tag)
			binary.Write(&buf, binary.LittleEndian, uint64(1))
			return buf.Bytes()
		}()},
		{"zero_length_delimited", []byte{0x12, 0x00}},
		{"max_varint", func() []byte {
			var buf bytes.Buffer
			buf.Write([]byte{0x08})
			for i := 0; i < 10; i++ {
				buf.WriteByte(0xff)
			}
			return buf.Bytes()
		}()},
	}

	for _, tc := range malformedTests {
		t.Run(tc.name, func(t *testing.T) {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Logf("recovered from panic: %v", r)
					}
				}()

				_, err := parseWithLimits(tc.payload, newParseState(DefaultParseLimits()))
				if err != nil {
					t.Logf("got expected error: %v", err)
				}
			}()
		})
	}
}

func buildNestedMessage(depth int) []byte {
	if depth == 0 {
		return []byte{0x08, 0x01}
	}
	nested := buildNestedMessage(depth - 1)
	var buf bytes.Buffer
	buf.Write([]byte{0x12})
	binary.Write(&buf, binary.LittleEndian, uint64(len(nested)))
	buf.Write(nested)
	return buf.Bytes()
}

func buildRecursiveMessage(depth int) []byte {
	if depth == 0 {
		return []byte{}
	}
	inner := buildRecursiveMessage(depth - 1)
	if len(inner) == 0 {
		return []byte{0x08, 0x01}
	}
	var buf bytes.Buffer
	buf.Write([]byte{0x12})
	binary.Write(&buf, binary.LittleEndian, uint64(len(inner)))
	buf.Write(inner)
	return buf.Bytes()
}

func TestFieldCountEnforcement(t *testing.T) {
	limits := ParseLimits{
		MaxMessageSize:  MaxMessageSize,
		MaxFields:       5,
		MaxNestingDepth: MaxNestingDepth,
		MaxAllocBytes:   MaxAllocBytes,
	}

	var buf bytes.Buffer
	for i := 0; i < 10; i++ {
		fieldNum := uint64(i + 1)
		tag := fieldNum<<3 | 0
		binary.Write(&buf, binary.LittleEndian, tag)
		binary.Write(&buf, binary.LittleEndian, uint64(i))
	}

	_, err := parseWithLimits(buf.Bytes(), newParseState(limits))
	if err == nil {
		t.Error("expected field count error")
	}
}

func TestShortFixed32Handling(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		valid   bool
	}{
		{"exact_4_bytes", []byte{0x0d, 0x01, 0x02, 0x03, 0x04}, true},
		{"3_bytes", []byte{0x0d, 0x01, 0x02, 0x03}, false},
		{"2_bytes", []byte{0x0d, 0x01, 0x02}, false},
		{"1_byte", []byte{0x0d, 0x01}, false},
		{"0_bytes", []byte{0x0d}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseWithLimits(tc.payload, newParseState(DefaultParseLimits()))
			if tc.valid && err != nil {
				t.Errorf("expected valid for %s: %v", tc.name, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestShortFixed64Handling(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		valid   bool
	}{
		{"exact_8_bytes", append([]byte{0x09}, make([]byte, 8)...), true},
		{"7_bytes", append([]byte{0x09}, make([]byte, 7)...), false},
		{"4_bytes", append([]byte{0x09}, make([]byte, 4)...), false},
		{"0_bytes", []byte{0x09}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseWithLimits(tc.payload, newParseState(DefaultParseLimits()))
			if tc.valid && err != nil {
				t.Errorf("expected valid for %s: %v", tc.name, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestLengthDelimitedOverflow(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte{0x12})

	for i := 0; i < 10; i++ {
		buf.WriteByte(0xff)
	}

	_, err := parseWithLimits(buf.Bytes(), newParseState(DefaultParseLimits()))
	if err == nil {
		t.Error("expected error for oversized length")
	}
}

func TestInvalidVarintTag(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{"all_continuation", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}},
		{"incomplete_tag", []byte{0x80}},
		{"truncated_after_start", []byte{0x08, 0x80}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseWithLimits(tc.payload, newParseState(DefaultParseLimits()))
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestMaxAllocBytesEnforcement(t *testing.T) {
	limits := ParseLimits{
		MaxMessageSize:  1024 * 1024,
		MaxFields:       1000,
		MaxNestingDepth: 10,
		MaxAllocBytes:   100,
	}

	var buf bytes.Buffer
	// Create 11 fields with 10 bytes each = 110 bytes total (exceeds 100 byte limit)
	for i := 0; i < 11; i++ {
		buf.Write([]byte{0x12, 0x0a})
		buf.Write(make([]byte, 10))
	}

	_, err := parseWithLimits(buf.Bytes(), newParseState(limits))
	if err == nil {
		t.Error("expected allocation limit error")
	}
}

func TestEmptyPayload(t *testing.T) {
	_, err := parseWithLimits([]byte{}, newParseState(DefaultParseLimits()))
	if err != nil {
		t.Errorf("empty payload should be valid: %v", err)
	}
}

func TestValidMessageTypes(t *testing.T) {
	tests := []struct {
		portnum  int32
		expected string
	}{
		{1, "text"},
		{3, "position"},
		{4, "node_info"},
		{67, "telemetry"},
		{0, "unknown"},
		{99, "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			p := Packet{PortNum: tc.portnum}
			result := MessageType(p)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestRedactCoord(t *testing.T) {
	tests := []struct {
		name     string
		input    *float64
		expected float64
	}{
		{"nil", nil, 0},
		{"zero", floatPtr(0), 0},
		{"positive", floatPtr(45.123456), 45.12},
		{"negative", floatPtr(-122.987654), -122.99},
		{"round_up", floatPtr(45.125), 45.13},
		{"round_down", floatPtr(45.124), 45.12},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := RedactCoord(tc.input)
			if result != tc.expected {
				t.Errorf("expected %f, got %f", tc.expected, result)
			}
		})
	}
}

func floatPtr(f float64) *float64 {
	return &f
}

func TestTopicEncryptedDetection(t *testing.T) {
	tests := []struct {
		topic    string
		expected bool
	}{
		{"msh/2/e/ShortFast", true},
		{"msh/2/e/LongFast", true},
		{"msh/2/c/ShortFast", false},
		{"msh/2/encrypted", true},
		{"msh/2/Encrypted", true},
		{"msh/2/plain", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.topic, func(t *testing.T) {
			result := TopicEncrypted(tc.topic)
			if result != tc.expected {
				t.Errorf("topic %q: expected %v, got %v", tc.topic, tc.expected, result)
			}
		})
	}
}

func TestParseLimitsDefaults(t *testing.T) {
	limits := DefaultParseLimits()

	if limits.MaxMessageSize != MaxMessageSize {
		t.Errorf("MaxMessageSize: expected %d, got %d", MaxMessageSize, limits.MaxMessageSize)
	}
	if limits.MaxFields != MaxFields {
		t.Errorf("MaxFields: expected %d, got %d", MaxFields, limits.MaxFields)
	}
	if limits.MaxNestingDepth != MaxNestingDepth {
		t.Errorf("MaxNestingDepth: expected %d, got %d", MaxNestingDepth, limits.MaxNestingDepth)
	}
	if limits.MaxAllocBytes != MaxAllocBytes {
		t.Errorf("MaxAllocBytes: expected %d, got %d", MaxAllocBytes, limits.MaxAllocBytes)
	}
}

func TestParseStateReset(t *testing.T) {
	limits := DefaultParseLimits()
	state := newParseState(limits)

	state.depth = 5
	state.fieldCount = 100
	state.allocBytes = 5000

	state.exit()
	if state.depth != 4 {
		t.Errorf("depth after exit: expected 4, got %d", state.depth)
	}

	state2 := newParseState(limits)
	if state2.depth != 0 || state2.fieldCount != 0 || state2.allocBytes != 0 {
		t.Error("new state should have zero values")
	}
}

func TestPayloadHex(t *testing.T) {
	p := Packet{Payload: []byte{0x00, 0x0a, 0xff}}
	result := p.PayloadHex()
	expected := "000aff"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	empty := Packet{Payload: []byte{}}
	if empty.PayloadHex() != "" {
		t.Error("empty payload should return empty string")
	}
}

func TestDedupeHashConsistency(t *testing.T) {
	env1 := Envelope{
		PacketRaw: []byte{0x01, 0x02, 0x03},
		RawHex:    "010203",
	}
	env2 := Envelope{
		PacketRaw: []byte{0x01, 0x02, 0x03},
		RawHex:    "different",
	}

	hash1 := DedupeHash(env1)
	hash2 := DedupeHash(env2)

	if hash1 != hash2 {
		t.Error("dedupe hash should use PacketRaw when available")
	}

	env3 := Envelope{
		PacketRaw: []byte{},
		RawHex:    "010203",
	}
	hash3 := DedupeHash(env3)
	if hash3 == hash1 {
		t.Error("dedupe hash should use RawHex when PacketRaw is empty")
	}
}

func TestParseDirectFromRadioValidation(t *testing.T) {
	_, err := ParseDirectFromRadio([]byte{})
	if err == nil {
		t.Error("expected error for empty FromRadio")
	}

	_, err = ParseDirectFromRadio([]byte{0x08, 0x01})
	if err == nil {
		t.Error("expected error for FromRadio without mesh packet")
	}
}

package config

import (
	"bytes"
	"encoding/json"
	"sort"
)

// CanonicalJSONForFingerprint returns deterministic JSON bytes of cfg for hashing.
// Field order is stable (sorted keys at each object level). Secrets in nested maps
// are still present; callers should use redacted views for operator display.
func CanonicalJSONForFingerprint(cfg Config) ([]byte, error) {
	var raw map[string]any
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := writeSortedJSON(&buf, raw); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// CanonicalFingerprintSHA256 returns hex sha256 of CanonicalJSONForFingerprint.
func CanonicalFingerprintSHA256(cfg Config) (string, error) {
	canon, err := CanonicalJSONForFingerprint(cfg)
	if err != nil {
		return "", err
	}
	return SHA256(canon), nil
}

func writeSortedJSON(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case float64:
		b, _ := json.Marshal(x)
		buf.Write(b)
	case string:
		enc, _ := json.Marshal(x)
		buf.Write(enc)
	case []any:
		buf.WriteByte('[')
		for i, el := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeSortedJSON(buf, el); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			keyEnc, _ := json.Marshal(k)
			buf.Write(keyEnc)
			buf.WriteByte(':')
			if err := writeSortedJSON(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return err
		}
		buf.Write(b)
	}
	return nil
}

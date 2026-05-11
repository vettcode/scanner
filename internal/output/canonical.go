// Package output implements the scanner's output pipeline:
// canonical JSON serialization, integrity signing, co-signing,
// terminal formatting, and JSON file output.
package output

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// CanonicalJSON serializes v to canonical JSON per RFC 8785 (JCS).
// Keys are sorted lexicographically at all nesting levels.
// No whitespace, no HTML escaping.
func CanonicalJSON(v interface{}) ([]byte, error) {
	// Marshal to standard JSON first
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("canonical json marshal: %w", err)
	}
	return CanonicalJSONFromRaw(data)
}

// CanonicalJSONFromRaw takes raw JSON bytes and produces canonical JSON.
// Round-trips through interface{} with UseNumber to preserve int/float
// distinction, then re-encodes with sorted map keys.
func CanonicalJSONFromRaw(raw []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var generic interface{}
	if err := dec.Decode(&generic); err != nil {
		return nil, fmt.Errorf("canonical json decode: %w", err)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(generic); err != nil {
		return nil, fmt.Errorf("canonical json encode: %w", err)
	}

	// json.Encoder.Encode appends a newline; strip it
	result := buf.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// CanonicalChecksum computes the SHA-256 hash of the canonical JSON
// representation of v and returns the hex-encoded hash string and the
// canonical bytes.
func CanonicalChecksum(v interface{}) (string, []byte, error) {
	data, err := CanonicalJSON(v)
	if err != nil {
		return "", nil, err
	}
	return checksumBytes(data), data, nil
}

// integrityComputedFields are the integrity sub-fields excluded from the hash.
// These are either the signature outputs themselves (circular dependency) or
// fields that are set AFTER signing completes (cosigned, verification_level).
// The nonce and scanner_public_key_id ARE included — the nonce binds the
// co-sign session to this specific scan data.
var integrityComputedFields = map[string]bool{
	"scan_checksum":          true,
	"scanner_signature":      true,
	"platform_cosignature":   true,
	"platform_public_key_id": true,
	"cosigned":               true,
	"verification_level":     true,
}

// CanonicalChecksumForSigning serializes v to canonical JSON, strips only
// the computed fields from the "integrity" block (checksum, signatures),
// and returns the SHA-256 hash. The nonce, scanner_public_key_id, and
// cosigned fields remain in the hash so the nonce binds to the scan data.
func CanonicalChecksumForSigning(v interface{}) (string, []byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", nil, fmt.Errorf("marshal: %w", err)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var generic interface{}
	if err := dec.Decode(&generic); err != nil {
		return "", nil, fmt.Errorf("decode: %w", err)
	}

	// Strip only the computed fields from the integrity block
	if m, ok := generic.(map[string]interface{}); ok {
		if integrity, ok := m["integrity"].(map[string]interface{}); ok {
			for field := range integrityComputedFields {
				delete(integrity, field)
			}
		}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(generic); err != nil {
		return "", nil, fmt.Errorf("encode: %w", err)
	}

	result := buf.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return checksumBytes(result), result, nil
}

func checksumBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

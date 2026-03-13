package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cross-language test vectors from Section 5.8 of the design doc.
// Both scanner (Go) and platform (Python) must produce identical
// SHA-256 hashes for the same input.

func TestCanonicalJSON_Vector1(t *testing.T) {
	input := `{"z": 1, "a": {"c": true, "b": [3, 1, 2]}, "m": null}`
	expected := `{"a":{"b":[3,1,2],"c":true},"m":null,"z":1}`
	expectedHash := "ad507d446db1dec51409507e057e5904c5507aecc69126227b28bf79c77e06f3"

	got, err := CanonicalJSONFromRaw([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, expected, string(got))
	assert.Equal(t, expectedHash, checksumBytes(got))
}

func TestCanonicalJSON_Vector2(t *testing.T) {
	input := `{"name": "Acme™ SaaS", "loc": 42600, "score": 87, "flags": []}`
	expected := `{"flags":[],"loc":42600,"name":"Acme™ SaaS","score":87}`
	expectedHash := "eba6b376ec325015a44114dd546bff5650df60b5f49beab4cb2f95d594261c6f"

	got, err := CanonicalJSONFromRaw([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, expected, string(got))
	assert.Equal(t, expectedHash, checksumBytes(got))
}

func TestCanonicalJSON_Vector3(t *testing.T) {
	input := `{"emoji": "🔒", "path": "src/auth/login.ts", "null_field": null}`
	expected := `{"emoji":"🔒","null_field":null,"path":"src/auth/login.ts"}`
	expectedHash := "f5611ee69af536c6027950e16e198e2438555b8fefb0faa7c52b3c580090c245"

	got, err := CanonicalJSONFromRaw([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, expected, string(got))
	assert.Equal(t, expectedHash, checksumBytes(got))
}

func TestCanonicalJSON_Struct(t *testing.T) {
	type inner struct {
		B []int `json:"b"`
		C bool  `json:"c"`
	}
	type outer struct {
		Z int     `json:"z"`
		A inner   `json:"a"`
		M *string `json:"m"`
	}

	v := outer{Z: 1, A: inner{B: []int{3, 1, 2}, C: true}, M: nil}
	got, err := CanonicalJSON(v)
	require.NoError(t, err)
	// Struct fields sorted: a < m < z
	assert.Equal(t, `{"a":{"b":[3,1,2],"c":true},"m":null,"z":1}`, string(got))
}

func TestCanonicalJSON_NoHTMLEscaping(t *testing.T) {
	input := `{"html": "<script>alert('xss')</script>"}`
	got, err := CanonicalJSONFromRaw([]byte(input))
	require.NoError(t, err)
	// <, >, & should NOT be escaped with SetEscapeHTML(false)
	assert.Equal(t, `{"html":"<script>alert('xss')</script>"}`, string(got))
}

func TestCanonicalJSON_NumberPreservation(t *testing.T) {
	input := `{"int": 42, "float": 3.14, "zero": 0}`
	got, err := CanonicalJSONFromRaw([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, `{"float":3.14,"int":42,"zero":0}`, string(got))
}

func TestCanonicalChecksumForSigning(t *testing.T) {
	type integrity struct {
		ScanChecksum       string  `json:"scan_checksum"`
		ScannerSignature   string  `json:"scanner_signature"`
		ScannerPublicKeyID string  `json:"scanner_public_key_id"`
		CosignNonce        *string `json:"cosign_nonce"`
		Cosigned           bool    `json:"cosigned"`
	}
	type data struct {
		Name      string    `json:"name"`
		Value     int       `json:"value"`
		Integrity integrity `json:"integrity"`
	}

	nonce := "test-nonce-123"
	v := data{
		Name:  "test",
		Value: 42,
		Integrity: integrity{
			ScanChecksum:       "should-be-excluded",
			ScannerSignature:   "should-be-excluded",
			ScannerPublicKeyID: "should-stay",
			CosignNonce:        &nonce,
			Cosigned:           false,
		},
	}
	checksum, canonical, err := CanonicalChecksumForSigning(v)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum)
	// Computed fields excluded
	assert.NotContains(t, string(canonical), "scan_checksum")
	assert.NotContains(t, string(canonical), "scanner_signature")
	// Nonce and key ID included (bind session to data)
	assert.Contains(t, string(canonical), "test-nonce-123")
	assert.Contains(t, string(canonical), "should-stay")
	assert.Contains(t, string(canonical), `"name":"test"`)
}

func TestCanonicalChecksumForSigning_NonceBound(t *testing.T) {
	type integrity struct {
		ScanChecksum string  `json:"scan_checksum"`
		CosignNonce  *string `json:"cosign_nonce"`
		Cosigned     bool    `json:"cosigned"`
	}
	type data struct {
		Value     int       `json:"value"`
		Integrity integrity `json:"integrity"`
	}

	nonce1 := "nonce-aaa"
	nonce2 := "nonce-bbb"
	v1 := data{Value: 1, Integrity: integrity{CosignNonce: &nonce1}}
	v2 := data{Value: 1, Integrity: integrity{CosignNonce: &nonce2}}

	hash1, _, err := CanonicalChecksumForSigning(v1)
	require.NoError(t, err)
	hash2, _, err := CanonicalChecksumForSigning(v2)
	require.NoError(t, err)
	// Different nonces must produce different checksums
	assert.NotEqual(t, hash1, hash2)
}

func TestCanonicalJSON_EmptyObject(t *testing.T) {
	got, err := CanonicalJSONFromRaw([]byte(`{}`))
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(got))
}

func TestCanonicalJSON_NestedSorting(t *testing.T) {
	input := `{"z": {"b": 2, "a": 1}, "a": {"d": 4, "c": 3}}`
	got, err := CanonicalJSONFromRaw([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, `{"a":{"c":3,"d":4},"z":{"a":1,"b":2}}`, string(got))
}

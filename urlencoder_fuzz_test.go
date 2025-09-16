package urlcodec

import (
	"encoding/json"
	"net/url"
	"testing"
)

// FuzzURLEncoderEncode fuzzes the Encode method.
// It expects the fuzz input to be a JSON string representing a map[string]any.
// If the input JSON is invalid, the iteration is skipped.
// Run with: go test -fuzz=FuzzURLEncoderEncode
func FuzzURLEncoderEncode(f *testing.F) {
	// Seed corpus with some valid JSON inputs.
	seeds := []string{
		`{"key": "value"}`,
		`{"number": 123, "flag": true}`,
		`{"nested": {"a": "b"}}`,
		`{"slice": ["x", "y", "z"]}`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, jsonStr string) {
		var data map[string]any
		// Skip inputs that don't unmarshal correctly.
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			t.Skip("invalid JSON input")
		}
		encoder := NewURLEncoder()
		values, err := encoder.Encode(data)
		// We don't require no error because certain inputs
		// (like missing JSON tags) may legitimately produce an error.
		if err == nil {
			// Optionally, try decoding the encoded values.
			decoded, err := encoder.Decode(values)
			if err != nil {
				t.Logf("Decode error: %v", err)
			} else if decoded == nil {
				t.Error("decoded map is nil")
			}
		}
	})
}

// FuzzURLEncoderDecode fuzzes the Decode method.
// It expects the fuzz input to be a query string (URL-encoded form data).
// If the query string cannot be parsed, the iteration is skipped.
// Run with: go test -fuzz=FuzzURLEncoderDecode
func FuzzURLEncoderDecode(f *testing.F) {
	// Seed corpus with some valid query strings.
	seeds := []string{
		"key=value",
		"number=123",
		"nested.field=abc",
		"slice[0]=a&slice[1]=b",
		"map.key=val",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, qs string) {
		values, err := url.ParseQuery(qs)
		if err != nil {
			t.Skip("invalid query string")
		}
		encoder := NewURLEncoder()
		_, err = encoder.Decode(values)
		// We only want to ensure that the Decode method does not panic.
		if err != nil {
			t.Logf("Decode error: %v", err)
		}
	})
}

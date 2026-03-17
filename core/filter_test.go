package core

import (
	"bytes"
	"testing"
)

func TestParseRawClockifyEnvelope(t *testing.T) {
	respBytes := []byte(`{
		"count": 2,
		"requests": [
			{"id": "req-1", "createdAt": "2025-12-01T00:00:00Z"},
			{"id": "req-2", "createdAt": "2025-12-02T00:00:00Z"}
		]
	}`)

	got, err := ParseRawClockifyEnvelope(respBytes)
	if err != nil {
		t.Fatalf("ParseRawClockifyEnvelope() error = %v", err)
	}

	if got.Count != 2 {
		t.Fatalf("got.Count = %d, want 2", got.Count)
	}

	if len(got.Requests) != 2 {
		t.Fatalf("len(got.Requests) = %d, want 2", len(got.Requests))
	}

	if !bytes.Contains(got.Requests[0], []byte(`"id": "req-1"`)) {
		t.Fatalf("first raw request does not contain expected JSON: %s", got.Requests[0])
	}
}

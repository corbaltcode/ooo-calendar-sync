package core

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"
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

func mustRawMessage(t *testing.T, v any) json.RawMessage {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	return json.RawMessage(b)
}

func TestFilterRawRequestsByCreatedAt(t *testing.T) {
	rangeStart := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2025, 12, 3, 0, 0, 0, 0, time.UTC)
	timeoffStart := "2025-12-10T00:00:00Z"
	timeoffEnd := "2025-12-10T23:59:59Z"
	timeZone := "America/New_York"

	beforeStart := mustRawMessage(t, makeRequestWithCreatedAt(
		"before-start",
		timeZone,
		timeoffStart,
		timeoffEnd,
		time.Date(2025, 11, 30, 23, 59, 59, 0, time.UTC),
	))

	atStart := mustRawMessage(t, makeRequestWithCreatedAt(
		"at-start",
		timeZone,
		timeoffStart,
		timeoffEnd,
		time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
	))

	withinRange := mustRawMessage(t, makeRequestWithCreatedAt(
		"within-range",
		timeZone,
		timeoffStart,
		timeoffEnd,
		time.Date(2025, 12, 2, 12, 0, 0, 0, time.UTC),
	))

	atEnd := mustRawMessage(t, makeRequestWithCreatedAt(
		"at-end",
		timeZone,
		timeoffStart,
		timeoffEnd,
		time.Date(2025, 12, 3, 0, 0, 0, 0, time.UTC),
	))

	rawRequests := []json.RawMessage{
		beforeStart,
		atStart,
		withinRange,
		atEnd,
	}

	got := FilterRawRequestsByCreatedAt(rawRequests, rangeStart, rangeEnd)

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	var gotIDs []string
	for _, raw := range got {
		var r ClockifyRequest
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatalf("json.Unmarshal(filtered raw) error = %v", err)
		}
		gotIDs = append(gotIDs, r.ID)
	}

	wantIDs := []string{"at-start", "within-range"}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("got IDs = %v, want %v", gotIDs, wantIDs)
	}
}

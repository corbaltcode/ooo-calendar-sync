package core

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	assert.Equal(t, 2, got.Count)
	assert.Len(t, got.Requests, 2)
	assert.True(t, bytes.Contains(got.Requests[0], []byte(`"id": "req-1"`)))
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

	require.Len(t, got, 2)

	var gotIDs []string
	for _, raw := range got {
		var r ClockifyRequest
		require.NoError(t, json.Unmarshal(raw, &r))
		gotIDs = append(gotIDs, r.ID)
	}

	wantIDs := []string{"at-start", "within-range"}
	assert.Equal(t, wantIDs, gotIDs)
}

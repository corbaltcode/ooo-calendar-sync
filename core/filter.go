package core

import (
	"encoding/json"
	"log"
	"time"
)

type rawClockifyEnvelope struct {
	Count    int               `json:"count"`
	Requests []json.RawMessage `json:"requests"`
}

type requestTimestamps struct {
	CreatedAt string `json:"createdAt"`
	Status    struct {
		ChangedAt string `json:"changedAt"`
	} `json:"status"`
}

func ParseRawClockifyEnvelope(respBytes []byte) (rawClockifyEnvelope, error) {
	var env rawClockifyEnvelope
	if err := json.Unmarshal(respBytes, &env); err != nil {
		return rawClockifyEnvelope{}, err
	}
	return env, nil
}

// ParseClockifyRequests converts valid raw request payloads into ClockifyRequest structs and skips malformed JSON entries, logging each unmarshal error via log.Printf.
func ParseClockifyRequests(rawRequests []json.RawMessage) []ClockifyRequest {
	requests := make([]ClockifyRequest, 0, len(rawRequests))

	for _, raw := range rawRequests {
		var r ClockifyRequest
		if err := json.Unmarshal(raw, &r); err != nil {
			log.Printf("skipping bad request: %v", err)
			continue
		}

		requests = append(requests, r)
	}

	return requests
}

func isTimestampInWindow(
	value string,
	start, end time.Time,
) bool {
	timestamp, err := ParseFlexibleRFC3339(value)
	if err != nil {
		return false
	}

	timestamp = timestamp.UTC()

	return !timestamp.Before(start) && timestamp.Before(end)
}

// Filters raw requests that were created or had their status updated in [start, end).
func FilterRawRequestsByActivity(
	rawRequests []json.RawMessage,
	start, end time.Time,
) []json.RawMessage {
	filtered := make([]json.RawMessage, 0, len(rawRequests))

	for _, raw := range rawRequests {
		var timestamps requestTimestamps
		if err := json.Unmarshal(raw, &timestamps); err != nil {
			continue
		}

		createdInWindow := isTimestampInWindow(timestamps.CreatedAt, start, end)

		statusUpdatedInWindow := isTimestampInWindow(timestamps.Status.ChangedAt, start, end)

		if createdInWindow || statusUpdatedInWindow {
			filtered = append(filtered, raw)
		}
	}

	return filtered
}

// FilterByActivity returns Clockify requests that were created or had their
// status updated within the given time span.
func FilterByActivity(
	respBytes []byte,
	start, end time.Time,
) (ClockifyEnvelope, error) {
	rawEnv, err := ParseRawClockifyEnvelope(respBytes)
	if err != nil {
		return ClockifyEnvelope{}, err
	}

	filtered := FilterRawRequestsByActivity(rawEnv.Requests, start, end)

	return ClockifyEnvelope{Requests: ParseClockifyRequests(filtered)}, nil
}

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

type createdOnly struct {
	CreatedAt string `json:"createdAt"`
}

func ParseRawClockifyEnvelope(respBytes []byte) (rawClockifyEnvelope, error) {
	var env rawClockifyEnvelope
	if err := json.Unmarshal(respBytes, &env); err != nil {
		return rawClockifyEnvelope{}, err
	}
	return env, nil
}

// Filters raw request payloads by createdAt in [start, end].
func FilterRawRequestsByCreatedAt(
	rawRequests []json.RawMessage,
	start, end time.Time,
) []json.RawMessage {

	filtered := make([]json.RawMessage, 0, len(rawRequests))

	for _, raw := range rawRequests {
		var c createdOnly
		if err := json.Unmarshal(raw, &c); err != nil {
			continue // skip if no createdAt
		}

		ct, err := ParseFlexibleRFC3339(c.CreatedAt)
		if err != nil {
			continue
		}
		ct = ct.UTC()

		if ct.Before(start) {
			continue
		}
		if !ct.Before(end) { // exclusive
			continue
		}

		filtered = append(filtered, raw)
	}
	return filtered
}

// Filter a raw Clockify response for out-of-office requests that were created within the given time span.
func FilterByCreatedAt(respBytes []byte, createdStart, createdEnd time.Time) (ClockifyEnvelope, error) {
	rawEnv, err := ParseRawClockifyEnvelope(respBytes)
	if err != nil {
		return ClockifyEnvelope{}, err
	}

	filtered := FilterRawRequestsByCreatedAt(rawEnv.Requests, createdStart, createdEnd)

	// TODO: Split parsing into a separate functions.
	var env ClockifyEnvelope
	for _, raw := range filtered {
		var r ClockifyRequest
		if err := json.Unmarshal(raw, &r); err != nil {
			log.Printf("skipping bad request: %v", err)
			continue
		}
		env.Requests = append(env.Requests, r)
	}

	return env, nil
}

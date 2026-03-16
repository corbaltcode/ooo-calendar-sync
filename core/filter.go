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

// Filter a raw Clockify response for out-of-office requests that were created within the given time span.
func FilterByCreatedAt(respBytes []byte, createdStart, createdEnd time.Time) (ClockifyEnvelope, error) {
	rawEnv, err := ParseRawClockifyEnvelope(respBytes)
	if err != nil {
		return ClockifyEnvelope{}, err
	}

	// TODO: Split filtering into a separate function.
	filtered := make([]json.RawMessage, 0, len(rawEnv.Requests))
	for _, raw := range rawEnv.Requests {
		var c createdOnly
		if err := json.Unmarshal(raw, &c); err != nil {
			continue // skip if no createdAt
		}

		ct, err := ParseFlexibleRFC3339(c.CreatedAt)
		if err != nil {
			continue
		}
		ct = ct.UTC()

		if ct.Before(createdStart) {
			continue
		}
		if !ct.Before(createdEnd) { // exclusive
			continue
		}

		filtered = append(filtered, raw)
	}

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

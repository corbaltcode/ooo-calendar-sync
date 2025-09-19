package core

import (
	"encoding/json"
	"log"
	"time"
)

// Filter a raw Clockify response for out-of-office requests that were created within the given time span.
func FilterByCreatedAt(respBytes []byte, createdStart, createdEnd time.Time) (Envelope, error) {
	// Decode top-level API response into raw messages
	type apiResp struct {
		Count    int               `json:"count"`
		Requests []json.RawMessage `json:"requests"`
	}
	var ar apiResp
	if err := json.Unmarshal(respBytes, &ar); err != nil {
		return Envelope{}, err
	}

	// Step 1: filter JSON by createdAt
	type createdOnly struct {
		CreatedAt string `json:"createdAt"`
	}

	filtered := make([]json.RawMessage, 0, len(ar.Requests))
	for _, raw := range ar.Requests {
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

	var env Envelope
	for _, raw := range filtered {
		var r Request
		if err := json.Unmarshal(raw, &r); err != nil {
			log.Printf("skipping bad request: %v", err)
			continue
		}
		env.Requests = append(env.Requests, r)
	}

	return env, nil
}

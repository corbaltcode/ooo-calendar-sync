package utils

import (
	"time"
)

var acceptedLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02",
}

func ParseTimeAny(s string) (time.Time, error) {
	var lastErr error
	for _, layout := range acceptedLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

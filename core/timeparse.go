package core

import (
	"fmt"
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

func ParseFlexibleRFC3339(s string) (time.Time, error) {
	return ParseTimeAny(s)
}

// Clockify expects YYYY-MM-DDTHH:MM:SS.ssssssZ (microseconds).
// https://docs.clockify.me/#tag/Time-Off/operation/getTimeOffRequest
func formatClockify(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000Z")
}

func ParseAndFormatClockifyTime(s string) (string, error) {
	t, err := ParseTimeAny(s)
	if err != nil {
		return "", fmt.Errorf("cannot parse time %q: %w", s, err)
	}
	return formatClockify(t), nil
}

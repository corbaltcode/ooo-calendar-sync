package core

import (
	"context"
	"time"

	"google.golang.org/api/calendar/v3"
)

// findClockifyEvents returns all events in calID whose private extended
// property "clockifyRequestId" == clockifyID, scoped to the given time range.
// It returns an empty slice if no such events exist.
func findClockifyEvents(
	ctx context.Context,
	srv *calendar.Service,
	calID string,
	clockifyID string,
	timeMin, timeMax time.Time,
) ([]*calendar.Event, error) {
	events, err := srv.Events.List(calID).
		// Filter by the extended property
		PrivateExtendedProperty("clockifyRequestId=" + clockifyID).
		TimeMin(timeMin.Format(time.RFC3339)).
		TimeMax(timeMax.Format(time.RFC3339)).
		SingleEvents(true).
		ShowDeleted(false).
		// Keep small but >1 so duplicates are visible
		MaxResults(10).
		Do()

	if err != nil {
		return nil, err
	}

	// events.Items will be nil or empty if no matches are found
	return events.Items, nil
}

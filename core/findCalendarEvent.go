package core

import (
	"context"
	"time"

	"google.golang.org/api/calendar/v3"
)

// findClockifyEvent returns the first event in calID whose private extended
// property "clockifyRequestId" == clockifyID, scoped to the given time range.
// It returns (nil, nil) if no such event exists.
func findClockifyEvent(
	ctx context.Context,
	srv *calendar.Service,
	calID string,
	clockifyID string,
	timeMin, timeMax time.Time,
) (*calendar.Event, error) {
	listCall := srv.Events.List(calID).
		// Filter by the extended property
		PrivateExtendedProperty("clockifyRequestId=" + clockifyID).
		TimeMin(timeMin.Format(time.RFC3339)).
		TimeMax(timeMax.Format(time.RFC3339)).
		SingleEvents(true).
		ShowDeleted(false).
		MaxResults(2)

	events, err := listCall.Do()
	if err != nil {
		return nil, err
	}
	if len(events.Items) == 0 {
		return nil, nil
	}
	// If somehow multiple exist, just return the first.
	return events.Items[0], nil
}

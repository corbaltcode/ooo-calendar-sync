package core

import (
	"errors"
	"time"
)

type SyncedClockifyRequest struct {
	ClockifyRequestID string `json:"clockifyRequestId"`
	UserID            string `json:"userId"`
	UserEmail         string `json:"userEmail"`
	Status            string `json:"status"`

	PeriodStart string `json:"periodStart"`
	PeriodEnd   string `json:"periodEnd"`

	CreatedAt string `json:"createdAt"`

	GoogleCalendarEventID string `json:"googleCalendarEventId,omitempty"`

	LastSeenAt string `json:"lastSeenAt"`
	SyncState  string `json:"syncState"`
}

// ToDynamoItem converts a Clockify request into the persistence model
// that will be stored in our DynamoDB table. The returned item represents
// the current known state of the request and serves as the first step
// toward a persistence layer that will eventually allow us to determine
// whether corresponding Google Calendar events should be created,
// updated, or deleted.
func ToDynamoItem(r ClockifyRequest, now time.Time) (*SyncedClockifyRequest, error) {
	if r.ID == "" {
		return nil, errors.New("missing Clockify request ID")
	}

	item := &SyncedClockifyRequest{
		ClockifyRequestID: r.ID,
		Status:            r.Status.StatusType,
		LastSeenAt:        now.UTC().Format(time.RFC3339),
		SyncState:         "pending",
	}

	item.PeriodStart = r.TimeOffPeriod.Period.Start
	item.PeriodEnd = r.TimeOffPeriod.Period.End
	item.CreatedAt = r.CreatedAt
	item.UserEmail = r.UserEmail

	return item, nil
}

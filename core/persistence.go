package core

import (
	"errors"
	"time"
)

type SyncedClockifyRequest struct {
	ClockifyRequestID string `json:"clockifyRequestId" dynamodbav:"ClockifyRequestId"`
	UserID            string `json:"userId" dynamodbav:"UserId"`
	UserEmail         string `json:"userEmail" dynamodbav:"UserEmail"`
	Status            string `json:"status" dynamodbav:"Status"`

	PeriodStart string `json:"periodStart" dynamodbav:"PeriodStart"`
	PeriodEnd   string `json:"periodEnd" dynamodbav:"PeriodEnd"`

	CreatedAt  string `json:"createdAt" dynamodbav:"CreatedAt"`
	LastSeenAt string `json:"lastSeenAt" dynamodbav:"LastSeenAt"`
	SyncState  string `json:"syncState" dynamodbav:"SyncState"`

	GoogleCalendarEventID string `json:"googleCalendarEventId,omitempty" dynamodbav:"GoogleCalendarEventId,omitempty"`
}

// ToDynamoItem converts a Clockify request into the persistence model
// that will be stored in our DynamoDB table. The returned item represents
// the current known state of the request and serves as the first step
// toward a persistence layer that will eventually allow us to determine
// whether corresponding Google Calendar events should be created,
// updated, or deleted.
func (r *ClockifyRequest) ToDynamoItem(options ...func(*SyncedClockifyRequest)) (*SyncedClockifyRequest, error) {
	if r.ID == "" {
		return nil, errors.New("missing Clockify request ID")
	}

	item := &SyncedClockifyRequest{
		ClockifyRequestID: r.ID,
		Status:            r.Status.StatusType,
		SyncState:         "pending",
		PeriodStart:       r.TimeOffPeriod.Period.Start,
		PeriodEnd:         r.TimeOffPeriod.Period.End,
		CreatedAt:         r.CreatedAt,
		UserEmail:         r.UserEmail,
	}

	for _, opt := range options {
		opt(item)
	}

	if item.LastSeenAt == "" {
		item.LastSeenAt = time.Now().UTC().Format(time.RFC3339)
	}

	return item, nil
}

func WithLastSeenAt(now time.Time) func(*SyncedClockifyRequest) {
	return func(item *SyncedClockifyRequest) {
		item.LastSeenAt = now.UTC().Format(time.RFC3339)
	}
}

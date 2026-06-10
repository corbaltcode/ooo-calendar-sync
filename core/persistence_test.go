package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToDynamoItem(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	req := ClockifyRequest{
		ID:           "request-123",
		UserEmail:    "person@example.com",
		CreatedAt:    "2026-06-08T10:00:00Z",
		PolicyName:   "Vacation",
		UserTimeZone: "America/New_York",
	}

	req.Status.StatusType = "APPROVED"

	req.TimeOffPeriod.Period.Start = "2026-06-10T00:00:00Z"
	req.TimeOffPeriod.Period.End = "2026-06-12T00:00:00Z"

	item, err := req.ToDynamoItem(WithLastSeenAt(now))

	require.NoError(t, err)
	require.NotNil(t, item)

	assert.Equal(t, "request-123", item.ClockifyRequestID)
	assert.Equal(t, "person@example.com", item.UserEmail)
	assert.Equal(t, "APPROVED", item.Status)
	assert.Equal(t, "2026-06-10T00:00:00Z", item.PeriodStart)
	assert.Equal(t, "2026-06-12T00:00:00Z", item.PeriodEnd)
	assert.Equal(t, "2026-06-08T10:00:00Z", item.CreatedAt)
	assert.Equal(t, "2026-06-08T12:00:00Z", item.LastSeenAt)
	assert.Equal(t, "pending", item.SyncState)
}

func TestToDynamoItemReturnsErrorWhenRequestIDMissing(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	req := ClockifyRequest{}

	item, err := req.ToDynamoItem(WithLastSeenAt(now))

	require.Error(t, err)
	assert.Nil(t, item)
	assert.Contains(t, err.Error(), "missing Clockify request ID")
}

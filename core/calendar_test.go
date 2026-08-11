package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/jwt"
)

func TestInsertOOOEvent_ReturnsErrorsForInvalidRequests(t *testing.T) {
	ctx := context.Background()
	jwtCfg := jwt.Config{}
	calendarIDs := []string{"primary"}

	reqs := []ClockifyRequest{
		fixtureBadTimeZone(),
		fixtureInvalidStartDate(),
		fixtureInvalidEndDate(),
	}

	for _, req := range reqs {
		t.Run(req.ID, func(t *testing.T) {
			_, err := InsertOOOEvents(ctx, jwtCfg, req, calendarIDs)

			require.Error(t, err)
		})
	}
}

func TestDeleteOOOEvents_ReturnsErrorsForFailedDeletes(t *testing.T) {
	ctx := context.Background()
	jwtCfg := jwt.Config{}
	userEmail := "test@email.com"

	events := []GoogleCalendarEvent{
		{
			CalendarID: "calendar-1",
			EventID:    "event-1",
		},
		{
			CalendarID: "calendar-2",
			EventID:    "event-2",
		},
	}

	err := DeleteOOOEvents(ctx, jwtCfg, userEmail, events)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "event-1")
	assert.Contains(t, err.Error(), "calendar-1")
	assert.Contains(t, err.Error(), "event-2")
	assert.Contains(t, err.Error(), "calendar-2")
}

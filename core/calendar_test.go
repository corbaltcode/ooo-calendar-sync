package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/jwt"
)

func TestInsertOOOEvent_ReturnsErrorsForInvalidRequests(t *testing.T) {
	ctx := context.Background()
	jwtCfg := &jwt.Config{}
	calendarIDs := []string{"primary"}

	reqs := []ClockifyRequest{
		fixtureBadTimeZone(),
		fixtureInvalidStartDate(),
		fixtureInvalidEndDate(),
	}

	for _, req := range reqs {
		t.Run(req.ID, func(t *testing.T) {
			err := InsertOOOEvent(ctx, jwtCfg, req, calendarIDs)

			require.Error(t, err)
		})
	}
}

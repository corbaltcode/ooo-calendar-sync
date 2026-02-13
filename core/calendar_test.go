package core

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/oauth2/jwt"
)

func TestInsertOOOEvents_CollectsErrors(t *testing.T) {
	ctx := context.Background()
	jwtCfg := &jwt.Config{}
	calendarIDs := []string{"primary"}

	reqs := []ClockifyRequest{
		fixtureBadTimeZone(),
		fixtureInvalidStartDate(),
		fixtureInvalidEndDate(),
	}

	err := InsertOOOEvents(ctx, jwtCfg, reqs, calendarIDs)
	if err == nil {
		t.Fatalf("expected non-nil error")
	}

	var multiErrors interface{ Unwrap() []error }
	if !errors.As(err, &multiErrors) {
		t.Fatalf("expected joined error with Unwrap() []error, got: %T", err)
	}

	collectedErrCount := len(multiErrors.Unwrap())
	expectedErrCount := len(reqs)

	if collectedErrCount != expectedErrCount {
		t.Fatalf("expected %d collected errors, got %d",
			expectedErrCount,
			collectedErrCount,
		)
	}
}

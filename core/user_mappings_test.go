package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractClockifyUserIDs(t *testing.T) {
	mappingsByEmail := map[string]UserCalendarMapping{
		"user1@example.com": {
			ClockifyUserID: "clockify-1",
		},
		"user2@example.com": {
			ClockifyUserID: "clockify-2",
		},
		"user3@example.com": {
			ClockifyUserID: "",
		},
	}

	userIDs := ExtractClockifyUserIDs(mappingsByEmail)

	assert.ElementsMatch(t, []string{"clockify-1", "clockify-2"}, userIDs)
}

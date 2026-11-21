package core

import "log"

type UserCalendarMapping struct {
	UserEmail      string
	ClockifyUserID string
	CalendarIDs    []string
}

func ExtractClockifyUserIDs(
	mappingsByEmail map[string]UserCalendarMapping,
) []string {
	clockifyUserIDs := make([]string, 0, len(mappingsByEmail))

	for userEmail, mapping := range mappingsByEmail {
		if mapping.ClockifyUserID == "" {
			log.Printf("Skipping user %s: missing Clockify user ID", userEmail)
			continue
		}

		clockifyUserIDs = append(clockifyUserIDs, mapping.ClockifyUserID)
	}

	return clockifyUserIDs
}

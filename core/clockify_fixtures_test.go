package core

import "time"

func makeRequest(id, tz, start, end string) ClockifyRequest {
	return makeRequestWithActivityTimestamps(
		id,
		tz,
		start,
		end,
		time.Date(2025, 12, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 1, 12, 0, 0, 0, time.UTC),
	)
}

func makeRequestWithActivityTimestamps(id, tz, start, end string, createdAt time.Time, statusChangedAt time.Time) ClockifyRequest {
	var r ClockifyRequest

	r.ID = id
	r.UserEmail = "fixture@example.com"
	r.UserTimeZone = tz
	r.CreatedAt = createdAt.Format(time.RFC3339)
	r.PolicyName = "Vacation"

	r.TimeOffPeriod.Period.Start = start
	r.TimeOffPeriod.Period.End = end
	r.Status.ChangedAt = statusChangedAt.Format(time.RFC3339)

	return r
}

func fixtureBadTimeZone() ClockifyRequest {
	return makeRequest("fixture-bad-time-zone", "Not/A_Timezone", "2025-12-10T00:00:00Z", "2025-12-10T23:59:59Z")
}
func fixtureInvalidStartDate() ClockifyRequest {
	return makeRequest("fixture-bad-start", "America/New_York", "not-a-date", "2025-12-10T23:59:59Z")
}
func fixtureInvalidEndDate() ClockifyRequest {
	return makeRequest("fixture-bad-end", "America/New_York", "2025-12-10T00:00:00Z", "not-a-date")
}

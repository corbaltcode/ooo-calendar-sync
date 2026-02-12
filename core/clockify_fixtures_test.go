package core

import "time"

func mkReq(id, tz, start, end string) ClockifyRequest {
	var r ClockifyRequest

	r.ID = id
	r.UserEmail = "fixture@example.com"
	r.UserTimeZone = tz
	r.CreatedAt = time.Date(2025, 12, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	r.PolicyName = "Vacation"

	r.TimeOffPeriod.Period.Start = start
	r.TimeOffPeriod.Period.End = end

	return r
}

func fixtureBadTimeZone() ClockifyRequest {
	return mkReq("fixture-bad-tz", "Not/A_Timezone", "2025-12-10T00:00:00Z", "2025-12-10T23:59:59Z")
}
func fixtureInvalidStartDate() ClockifyRequest {
	return mkReq("fixture-bad-start", "America/New_York", "not-a-date", "2025-12-10T23:59:59Z")
}
func fixtureInvalidEndDate() ClockifyRequest {
	return mkReq("fixture-bad-end", "America/New_York", "2025-12-10T00:00:00Z", "not-a-date")
}

package core

import (
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"
)

func handleAllDay(startUTC time.Time,
	endUTC time.Time,
	loc *time.Location) (time.Time, time.Time) {

	// Normalize to local dates
	startLocal := startUTC.In(loc)
	endLocal := endUTC.In(loc)

	y1, m1, d1 := startLocal.Date()
	y2, m2, d2 := endLocal.Date()

	// All-day local time window used for Events.List (TimeMin / TimeMax).
	allDayStart := time.Date(y1, m1, d1, 0, 0, 0, 0, loc)
	// Clockify is inclusive; GCal all-day is [start, end) exclusive.
	// So cover the last OOO day by adding +1 local day to the end date.
	allDayEndExclusive := time.Date(y2, m2, d2, 0, 0, 0, 0, loc).AddDate(0, 0, 1)

	return allDayStart, allDayEndExclusive
}

func handleHalfDay(
	halfDayHours *ClockifyTimePeriod,
	loc *time.Location,
) (time.Time, time.Time, error) {
	if halfDayHours == nil {
		return time.Time{}, time.Time{},
			fmt.Errorf("half-day request is missing halfDayHours")
	}

	startUTC, err := ParseTimeAny(halfDayHours.Start)
	if err != nil {
		return time.Time{}, time.Time{},
			fmt.Errorf("bad halfDayHours.start: %w", err)
	}

	endUTC, err := ParseTimeAny(halfDayHours.End)
	if err != nil {
		return time.Time{}, time.Time{},
			fmt.Errorf("bad halfDayHours.end: %w", err)
	}

	startLocal := startUTC.In(loc)
	endLocal := endUTC.In(loc)

	if !endLocal.After(startLocal) {
		return time.Time{}, time.Time{},
			fmt.Errorf("half-day end must be after start")
	}

	return startLocal, endLocal, nil
}

func calendarEventTimeValue(eventTime *calendar.EventDateTime) string {
	if eventTime == nil {
		return ""
	}

	if eventTime.DateTime != "" {
		return eventTime.DateTime
	}

	return eventTime.Date
}

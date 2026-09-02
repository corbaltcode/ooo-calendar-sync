package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/calendar/v3"
)

func TestHandleAllDay(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	tests := []struct {
		name      string
		startUTC  time.Time
		endUTC    time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name: "single-day request",
			startUTC: time.Date(
				2026, time.September, 3,
				4, 0, 0, 0,
				time.UTC,
			),
			endUTC: time.Date(
				2026, time.September, 4,
				3, 59, 59, 999000000,
				time.UTC,
			),
			wantStart: time.Date(
				2026, time.September, 3,
				0, 0, 0, 0,
				loc,
			),
			wantEnd: time.Date(
				2026, time.September, 4,
				0, 0, 0, 0,
				loc,
			),
		},
		{
			name: "multi-day request",
			startUTC: time.Date(
				2026, time.September, 3,
				4, 0, 0, 0,
				time.UTC,
			),
			endUTC: time.Date(
				2026, time.September, 7,
				3, 59, 59, 999000000,
				time.UTC,
			),
			wantStart: time.Date(
				2026, time.September, 3,
				0, 0, 0, 0,
				loc,
			),
			wantEnd: time.Date(
				2026, time.September, 7,
				0, 0, 0, 0,
				loc,
			),
		},
		{
			name: "request spanning daylight-saving transition",
			startUTC: time.Date(
				2026, time.March, 8,
				5, 0, 0, 0,
				time.UTC,
			),
			endUTC: time.Date(
				2026, time.March, 9,
				3, 59, 59, 999000000,
				time.UTC,
			),
			wantStart: time.Date(
				2026, time.March, 8,
				0, 0, 0, 0,
				loc,
			),
			wantEnd: time.Date(
				2026, time.March, 9,
				0, 0, 0, 0,
				loc,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := handleAllDay(
				tt.startUTC,
				tt.endUTC,
				loc,
			)

			assert.Equal(t, tt.wantStart, gotStart)
			assert.Equal(t, tt.wantEnd, gotEnd)
		})
	}
}

func TestHandleHalfDay(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	tests := []struct {
		name      string
		hours     *ClockifyTimePeriod
		wantStart time.Time
		wantEnd   time.Time
		wantError string
	}{
		{
			name: "converts half-day hours to local time",
			hours: &ClockifyTimePeriod{
				Start: "2026-09-03T13:00:00Z",
				End:   "2026-09-03T16:30:00Z",
			},
			wantStart: time.Date(
				2026, time.September, 3,
				9, 0, 0, 0,
				loc,
			),
			wantEnd: time.Date(
				2026, time.September, 3,
				12, 30, 0, 0,
				loc,
			),
		},
		{
			name:      "returns error when half-day hours are missing",
			hours:     nil,
			wantError: "half-day request is missing halfDayHours",
		},
		{
			name: "returns error when start is invalid",
			hours: &ClockifyTimePeriod{
				Start: "not-a-time",
				End:   "2026-09-03T16:30:00Z",
			},
			wantError: "bad halfDayHours.start",
		},
		{
			name: "returns error when end is invalid",
			hours: &ClockifyTimePeriod{
				Start: "2026-09-03T13:00:00Z",
				End:   "not-a-time",
			},
			wantError: "bad halfDayHours.end",
		},
		{
			name: "returns error when end equals start",
			hours: &ClockifyTimePeriod{
				Start: "2026-09-03T13:00:00Z",
				End:   "2026-09-03T13:00:00Z",
			},
			wantError: "half-day end must be after start",
		},
		{
			name: "returns error when end is before start",
			hours: &ClockifyTimePeriod{
				Start: "2026-09-03T16:30:00Z",
				End:   "2026-09-03T13:00:00Z",
			},
			wantError: "half-day end must be after start",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd, err := handleHalfDay(tt.hours, loc)

			if tt.wantError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
				assert.True(t, gotStart.IsZero())
				assert.True(t, gotEnd.IsZero())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantStart, gotStart)
			assert.Equal(t, tt.wantEnd, gotEnd)
		})
	}
}

func TestCalendarEventTimeValue(t *testing.T) {
	tests := []struct {
		name      string
		eventTime *calendar.EventDateTime
		want      string
	}{
		{
			name:      "returns empty string for nil event time",
			eventTime: nil,
			want:      "",
		},
		{
			name: "returns DateTime for timed event",
			eventTime: &calendar.EventDateTime{
				DateTime: "2026-09-03T09:00:00-04:00",
			},
			want: "2026-09-03T09:00:00-04:00",
		},
		{
			name: "returns Date for all-day event",
			eventTime: &calendar.EventDateTime{
				Date: "2026-09-03",
			},
			want: "2026-09-03",
		},
		{
			name:      "returns empty string when both values are empty",
			eventTime: &calendar.EventDateTime{},
			want:      "",
		},
		{
			name: "prefers DateTime when both values are populated",
			eventTime: &calendar.EventDateTime{
				Date:     "2026-09-03",
				DateTime: "2026-09-03T09:00:00-04:00",
			},
			want: "2026-09-03T09:00:00-04:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calendarEventTimeValue(tt.eventTime)

			assert.Equal(t, tt.want, got)
		})
	}
}

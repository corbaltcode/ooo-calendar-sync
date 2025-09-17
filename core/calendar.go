package core

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func InsertOOOEvents(ctx context.Context, jwtCfg *jwt.Config, requests []Request, calendarIDs []string) error {
	for _, r := range requests {
		// Load user's local timezone
		loc, err := time.LoadLocation(r.UserTimeZone)
		if err != nil {
			log.Printf("skip %s: unknown tz %q: %v", r.ID, r.UserTimeZone, err)
			continue
		}

		startUTC, err := ParseTimeAny(r.TimeOffPeriod.Period.Start)
		if err != nil {
			log.Printf("skip %s: bad period.start: %v", r.ID, err)
			continue
		}
		endUTC, err := ParseTimeAny(r.TimeOffPeriod.Period.End)
		if err != nil {
			log.Printf("skip %s: bad period.end: %v", r.ID, err)
			continue
		}

		// Normalize to local dates
		startLocal := startUTC.In(loc)
		endLocal := endUTC.In(loc)

		y1, m1, d1 := startLocal.Date()
		y2, m2, d2 := endLocal.Date()

		startDate := time.Date(y1, m1, d1, 0, 0, 0, 0, loc).Format("2006-01-02")
		// Clockify is inclusive; GCal all-day is [start, end) exclusive.
		// So cover the last OOO day by adding +1 local day to the end date.
		endDate := time.Date(y2, m2, d2, 0, 0, 0, 0, loc).AddDate(0, 0, 1).Format("2006-01-02")

		cfg := *jwtCfg
		cfg.Subject = r.UserEmail
		client := cfg.Client(ctx)

		srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			log.Printf("user %s: calendar service error: %v", r.UserEmail, err)
			continue
		}

		summary := "[TEST] OOO"
		if r.PolicyName != "" {
			summary = fmt.Sprintf("[TEST] OOO — %s", r.PolicyName)
		}
		ev := &calendar.Event{
			Summary:     summary,
			Description: fmt.Sprintf("Clockify request: %s\nCreatedAt: %s", r.ID, r.CreatedAt),
			Start:       &calendar.EventDateTime{Date: startDate},
			End:         &calendar.EventDateTime{Date: endDate}, // exclusive
			// Attaching the Clockify request ID as a private extended property.
			// TODO: Before inserting, check for an existing event with this key and insert event/skip:
			ExtendedProperties: &calendar.EventExtendedProperties{
				Private: map[string]string{
					"clockifyRequestId": r.ID,
				},
			},
		}

		// Insert into calendars
		for _, calID := range calendarIDs {
			_, err := srv.Events.Insert(calID, ev).Do()
			if err != nil {
				log.Printf("insert %s (user=%s cal=%s) failed: %v",
					r.ID, r.UserEmail, calID, err)
				continue
			}
			fmt.Printf("Inserted OOO for req=%s user=%s cal=%s (%s → %s)\n",
				r.ID, r.UserEmail, calID, startDate, endDate)
		}
	}
	return nil
}

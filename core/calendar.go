package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func InsertOOOEvent(ctx context.Context, jwtCfg *jwt.Config, r ClockifyRequest, calendarIDs []string) ([]GoogleCalendarEvent, error) {
	var syncedEvents []GoogleCalendarEvent
	var errs []error

	// Load user's local timezone
	loc, err := time.LoadLocation(r.UserTimeZone)
	if err != nil {
		log.Printf("skip %s: unknown tz %q: %v", r.ID, r.UserTimeZone, err)
		return nil, fmt.Errorf("req=%s user=%s: unknown tz %q: %w", r.ID, r.UserEmail, r.UserTimeZone, err)
	}

	startUTC, err := ParseTimeAny(r.TimeOffPeriod.Period.Start)
	if err != nil {
		log.Printf("skip %s: bad period.start: %v", r.ID, err)
		return nil, fmt.Errorf("req=%s user=%s: bad period.start: %w", r.ID, r.UserEmail, err)
	}
	endUTC, err := ParseTimeAny(r.TimeOffPeriod.Period.End)
	if err != nil {
		log.Printf("skip %s: bad period.end: %v", r.ID, err)
		return nil, fmt.Errorf("req=%s user=%s: bad period.end: %w", r.ID, r.UserEmail, err)
	}

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

	// YYYY-MM-DD string format is used for the Insert event payload.
	startDate := allDayStart.Format("2006-01-02")
	endDate := allDayEndExclusive.Format("2006-01-02")

	cfg := *jwtCfg
	cfg.Subject = r.UserEmail
	client := cfg.Client(ctx)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Printf("user %s: calendar service error: %v", r.UserEmail, err)
		return nil, fmt.Errorf("req=%s user=%s: calendar service error: %w", r.ID, r.UserEmail, err)
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
		existing, err := findClockifyEvents(
			ctx, srv, calID, r.ID,
			allDayStart, allDayEndExclusive,
		)
		if err != nil {
			log.Printf("lookup %s (user=%s cal=%s) failed: %v",
				r.ID, r.UserEmail, calID, err)
			errs = append(errs, fmt.Errorf("req=%s user=%s cal=%s: lookup failed: %w", r.ID, r.UserEmail, calID, err))
			continue
		}

		if len(existing) > 0 {
			for _, e := range existing {
				syncedEvents = append(syncedEvents, GoogleCalendarEvent{
					CalendarID: calID,
					EventID:    e.Id,
				})

				log.Printf(
					"FOUND existing OOO event for req=%s user=%s cal=%s eventId=%s (%s → %s)",
					r.ID,
					r.UserEmail,
					calID,
					e.Id,
					e.Start.Date,
					e.End.Date,
				)
			}

			continue
		}

		// No existing event
		insertedEvent, err := srv.Events.Insert(calID, ev).Do()
		if err != nil {
			log.Printf("insert %s (user=%s cal=%s) failed: %v",
				r.ID, r.UserEmail, calID, err)
			errs = append(errs, fmt.Errorf("req=%s user=%s cal=%s: insert failed: %w", r.ID, r.UserEmail, calID, err))
			continue
		}

		syncedEvents = append(syncedEvents, GoogleCalendarEvent{
			CalendarID: calID,
			EventID:    insertedEvent.Id,
		})

		log.Printf(
			"Inserted OOO for req=%s user=%s cal=%s (%s → %s)\n",
			r.ID, r.UserEmail, calID, startDate, endDate,
		)
	}

	return syncedEvents, errors.Join(errs...)
}

func DeleteOOOEvents(
	ctx context.Context,
	jwtCfg *jwt.Config,
	events []GoogleCalendarEvent,
) error {
	cfg := *jwtCfg
	client := cfg.Client(ctx)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))

	if err != nil {
		return fmt.Errorf("create calendar service: %w", err)
	}

	var errs []error

	for _, event := range events {
		err := srv.Events.
			Delete(event.CalendarID, event.EventID).
			Do()
		if err != nil {
			errs = append(
				errs,
				fmt.Errorf(
					"delete calendar event %s from calendar %s: %w",
					event.EventID,
					event.CalendarID,
					err,
				),
			)
			continue
		}

		log.Printf(
			"DELETED OOO event cal=%s eventId=%s",
			event.CalendarID,
			event.EventID,
		)
	}

	return errors.Join(errs...)
}

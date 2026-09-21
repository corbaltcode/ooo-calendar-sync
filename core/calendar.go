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

type RequestToProcess struct {
	Request        ClockifyRequest
	ExistingRecord *SyncedClockifyRequest
}

func InsertOOOEvents(ctx context.Context, jwtCfg jwt.Config, r ClockifyRequest, calendarIDs []string) ([]GoogleCalendarEvent, error) {
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

	var eventStart, eventEnd time.Time

	if r.TimeOffPeriod.HalfDay {
		eventStart, eventEnd, err = handleHalfDay(
			r.TimeOffPeriod.HalfDayHours,
			loc,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"req=%s user=%s: %w",
				r.ID,
				r.UserEmail,
				err,
			)
		}
	} else {
		eventStart, eventEnd, err = handleAllDay(startUTC, endUTC, loc)

		if err != nil {
			return nil, fmt.Errorf(
				"req=%s user=%s: %w",
				r.ID,
				r.UserEmail,
				err,
			)
		}

	}

	var calendarStart, calendarEnd *calendar.EventDateTime

	// Google Calendar uses RFC3339 DateTime values for timed events
	// and YYYY-MM-DD Date values for all-day events.
	if r.TimeOffPeriod.HalfDay {
		calendarStart = &calendar.EventDateTime{
			DateTime: eventStart.Format(time.RFC3339),
		}
		calendarEnd = &calendar.EventDateTime{
			DateTime: eventEnd.Format(time.RFC3339),
		}
	} else {
		calendarStart = &calendar.EventDateTime{
			Date: eventStart.Format("2006-01-02"),
		}
		calendarEnd = &calendar.EventDateTime{
			Date: eventEnd.Format("2006-01-02"),
		}
	}

	cfg := jwtCfg
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
		Start:       calendarStart,
		End:         calendarEnd,
		// Attaching the Clockify request ID as a private extended property.
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				"clockifyRequestId": r.ID,
			},
		},
	}

	// Insert into calendars
	for _, calID := range calendarIDs {
		existing, err := findClockifyEvents(
			srv, calID, r.ID,
			eventStart, eventEnd,
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
					calendarEventTimeValue(e.Start),
					calendarEventTimeValue(e.End),
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
			r.ID, r.UserEmail, calID, calendarEventTimeValue(calendarStart), calendarEventTimeValue(calendarEnd),
		)
	}

	return syncedEvents, errors.Join(errs...)
}

func DeleteOOOEvents(
	ctx context.Context,
	jwtCfg jwt.Config,
	userEmail string,
	events []GoogleCalendarEvent,
) error {
	cfg := jwtCfg
	cfg.Subject = userEmail
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

func SyncOOORequest(
	ctx context.Context,
	jwtCfg jwt.Config,
	req RequestToProcess,
	calendarIDs []string,
) ([]GoogleCalendarEvent, error) {
	switch req.Request.Status.StatusType {
	case ClockifyStatusApproved:
		return InsertOOOEvents(
			ctx,
			jwtCfg,
			req.Request,
			calendarIDs,
		)

	case ClockifyStatusRejected:
		if req.ExistingRecord == nil {
			return nil, fmt.Errorf(
				"cannot reject Clockify request %s: no existing synced record",
				req.Request.ID,
			)
		}

		err := DeleteOOOEvents(
			ctx,
			jwtCfg,
			req.Request.UserEmail,
			req.ExistingRecord.GoogleCalendarEvents,
		)
		if err != nil {
			return nil, err
		}

		return nil, nil

	default:
		return nil, fmt.Errorf(
			"unsupported Clockify request status %q",
			req.Request.Status.StatusType,
		)
	}
}

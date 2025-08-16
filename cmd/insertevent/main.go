package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()

	// Go get the service account credentials.
	b, err := os.ReadFile("service-account.json")
	if err != nil {
		log.Fatalf("Unable to read service account file: %v", err)
	}

	// Parse the service account credentials and configure for Google Calendar access
	config, err := google.JWTConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		log.Fatalf("Unable to parse service account credentials: %v", err)
	}

	// Our impersonated user
	config.Subject = "x-clockify-test@corbalt.com"

	// Create an HTTP client authorized as the impersonated user
	client := config.Client(ctx)

	// Using our client, create a calendar service.
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Calendar client: %v", err)
	}

	// Here's some event details
	startDate := "2025-11-26"
	endDate := "2025-11-26"

	event := &calendar.Event{
		Summary:     "[TEST] OOO",
		Description: "OOO test event inserted via service account",
		Start: &calendar.EventDateTime{
			Date: startDate,
		},
		End: &calendar.EventDateTime{
			Date: endDate,
		},
	}

	calendarIDs := []string{
		"primary",
		// Add other calendars
	}

	for _, calID := range calendarIDs {
		_, err := srv.Events.Insert(calID, event).Do()
		if err != nil {
			log.Printf("Failed to insert into %s: %v", calID, err)
		} else {
			fmt.Printf("Inserted OOO event into %s\n", calID)
		}
	}
}

package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/corbaltcode/ooo-calendar-sync/core"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}

	store := core.NewDynamoStore(
		dynamodb.NewFromConfig(cfg),
		"ooo-calendar-sync-requests",
	)

	item := &core.SyncedClockifyRequest{
		ClockifyRequestID: "test-request-123",
		UserEmail:         "test@example.com",
		Status:            "APPROVED",
		SyncState:         "pending",
		LastSeenAt:        "2026-06-20T00:00:00Z",
	}

	// if err := store.PutSyncedRequest(ctx, item); err != nil {
	// 	log.Fatalf("put item: %v", err)
	// }

	// log.Println("successfully wrote item")

	if err := store.DeleteSyncedRequest(ctx, item.ClockifyRequestID); err != nil {
		log.Fatalf("delete item: %v", err)
	}

	log.Println("successfully deleted item")
}

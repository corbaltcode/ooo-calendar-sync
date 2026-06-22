package core

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoStore struct {
	Client    *dynamodb.Client
	TableName string
}

func NewDynamoStore(client *dynamodb.Client, tableName string) *DynamoStore {
	return &DynamoStore{
		Client:    client,
		TableName: tableName,
	}
}

func (s *DynamoStore) PutSyncedRequest(ctx context.Context, item *SyncedClockifyRequest) error {
	if item.ClockifyRequestID == "" {
		return errors.New("missing Clockify request ID")
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}

	_, err = s.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.TableName,
		Item:      av,
	})

	return err
}

func (s *DynamoStore) DeleteSyncedRequest(
	ctx context.Context,
	clockifyRequestID string,
) error {
	if clockifyRequestID == "" {
		return errors.New("missing Clockify request ID")
	}

	_, err := s.Client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &s.TableName,
		Key: map[string]types.AttributeValue{
			"ClockifyRequestId": &types.AttributeValueMemberS{
				Value: clockifyRequestID,
			},
		},
	})

	return err
}

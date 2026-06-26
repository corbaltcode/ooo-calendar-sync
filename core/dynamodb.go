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

func (s *DynamoStore) GetSyncedRequest(
	ctx context.Context,
	clockifyRequestID string,
) (*SyncedClockifyRequest, error) {
	if clockifyRequestID == "" {
		return nil, errors.New("missing Clockify request ID")
	}

	response, err := s.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.TableName,
		Key: map[string]types.AttributeValue{
			"ClockifyRequestId": &types.AttributeValueMemberS{
				Value: clockifyRequestID,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if len(response.Item) == 0 {
		return nil, nil
	}

	var item SyncedClockifyRequest

	if err := attributevalue.UnmarshalMap(response.Item, &item); err != nil {
		return nil, err
	}

	return &item, nil
}

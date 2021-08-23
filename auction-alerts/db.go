package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var SERVICE_NAME = "AuctionAlerts"

type AlertTime struct {
	ServiceName string    `json:"ServiceName"`
	RpcEndpoint string    `json:"RpcEndpoint"`
	Timestamp   time.Time `json:"Timestamp"`
}

type Db struct {
	svc *dynamodb.Client
}

func NewDb() (Db, error) {
	awsCfg, err := aws_config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return Db{}, err
	}

	return Db{dynamodb.NewFromConfig(awsCfg)}, nil
}

/// Gets the latest alert time
/// Returns AlertTime, false if item not found, error
func (db *Db) GetLatestAlert(tableName string, rpcUrl string) (AlertTime, bool, error) {
	lastAlert, err := db.svc.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"ServiceName": &types.AttributeValueMemberS{Value: SERVICE_NAME},
			"RpcEndpoint": &types.AttributeValueMemberS{Value: rpcUrl},
		},
	})
	if err != nil {
		return AlertTime{}, false, err
	}

	// Previous time set, check if within alert frequency
	if lastAlert.Item != nil {
		item := AlertTime{}
		if err := attributevalue.UnmarshalMap(lastAlert.Item, &item); err != nil {
			return AlertTime{}, false, nil
		}

		return item, true, nil
	}

	// Return default if no items found
	return AlertTime{}, false, nil
}

func (db *Db) SaveAlert(tableName string, rpcUrl string, d time.Time) (*dynamodb.PutItemOutput, error) {
	return db.svc.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"ServiceName": &types.AttributeValueMemberS{Value: SERVICE_NAME},
			"RpcEndpoint": &types.AttributeValueMemberS{Value: rpcUrl},
			"Timestamp":   &types.AttributeValueMemberS{Value: d.Format(time.RFC3339)},
		},
	})
}

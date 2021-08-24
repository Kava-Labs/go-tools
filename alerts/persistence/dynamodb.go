package persistence

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// AlertTime defines a previous alert time for a specific service and network
type AlertTime struct {
	ServiceName string    `json:"ServiceName"`
	RpcEndpoint string    `json:"RpcEndpoint"`
	Timestamp   time.Time `json:"Timestamp"`
}

// Db wraps a DynamoDB client to provide simple functions to get and save alerts
type Db struct {
	svc *dynamodb.Client
}

// Verify interface compliance at compile time
var _ AlertPersister = (*Db)(nil)

// NewDb returns a db with the AWS configuration initialized
func NewDb() (Db, error) {
	awsCfg, err := aws_config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return Db{}, err
	}

	return Db{dynamodb.NewFromConfig(awsCfg)}, nil
}

// GetLatestAlert returns the latest alert time and if the item was found
func (db *Db) GetLatestAlert(tableName string, serviceName string, rpcUrl string) (AlertTime, bool, error) {
	lastAlert, err := db.svc.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"ServiceName": &types.AttributeValueMemberS{Value: serviceName},
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

// SaveAlert saves an alert for a given RpcEndpoint
func (db *Db) SaveAlert(tableName string, serviceName string, rpcUrl string, d time.Time) error {
	_, err := db.svc.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"ServiceName": &types.AttributeValueMemberS{Value: serviceName},
			"RpcEndpoint": &types.AttributeValueMemberS{Value: rpcUrl},
			"Timestamp":   &types.AttributeValueMemberS{Value: d.Format(time.RFC3339)},
		},
	})

	return err
}

package auth

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// DynamoDBTables is a map whose keys are DynamoDB table names and whose values are `dynamodb.CreateTableInput` instances.
var DynamoDBTables = map[string]*dynamodb.CreateTableInput{
	"AccessCodes": &dynamodb.CreateTableInput{
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("Code"),
				KeyType:       aws.String("HASH"), // partition key
			},
		},
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("Code"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("LastUpdate"),
				AttributeType: aws.String("N"),
			},
			{
				AttributeName: aws.String("Created"),
				AttributeType: aws.String("N"),
			},
			{
				AttributeName: aws.String("Expires"),
				AttributeType: aws.String("N"),
			},
		},
		GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
			{
				IndexName: aws.String("updated"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("Code"),
						KeyType:       aws.String("HASH"),
					},
					{
						AttributeName: aws.String("LastUpdate"),
						KeyType:       aws.String("RANGE"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: aws.String("ALL"),
				},
			},
			{
				IndexName: aws.String("created"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("Code"),
						KeyType:       aws.String("HASH"),
					},
					{
						AttributeName: aws.String("Created"),
						KeyType:       aws.String("RANGE"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: aws.String("ALL"),
				},
			},
			{
				IndexName: aws.String("expires"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("Code"),
						KeyType:       aws.String("HASH"),
					},
					{
						AttributeName: aws.String("Expires"),
						KeyType:       aws.String("RANGE"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: aws.String("ALL"),
				},
			},
		},
		BillingMode: aws.String("PAY_PER_REQUEST"),
		// TableName:   set inline below
	},
}

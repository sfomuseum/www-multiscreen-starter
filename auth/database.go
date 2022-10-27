package auth

import (
	"context"
	"fmt"
	aa_dynamodb "github.com/aaronland/go-aws-dynamodb"
	aa_session "github.com/aaronland/go-aws-session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"gocloud.dev/docstore"
	"gocloud.dev/docstore/awsdynamodb"
	"io"
	"net/url"
)

// NewAccessCodeDatabase returns a new `docstore.Collection` instance derived from 'database_uri'.
func NewAccessCodesDatabase(ctx context.Context, database_uri string) (*docstore.Collection, error) {

	var db *docstore.Collection

	db_u, err := url.Parse(database_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse '%s', %w", database_uri, err)
	}

	// We need to special-case dynamodb stuff in order to account for queries
	// queries with ordering requirements. Table definitions are handled in
	// cmd/dynamodb_tables/main.go

	if db_u.Scheme == "awsdynamodb" {

		// Connect local dynamodb using Golang
		// https://gist.github.com/Tamal/02776c3e2db7eec73c001225ff52e827
		// https://gocloud.dev/howto/docstore/#dynamodb-ctor

		table := db_u.Host

		db_q := db_u.Query()

		partition_key := db_q.Get("partition_key")
		region := db_q.Get("region")
		endpoint := db_q.Get("endpoint")

		credentials := db_q.Get("credentials")

		cfg, err := aa_session.NewConfigWithCredentialsAndRegion(credentials, region)

		if err != nil {
			return nil, fmt.Errorf("Failed to create new session for credentials '%s', %w", credentials, err)
		}

		if endpoint != "" {
			cfg.Endpoint = aws.String(endpoint)
		}

		sess, err := session.NewSession(cfg)

		if err != nil {
			return nil, fmt.Errorf("Failed to create AWS session, %w", err)
		}

		// START OF create dynamodb tables if necessary
		// Unfortunately there's nothing about creating tables in the Go Cloud abstractions

		client := dynamodb.New(sess)

		table_opts := &aa_dynamodb.CreateTablesOptions{
			Tables:  DynamoDBTables,
			Refresh: false,
		}

		err = aa_dynamodb.CreateTables(client, table_opts)

		if err != nil {
			return nil, fmt.Errorf("Failed to create DynamoDB tables, %w", err)
		}

		// END OF create dynamodb tables if necessary

		// START OF necessary for order by created/lastupdate dates
		// https://pkg.go.dev/gocloud.dev@v0.23.0/docstore/awsdynamodb#InMemorySortFallback

		create_func := func() interface{} {
			return &RelayCode{}
		}

		fallback_func := awsdynamodb.InMemorySortFallback(create_func)

		opts := &awsdynamodb.Options{
			AllowScans:       true,
			RunQueryFallback: fallback_func,
		}

		// END OF necessary for order by created/lastupdate dates

		col, err := awsdynamodb.OpenCollection(dynamodb.New(sess), table, partition_key, "", opts)

		if err != nil {
			return nil, fmt.Errorf("Failed to open collection, %w", err)
		}

		db = col

	} else {

		col, err := docstore.OpenCollection(ctx, database_uri)

		if err != nil {
			return nil, fmt.Errorf("Failed to create database for '%s', %w", database_uri, err)
		}

		db = col

	}

	return db, nil
}

// PruneAccessCodesDatabase will remove an access codes from 'db' whosse expiry time is less than 'expires'.
func PruneAccessCodesDatabase(ctx context.Context, db *docstore.Collection, expires int64) error {

	q := db.Query()
	q = q.Where("Expires", "<", expires)

	iter := q.Get(ctx)
	defer iter.Stop()

	for {

		select {
		case <-ctx.Done():
			return nil
		default:
			// pass
		}

		var rc RelayCode
		err := iter.Next(ctx, &rc)

		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("Failed to interate, %w", err)
		} else {
			// fmt.Printf("%s: %d\n", rc.Code, rc.Expires)
		}

		err = db.Delete(ctx, &rc)

		if err != nil {
			return fmt.Errorf("Failed to delete access code '%s', %v", rc.Code, err)
		}
	}

	return nil
}

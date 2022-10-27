// server implements a HTTP server for ...
package main

import (
	_ "gocloud.dev/docstore/awsdynamodb"
	_ "gocloud.dev/docstore/memdocstore"
	_ "gocloud.dev/pubsub/mempubsub"
)

import (
	"context"
	app "github.com/sfomuseum/www-multiscreen-starter/app/server"
	"log"
)

func main() {

	ctx := context.Background()
	logger := log.Default()

	err := app.Run(ctx, logger)

	if err != nil {
		logger.Fatalf("Failed to run application, %v", err)
	}
}

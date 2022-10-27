package server

import (
	"context"
	"flag"
	"fmt"
	"github.com/rs/cors"
	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-pubsub/publisher"
	"github.com/sfomuseum/go-pubsub/subscriber"
	"github.com/sfomuseum/www-multiscreen-starter/auth"
	"github.com/sfomuseum/www-multiscreen-starter/http"
	"github.com/sfomuseum/www-multiscreen-starter/sse"
	"github.com/sfomuseum/www-multiscreen-starter/static/controller"
	"github.com/sfomuseum/www-multiscreen-starter/static/receiver"
	"github.com/whosonfirst/go-pubssed/broker"
	"log"
	gohttp "net/http"
	"os"
	"os/signal"
	"time"
)

// Run will start the multiscreen webserver using the flagset defined by the `DefaultFlagSet` method.
func Run(ctx context.Context, logger *log.Logger) error {
	fs := DefaultFlagSet()
	return RunWithFlagSet(ctx, fs, logger)
}

// Run will start the multiscreen webserver using 'fs'.
func RunWithFlagSet(ctx context.Context, fs *flag.FlagSet, logger *log.Logger) error {

	flagset.Parse(fs)

	err := flagset.SetFlagsFromEnvVars(fs, "RELAY")

	if err != nil {
		return fmt.Errorf("Failed to set flags from env vars, %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ws_pub, err := publisher.NewPublisher(ctx, publisher_uri)

	if err != nil {
		return fmt.Errorf("Failed to create new publisher for '%s', %v", publisher_uri, err)
	}

	defer ws_pub.Close()

	sse_sub, err := subscriber.NewSubscriber(ctx, subscriber_uri)

	if err != nil {
		return fmt.Errorf("Failed to create subscriber for '%s', %v", subscriber_uri, err)
	}

	defer sse_sub.Close()

	// Set up the docstore.Collection for storing access tokens

	db, err := auth.NewAccessCodesDatabase(ctx, database_uri)

	if err != nil {
		return fmt.Errorf("Failed to create access codes database for '%s', %v", database_uri, err)
	}

	defer db.Close()

	// Prune all previous access code

	now := time.Now()
	ts := now.Unix()

	err = auth.PruneAccessCodesDatabase(ctx, db, ts)

	if err != nil {
		return fmt.Errorf("Failed to prune access codes, %v", err)
	}

	// Create a new access code and start a timer to refresh them every (n) seconds

	_, err = auth.NewRelayCodeWithCollection(ctx, db, ttl)

	if err != nil {
		return fmt.Errorf("Failed to create new relay code, %v", err)
	}

	//app_log.Printf("Starting access code is '%s'\n", rc.Code)

	// Set up a time to prune old access codes in the background

	go func(ctx context.Context) {

		ticker := time.NewTicker(time.Duration(2) * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:

				ts := now.Unix()
				expires := ts - int64(ttl)

				err := auth.PruneAccessCodesDatabase(ctx, db, expires)

				if err != nil {
					logger.Printf("Failed to prune access codes, %v", err)
				}

			}
		}

	}(ctx)

	// Set up a timer to mint new access codes in the background

	go func(ctx context.Context) {

		new_code := func(ctx context.Context, now time.Time) {

			ts := now.Unix()

			current_code, err := auth.CurrentRelayCodeWithCollection(ctx, db, ttl)

			if err != nil {
				logger.Printf("Unable to determine current access code, %v", err)
			}

			if current_code != nil && current_code.Expires > ts {
				logger.Printf("There is an unexpired access code %s (%d) already in use", current_code.Code, current_code.Expires)
				return
			}

			rc, err := auth.NewRelayCodeWithCollection(ctx, db, ttl)

			if err != nil {
				logger.Printf("Failed to create new relay code, %v", err)
				return
			}

			msg := sse.NewAccessCodeMessage(rc)
			err = msg.Publish(ctx, ws_pub)

			if err != nil {
				logger.Printf("Failed to publish relay code, %v", err)
				return
			}

			fmt.Printf("Reset access code '%s'\n", rc.Code)
			logger.Printf("Reset access code\n")
		}

		now := time.Now()
		new_code(ctx, now)

		ticker := time.NewTicker(time.Duration(ttl) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:

				new_code(ctx, now)
			}
		}

	}(ctx)

	// Start building the HTTP endpoints

	mux := gohttp.NewServeMux()

	pong_wait := 60 * time.Second
	ping_period := 30 * time.Second // (pong_wait * 9) / 10
	write_wait := 30 * time.Second  // this is very long...

	check_origin := func(req *gohttp.Request) bool {
		return true
	}

	ws_opts := &http.WebsocketHandlerOptions{
		Publisher:   ws_pub,
		Database:    db,
		PingPeriod:  ping_period,
		PongWait:    pong_wait,
		WriteWait:   write_wait,
		Logger:      logger,
		CheckOrigin: check_origin,
	}

	//

	ws_handler, err := http.WebsocketHandler(ws_opts)

	if err != nil {
		return fmt.Errorf("Failed to create websocket handler, %v", err)
	}

	mux.Handle("/ws/", ws_handler)

	// SSE endpoint - this is where the target (iPad) will listen for updates
	// See notes above about "publishers" and Redis

	sse_broker, err := broker.NewBroker()

	if err != nil {
		return fmt.Errorf("Failed to create SSE broker, %v", err)
	}

	sse_broker.Logger = logger

	err = sse_broker.Start(ctx, sse_sub)

	if err != nil {
		return fmt.Errorf("Failed to start SSE broker, %v", err)
	}

	sse_handler_ttl := time.Duration(sse_ttl) * time.Second

	sse_handler, err := sse_broker.HandlerFuncWithTimeout(&sse_handler_ttl)

	if err != nil {
		return fmt.Errorf("Failed to create SSE handler, %v", err)
	}

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})

	// Note the part where we need to explicitly type the
	// result as a HandlerFunc - I wish rs/cors just did
	// this for us but it doesn't.

	sse_handler = c.Handler(sse_handler).(gohttp.HandlerFunc)

	mux.HandleFunc("/sse/", sse_handler)

	code_opts := &http.AccessCodeHandlerOptions{
		Database:  db,
		Publisher: ws_pub,
		Logger:    logger,
		TTL:       ttl,
	}

	code_handler, err := http.AccessCodeHandler(code_opts)

	if err != nil {
		return fmt.Errorf("Failed to create access code handler, %v", err)
	}

	code_handler = c.Handler(code_handler).(gohttp.HandlerFunc)
	mux.Handle("/code/", code_handler)

	// Controller (index) webpage

	http_fs := gohttp.FS(controller.FS)
	fs_handler := gohttp.FileServer(http_fs)

	mux.Handle("/", fs_handler)

	// Receiver webpage

	if enable_receiver {

		http_fs := gohttp.FS(receiver.FS)
		fs_handler := gohttp.FileServer(http_fs)
		fs_handler = gohttp.StripPrefix("/receiver", fs_handler)

		mux.Handle("/receiver/", fs_handler)
	}

	// Start the server

	// https://medium.com/khanakia/go-1-16-signal-notifycontext-fac21b3eaa1c
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	addr := fmt.Sprintf("%s:%d", host, port)

	server := &gohttp.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {

		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		server.Shutdown(ctx)
	}()

	logger.Printf("Listening on %s\n", addr)
	err = server.ListenAndServe()

	if err != nil {
		return fmt.Errorf("Failed to serve requests, %v", err)
	}

	return nil
}

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/sfomuseum/go-pubsub/publisher"
	"github.com/sfomuseum/www-multiscreen-starter/auth"
	"github.com/sfomuseum/www-multiscreen-starter/sse"
	"github.com/sfomuseum/www-multiscreen-starter/ws"
	"gocloud.dev/docstore"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// WebsocketHandlerOptions defines a struct containing configuration options for use by
// the http.Handler return by the WebsocketHandler method.
type WebsocketHandlerOptions struct {
	// A valid publisher.Publisher instance used to relay messages sent to the Websocket endpoint.
	Publisher publisher.Publisher
	// A valid docstore.Collection instance where access codes will be stored and retrieved from.
	Database *docstore.Collection
	// The amount of time to allow for Websocket pong requests.
	PongWait time.Duration
	// The amount of time to allow for Websocket ping requests.
	PingPeriod time.Duration
	// The amount of time to allow Websocket write operations to complete.
	WriteWait time.Duration
	// A custom "check origin" function to pass to the gorilla/websocket.Upgrader method.
	CheckOrigin func(r *http.Request) bool
	// A valid *log.Logger  instance
	Logger *log.Logger
}

// WebsocketHandler returns an http.Handler for serving Websocket requests.
func WebsocketHandler(opts *WebsocketHandlerOptions) (http.Handler, error) {

	// Note the way we are assigning a custom "check origin" function
	// which may be nil
	// https://pkg.go.dev/github.com/gorilla/websocket#Upgrader.CheckOrigin

	upgrader := websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
		CheckOrigin:     opts.CheckOrigin,
	}

	mu := new(sync.RWMutex)

	fn := func(rsp http.ResponseWriter, req *http.Request) {

		log.Println("WS Connect")

		if req.Method != "GET" {
			http.Error(rsp, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := req.Context()
		ctx, cancel := context.WithCancel(ctx)

		defer cancel()

		conn, err := upgrader.Upgrade(rsp, req, nil)

		if err != nil {
			LogWithRequest(opts.Logger, req, "Failed to upgrade websocket connection, %v", err)
			msg := fmt.Sprintf("Unable to upgrade to websockets")
			http.Error(rsp, msg, http.StatusBadRequest)
			return
		}

		// The Close and WriteControl methods can be called concurrently with all other methods.
		// https://pkg.go.dev/github.com/gorilla/websocket

		defer conn.Close()

		// START OF ...
		// https://github.com/gorilla/websocket/blob/master/examples/filewatch/main.go

		conn.SetReadLimit(512)

		conn.SetReadDeadline(time.Now().Add(opts.PongWait))

		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(opts.PongWait))
			return nil
		})

		ping_ticker := time.NewTicker(opts.PingPeriod)
		defer ping_ticker.Stop()

		// These are here to (try and) prevent AWS ELB timeouts

		go func() {

			for {
				select {
				case <-ctx.Done():
					return
				case <-ping_ticker.C:

					select {
					case <-ctx.Done():
						return
					default:
						// pass
					}

					go func() {

						conn.SetWriteDeadline(time.Now().Add(opts.WriteWait))

						mu.Lock()
						defer mu.Unlock()

						err := conn.WriteMessage(websocket.PingMessage, []byte{})

						if err != nil {
							LogWithRequest(opts.Logger, req, "Failed to send WS ping message, %v\n", err)
						}
					}()
				}
			}

		}()

		// END OF ...

		for {

			select {
			case <-ctx.Done():
				break
			default:
				// pass
			}

			mt, data, err := conn.ReadMessage()

			if err != nil {

				// https://pkg.go.dev/github.com/gorilla/websocket#pkg-constants
				//
				// This is sent in javascript/t2.controller.js after receiving
				// an "expired" message (below)

				// https://stackoverflow.com/questions/61108552/go-websocket-error-close-1006-abnormal-closure-unexpected-eof

				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) || err == io.EOF {
					msg := fmt.Sprintf("WS connection closed 2, %v", err)
					LogWithRequest(opts.Logger, req, msg)
					break
				}

				if err != nil {
					LogWithRequest(opts.Logger, req, "Unexpected error reading message, %v", err)
					break
				}
			}

			switch mt {
			case websocket.TextMessage:

				br := bytes.NewReader(data)

				var update_msg *ws.UpdateMessage

				dec := json.NewDecoder(br)
				err := dec.Decode(&update_msg)

				if err != nil {
					LogWithRequest(opts.Logger, req, "Failed to decode message, %v", err)
					continue
				}

				if update_msg.Type == "ping" {

					go func() {

						mu.Lock()
						defer mu.Unlock()

						err := conn.WriteMessage(websocket.TextMessage, []byte("pong"))

						if err != nil {
							LogWithRequest(opts.Logger, req, "Failed to send WS pong message, %v", err)
						}
					}()

					continue
				}

				LogWithRequest(opts.Logger, req, "Received '%s' message (%s)\n", update_msg.Type, update_msg.Code)

				// START OF check relay code

				if opts.Database != nil {

					// log.Printf("Validate code")

					update_code := &auth.RelayCode{
						Code: update_msg.Code,
					}

					err := opts.Database.Get(ctx, update_code)

					if err != nil {

						LogWithRequest(opts.Logger, req, "Failed to get %s, %v", update_msg.Code, err)

						go func() {

							mu.Lock()
							defer mu.Unlock()

							conn.SetWriteDeadline(time.Now().Add(opts.WriteWait))
							err := conn.WriteMessage(websocket.TextMessage, []byte("invalid"))

							if err != nil {
								LogWithRequest(opts.Logger, req, "Failed to send invalid notice for '%s', %v\n", update_msg.Code, err)
							}
						}()

						break
					}

					// Note the way we are returning the results in ascending order
					// This is to ensure that the "last update" checks below always
					// work and don't get unintentionally reset by an updated access
					// code which is (2+) steps ahead of an expired code but hasn't
					// been used yet.

					q := opts.Database.Query()
					q = q.Where("Created", ">", update_code.Created)

					// query requires a table scan, but has an ordering requirement;
					//  add an index or provide Options.RunQueryFallback (code=Unimplemented)
					q = q.OrderBy("Created", docstore.Ascending)

					iter := q.Get(ctx)
					defer iter.Stop()

					// other_code can't be a pointer without freaking out the Docstore code
					// so we can't test it for 'nil' below.

					var other_code auth.RelayCode
					err = iter.Next(ctx, &other_code)

					// Query failed - io.EOF is equivalent of "no rows"

					if err != nil && err != io.EOF {

						LogWithRequest(opts.Logger, req, "Bunk next, %v", err)

						go func() {

							mu.Lock()
							defer mu.Unlock()

							conn.SetWriteDeadline(time.Now().Add(opts.WriteWait))
							err := conn.WriteMessage(websocket.TextMessage, []byte("invalid"))

							if err != nil {
								LogWithRequest(opts.Logger, req, "Failed to send invalid notice for '%s', %v\n", update_msg.Code, err)
							}
						}()

						break
					}

					// There is a newer code. If it's in use then this code is no longer
					// valid and we drop the update on the floor

					if other_code.Code != "" {

						if other_code.LastUpdate > update_code.Created {

							LogWithRequest(opts.Logger, req, "Code '%s' has expired and another code ('%s') is in use\n", update_msg.Code, other_code.Code)

							go func() {

								mu.Lock()
								defer mu.Unlock()

								conn.SetWriteDeadline(time.Now().Add(opts.WriteWait))
								err := conn.WriteMessage(websocket.TextMessage, []byte("expired"))

								if err != nil {
									LogWithRequest(opts.Logger, req, "Failed to send expiry notice for '%s', %v\n", update_msg.Code, err)
								}
							}()

							continue
						}
					}

					// This code hasn't been used yet so send a message to hide the
					// QR code

					// log.Println("DEBUG", update_code.LastUpdate)

					if update_code.LastUpdate == 0 {

						go func(ctx context.Context) {

							msg := sse.NewHideCodeMessage()
							err := msg.Publish(ctx, opts.Publisher)

							if err != nil {
								LogWithRequest(opts.Logger, req, "Failed to publish message, %v", err)
								return
							}

						}(ctx)
					}

					// Set last update for the current code

					now := time.Now()
					ts := now.Unix()

					// log.Printf("Set last update %d\n", ts)

					mod := docstore.Mods{"LastUpdate": ts}
					err = opts.Database.Update(ctx, update_code, mod)

					if err != nil {
						LogWithRequest(opts.Logger, req, "Failed to set last update for '%s', %v", update_code.Code, err)
					}
				}

				// END OF check relay code

				// Finally send the update down to the receiver

				go func(ctx context.Context, update_msg *ws.UpdateMessage) {

					// log.Printf("WS RELAY '%s'\n", string(data))

					msg := sse.NewMessageFromUpdate(update_msg)
					err := msg.Publish(ctx, opts.Publisher)

					if err != nil {
						LogWithRequest(opts.Logger, req, "Failed to publish message, %v", err)
						return
					}

					mu.Lock()
					defer mu.Unlock()

					conn.WriteMessage(websocket.TextMessage, []byte("relay"))

				}(ctx, update_msg)

			default:
				// pass
			}
		}

		mu.Lock()
		defer mu.Unlock()

		conn.WriteMessage(websocket.CloseMessage, []byte{})
	}

	h := http.HandlerFunc(fn)
	return h, nil
}

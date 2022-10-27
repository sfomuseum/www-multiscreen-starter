// Package sse provides methods for creating and dispatching Server-Sent Event (SSE) messages.
package sse

import (
	"context"
	"encoding/json"
	"github.com/sfomuseum/go-pubsub/publisher"
	"github.com/sfomuseum/www-multiscreen-starter/ws"
	_ "log"
)

// type SSEMessage is a struct used to dispatch messages to SSE endpoints.
type SSEMessage struct {
	Type string      `json:"type"` // make this an iota
	Data interface{} `json:"data"`
}

// Publish a message to a publisher.Publisher instance.
func (msg *SSEMessage) Publish(ctx context.Context, p publisher.Publisher) error {

	enc_msg, err := json.Marshal(msg)

	if err != nil {
		return err
	}

	str_msg := string(enc_msg)
	// log.Printf("SSE PUBLISH '%s'\n", enc_msg)
	return p.Publish(ctx, str_msg)
}

// Create a new SSE message for updating the current access code.
func NewAccessCodeMessage(data interface{}) *SSEMessage {

	msg := &SSEMessage{
		Type: "showCode",
		Data: data,
	}

	return msg
}

// Create a new SSE message to indicate that the QR (access) code should be hidden.
func NewHideCodeMessage() *SSEMessage {

	msg := &SSEMessage{
		Type: "hideCode",
	}

	return msg
}

// Empty "ping"-style message to send clients in order to prevent
// AWS ELB connection timeouts (generally 60 seconds)
func NewPingMessage() *SSEMessage {

	msg := &SSEMessage{
		Type: "ping",
	}

	return msg
}

// Create a new SSE message for a ws.UpdateMessage instance (messages sent by the web application over a WebSocket connection).
func NewMessageFromUpdate(update *ws.UpdateMessage) *SSEMessage {

	msg := &SSEMessage{
		Type: update.Type,
		// Note that this used to be just Data: updateBody
		// but because of the way that SSE messages are being
		// decoded in ios-multiscreen-starter it is necessary to
		// pass a dictionary.
		Data: map[string]interface{}{"body": update.Body},
	}

	return msg
}

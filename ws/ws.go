// Package ws (websockets) provides methods for working with messages over WebSocket connections.
package ws

// type UpdateMessage is the common structure for all messages relayed over WebSocket connections.
type UpdateMessage struct {
	// Type is the type of message being sent.
	Type string `json:"type"`
	// Code is a numeric status code associated with the message being sent.
	Code string `json:"code"`
	// Body is any additional detail associated with the message.
	Body interface{} `json:"body"`
}

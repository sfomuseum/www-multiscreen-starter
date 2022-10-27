package server

import (
	"flag"
	"github.com/sfomuseum/go-flags/flagset"
)

// The host name to listen for requests on.
var host string

// The port number to listen for requests on.
var port int

// A valid sfomuseum/go-pubsub/publisher URI.
var publisher_uri string

// A valid sfomuseum/go-pubsub/publisher URI.
var subscriber_uri string

// A valid gocloud.dev/docstore URI.
var database_uri string

// The time-to-live in number of seconds for access codes.
var ttl int

// The number of seconds to allow SSE connections to stay open.
var sse_ttl int

// Enable a /receiver endpoint on the web server. Used for debugging.
var enable_receiver bool

// DefaultFlagSet returns a `*flag.FlagSet` with default flags for starting to multiscreen webserver.
func DefaultFlagSet() *flag.FlagSet {

	fs := flagset.NewFlagSet("relay")

	fs.StringVar(&host, "host", "localhost", "The host name to listen for requests on.")
	fs.IntVar(&port, "port", 8080, "The port number to listen for requests on.")

	fs.StringVar(&publisher_uri, "publisher-uri", "mem://pubssed", "A valid sfomuseum/go-pubsub/publisher URI.")
	fs.StringVar(&subscriber_uri, "subscriber-uri", "mem://pubssed", "A valid sfomuseum/go-pububs/subscriber URI.")

	fs.StringVar(&database_uri, "database-uri", "mem://access/Code", "A valid gocloud.dev/docstore URI.")

	fs.IntVar(&ttl, "access-code-ttl", 300, "The time-to-live in number of seconds for access codes.")

	fs.IntVar(&sse_ttl, "sse-handler-ttl", 1200, "The number of seconds to allow SSE connections to stay open.")

	fs.BoolVar(&enable_receiver, "enable-receiver", false, "Enable a /receiver endpoint on the web server. Used for debugging.")
	return fs
}

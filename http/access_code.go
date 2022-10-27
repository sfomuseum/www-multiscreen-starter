package http

import (
	"github.com/sfomuseum/go-pubsub/publisher"
	"github.com/sfomuseum/www-multiscreen-starter/auth"
	"github.com/sfomuseum/www-multiscreen-starter/sse"
	"gocloud.dev/docstore"
	"log"
	"net/http"
	"time"
)

// AccessCodeHandlerOption defines a struct containing configuration options for the
// AccessCodeHandler http.Handler
type AccessCodeHandlerOptions struct {
	// A valid sfomuseum/go-pubsub/publisher.Publisher for broadcasting events.
	Publisher publisher.Publisher
	// A valid gocloud.dev/docstore.Collection instance for storing and retrieving access codes.
	Database *docstore.Collection
	// A valid *log.Logger  instance
	Logger *log.Logger
	// The time to live for access codes
	TTL int
}

// AccessCodeHandler returns an HTTP handler that will attempt to retrieve the most
// recent access code and dispatch to the application's Publisher instance.
func AccessCodeHandler(opts *AccessCodeHandlerOptions) (http.Handler, error) {

	fn := func(rsp http.ResponseWriter, req *http.Request) {

		now := time.Now()
		ts := now.Unix()

		ctx := req.Context()

		q := opts.Database.Query()
		q = q.Where("Created", ">", ts-int64(opts.TTL))

		// query requires a table scan, but has an ordering requirement; add an index or provide Options.RunQueryFallback (code=Unimplemented)
		q = q.OrderBy("Created", docstore.Descending)

		iter := q.Get(ctx)
		defer iter.Stop()

		var rc auth.RelayCode
		err := iter.Next(ctx, &rc)

		if err != nil {
			LogWithRequest(opts.Logger, req, "Failed to retrieve relay code, %v", err)
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		// START OF reset last update date to 0 - this is mostly for weird edge cases
		// that pop up during debugging where I restart the (iOS) app before
		// the most recent access code has expired and I end up with a QR code
		// that doesn't get hidden (20210816/thisisaaronland)

		update_code := &auth.RelayCode{
			Code: rc.Code,
		}

		mod := docstore.Mods{
			"LastUpdate": 0,
		}

		err = opts.Database.Update(ctx, update_code, mod)

		if err != nil {
			LogWithRequest(opts.Logger, req, "Failed to set last update for '%s', %v", rc.Code, err)
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		// END OF reset last update date to 0

		msg := sse.NewAccessCodeMessage(rc)
		err = msg.Publish(ctx, opts.Publisher)

		if err != nil {
			LogWithRequest(opts.Logger, req, "Failed to publish access code, %v", err)
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		rsp.Write([]byte("OK"))
		return
	}

	h := http.HandlerFunc(fn)
	return h, nil
}

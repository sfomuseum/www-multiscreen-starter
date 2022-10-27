// Package auth provides methods for ensuring that a client/controller request has a valid access token.
package auth

import (
	"context"
	"fmt"
	"github.com/aaronland/go-string/random"
	"gocloud.dev/docstore"
	"io"
	_ "log"
	"time"
)

// type RelayCode is a struct that encapsulates information about an access token (code).
type RelayCode struct {
	// The Unix timestamp when the code was created.
	Created int64 `json:"created"`
	// The Unix timestamp when the code was last updated.
	LastUpdate int64 `json:"lastupdate"`
	// The Unix timestamp when the code expires.
	Expires int64 `json:"expires"`
	// A unique access code.
	Code string `json:"code"`
}

// CurrentRelayCodeWithCollection returns the most create `RelayCode` from 'col' whose creation time is greater than 'ttl'.
func CurrentRelayCodeWithCollection(ctx context.Context, col *docstore.Collection, ttl int) (*RelayCode, error) {

	now := time.Now()
	ts := now.Unix()

	q := col.Query()
	q = q.Where("Created", ">", ts-int64(ttl))

	// query requires a table scan, but has an ordering requirement; add an index or provide Options.RunQueryFallback (code=Unimplemented)
	q = q.OrderBy("Created", docstore.Descending)

	iter := q.Get(ctx)
	defer iter.Stop()

	var rc RelayCode
	err := iter.Next(ctx, &rc)

	if err != nil && err == io.EOF {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("Failed to iterate through codes, %w", err)
	}

	return &rc, nil
}

// NewRelayCodeWithCollection creates (and returns) a new `RelayCode` instance in 'col'.
func NewRelayCodeWithCollection(ctx context.Context, col *docstore.Collection, ttl int) (*RelayCode, error) {

	rc, err := NewRelayCode(ttl)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new relay code, %w", err)
	}

	err = col.Put(ctx, rc)

	if err != nil {
		return nil, fmt.Errorf("Failed to store new relay code, %w", err)
	}

	return rc, nil
}

// NewRelayCode creates a new `RelayCode` with an expiry date 'ttl' seconds from the current time.
func NewRelayCode(ttl int) (*RelayCode, error) {

	code, err := NewAccessCode()

	if err != nil {
		return nil, fmt.Errorf("Failed to create new access code, %w", err)
	}

	now := time.Now()
	created := now.Unix()
	expires := created + int64(ttl)

	r := &RelayCode{
		Code:    code,
		Created: created,
		Expires: expires,
	}

	return r, nil
}

// Return a new unique access code.
func NewAccessCode() (string, error) {

	opts := random.DefaultOptions()
	opts.Length = 16
	opts.AlphaNumeric = true

	return random.String(opts)
}

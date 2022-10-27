// package publisher provides a common interface for publish operations.
package publisher

import (
	"context"
	"fmt"
	"github.com/aaronland/go-roster"
	"net/url"
	"sort"
	"strings"
)

type Publisher interface {
	Publish(context.Context, string) error
	Close() error
}

type PublisherInitializeFunc func(ctx context.Context, uri string) (Publisher, error)

var publishers roster.Roster

func ensurePublisherRoster() error {

	if publishers == nil {

		r, err := roster.NewDefaultRoster()

		if err != nil {
			return err
		}

		publishers = r
	}

	return nil
}

func RegisterPublisher(ctx context.Context, scheme string, f PublisherInitializeFunc) error {

	err := ensurePublisherRoster()

	if err != nil {
		return err
	}

	return publishers.Register(ctx, scheme, f)
}

func Schemes() []string {

	ctx := context.Background()
	schemes := []string{}

	err := ensurePublisherRoster()

	if err != nil {
		return schemes
	}

	for _, dr := range publishers.Drivers(ctx) {
		scheme := fmt.Sprintf("%s://", strings.ToLower(dr))
		schemes = append(schemes, scheme)
	}

	sort.Strings(schemes)
	return schemes
}

func NewPublisher(ctx context.Context, uri string) (Publisher, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	scheme := u.Scheme

	i, err := publishers.Driver(ctx, scheme)

	if err != nil {
		return nil, err
	}

	f := i.(PublisherInitializeFunc)
	return f(ctx, uri)
}

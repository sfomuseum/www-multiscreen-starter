package publisher

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"net/url"
)

type RedisPublisher struct {
	Publisher
	redis_client  *redis.Client
	redis_channel string
}

func init() {
	ctx := context.Background()
	RegisterPublisher(ctx, "redis", NewRedisPublisher)
}

func NewRedisPublisher(ctx context.Context, uri string) (Publisher, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	q := u.Query()

	host := q.Get("host")
	port := q.Get("port")
	channel := q.Get("channel")

	endpoint := fmt.Sprintf("%s:%s", host, port)

	client := redis.NewClient(&redis.Options{
		Addr: endpoint,
	})

	p := &RedisPublisher{
		redis_client:  client,
		redis_channel: channel,
	}

	return p, nil
}

func (p *RedisPublisher) Publish(ctx context.Context, msg string) error {

	select {
	case <-ctx.Done():
		return nil
	default:
		// pass
	}

	cmd := p.redis_client.Publish(ctx, p.redis_channel, msg)
	return cmd.Err()
}

func (p *RedisPublisher) Close() error {
	return p.redis_client.Close()
}

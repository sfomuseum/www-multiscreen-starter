package publisher

import (
	"context"
	"gocloud.dev/pubsub"
)

type GoCloudPublisher struct {
	Publisher
	topic *pubsub.Topic
}

func init() {

	ctx := context.Background()

	for _, scheme := range pubsub.DefaultURLMux().TopicSchemes() {

		err := RegisterPublisher(ctx, scheme, NewGoCloudPublisher)

		if err != nil {
			panic(err)
		}
	}
}

func NewGoCloudPublisher(ctx context.Context, uri string) (Publisher, error) {

	topic, err := pubsub.OpenTopic(ctx, uri)

	if err != nil {
		return nil, err
	}

	pub := &GoCloudPublisher{
		topic: topic,
	}

	return pub, err
}

func (pub *GoCloudPublisher) Publish(ctx context.Context, str_msg string) error {

	msg := &pubsub.Message{
		Body: []byte(str_msg),
	}

	return pub.topic.Send(ctx, msg)
}

func (pub *GoCloudPublisher) Close() error {
	ctx := context.Background()
	return pub.topic.Shutdown(ctx)
}

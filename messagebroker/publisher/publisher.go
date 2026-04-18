package publisher

import (
	"context"

	"github.com/triasbrata/higo/messagebroker/publisher/envelop"
)

type PublishPayload struct {
	Body   []byte
	Header map[string]interface{}
}
type PublisherMiddleware func(ctx context.Context, mail envelop.Envelope) (envelop.Envelope, error)
type Publisher interface {
	PublishToQueue(ctx context.Context, queueName string, Payload PublishPayload) error
	Publish(ctx context.Context, envelope envelop.EnvelopeOption) error
	Use(middleware PublisherMiddleware)
}

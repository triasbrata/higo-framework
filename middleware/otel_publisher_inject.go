package middleware

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/triasbrata/higo/messagebroker/publisher"
	"github.com/triasbrata/higo/messagebroker/publisher/envelop"
	"go.opentelemetry.io/otel"
)

func OtelPublisherInject() publisher.PublisherMiddleware {
	return func(ctx context.Context, mail envelop.Envelope) (envelop.Envelope, error) {
		var header map[string]interface{}
		if mail.AMQP != nil && reflect.ValueOf(header).IsNil() {
			header = mail.AMQP.Payload.Headers
		}
		if reflect.ValueOf(header).IsNil() {
			slog.ErrorContext(ctx, "no header was passing")
			header = make(map[string]interface{})
		}
		otel.GetTextMapPropagator().Inject(ctx, MessageBrokerCarrier(header))
		return mail, nil
	}
}

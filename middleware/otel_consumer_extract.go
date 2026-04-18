package middleware

import (
	"log"

	"github.com/triasbrata/higo/instrumentation"
	"github.com/triasbrata/higo/messagebroker/consumer"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func OtelConsumerExtract() consumer.ConsumeHandler {
	return func(c consumer.CtxConsumer) error {
		h := c.Header()
		// fmt.Printf("h: %v\n", h)
		ctx := otel.GetTextMapPropagator().Extract(c.UserContext(), MessageBrokerCarrier(h))
		ctx, span := instrumentation.Tracer().Start(trace.ContextWithRemoteSpanContext(ctx, trace.SpanContextFromContext(ctx)), c.Route(), trace.WithSpanKind(trace.SpanKindConsumer))
		defer span.End()
		if !span.SpanContext().IsValid() {
			log.Printf("%s: no valid span context found (extraction failed)", c.Route())
			return c.Next()
		}
		c.SetUserContext(ctx)
		return c.Next()
	}
}

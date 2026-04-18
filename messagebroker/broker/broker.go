package broker

import (
	"context"
	"time"

	"github.com/triasbrata/higo/messagebroker/consumer"
	"github.com/triasbrata/higo/messagebroker/publisher"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type BrokerDestination struct {
	Amqp bool
	AmqpConsumerConfig
	UseOtel   bool
	TProvider trace.TracerProvider
	MProvider metric.MeterProvider
}
type AmqpConsumerConfig struct {
	RestartTime time.Duration
}
type otelConfig interface {
	apply(b BrokerDestination) BrokerDestination
}

type applyFunc func(b BrokerDestination) BrokerDestination

func (afx applyFunc) apply(b BrokerDestination) BrokerDestination {
	return afx(b)
}

func WithOtel(TProvider trace.TracerProvider, MProvider metric.MeterProvider) otelConfig {
	return applyFunc(func(b BrokerDestination) BrokerDestination {
		b.UseOtel = true
		b.TProvider = TProvider
		b.MProvider = MProvider
		return b
	})
}

func ConsumeWithAmqp(config AmqpConsumerConfig, otelConfs ...otelConfig) ConBuilder {
	return func() BrokerDestination {
		b := BrokerDestination{Amqp: true, AmqpConsumerConfig: config}
		for _, ofx := range otelConfs {
			b = ofx.apply(b)
		}
		return b
	}
}
func PublishWithAmqp() PubBuilder {
	return func() BrokerDestination {
		return BrokerDestination{Amqp: true}
	}
}

type ConBuilder func() BrokerDestination
type PubBuilder func() BrokerDestination

type Broker interface {
	Publisher(ctx context.Context, destination PubBuilder) (publisher.Publisher, error)
	Consumer(ctx context.Context, builder ConBuilder) (consumer.Consumer, error)
}

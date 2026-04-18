package envelop

import (
	"reflect"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/triasbrata/higo/utils"
)

type EnvelopeOption func() Envelope
type Envelope struct {
	AMQP    *AMQPEnvelope
	Timeout time.Duration
}
type AMQPEnvelope struct {
	Payload   amqp091.Publishing
	Exchange  AMQPEnvelopeExchange
	Mandatory utils.OptionBool
}
type AMQPEnvelopeExchange struct {
	RoutingKey   string
	ExchangeName string
}

func WithAMQPEnvelope(envelope AMQPEnvelope, timeout ...time.Duration) EnvelopeOption {
	return func() Envelope {
		tOut := time.Second
		if len(timeout) == 1 {
			tOut = timeout[0]
		}
		if reflect.ValueOf(envelope.Payload.Headers).IsNil() == true {
			envelope.Payload.Headers = make(amqp091.Table)
		}
		return Envelope{
			AMQP:    &envelope,
			Timeout: tOut,
		}
	}
}

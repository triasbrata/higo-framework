package impl

import (
	"context"
	"fmt"

	"github.com/triasbrata/higo/messagebroker/broker"
	"github.com/triasbrata/higo/messagebroker/consumer"
	"github.com/triasbrata/higo/messagebroker/consumer/amqp"
)

// Consumer implements broker.Broker.
func (b *brk) Consumer(ctx context.Context, builder broker.ConBuilder) (consumer.Consumer, error) {
	config := builder()
	switch {
	case config.Amqp:
		if b.cfg.amqp == nil {
			return nil, fmt.Errorf("configuration amqp was not found")
		}
		conHolder, err := b.openConnectionAmqp(ctx)
		if err != nil {
			return nil, err
		}
		return amqp.NewConsumer(conHolder, config.AmqpConsumerConfig.RestartTime), nil
	}
	return nil, fmt.Errorf("Consumer cant open")
}

package consumer

import (
	"github.com/rabbitmq/amqp091-go"
	"github.com/triasbrata/higo/utils"
)

type AmqpTopologyConsumer struct {
	AutoAck       utils.OptionBool
	Exclusive     utils.OptionBool
	NoLocal       utils.OptionBool
	NoWait        utils.OptionBool
	Durable       utils.OptionBool
	AutoDelete    utils.OptionBool
	Args          amqp091.Table
	BindExchange  *AmqpBindExchange
	PrefetchCount int64
}
type AmqpBindExchange struct {
	NoWait       utils.OptionBool
	RoutingKey   string
	ExchangeName string
	Args         amqp091.Table
	Exchange     *AmqpTopologyConsumerExchange
}
type AmqpTopologyConsumerExchange struct {
	Kind       string
	AutoDelete utils.OptionBool
	Durable    utils.OptionBool
	Internal   utils.OptionBool
	NoWait     utils.OptionBool
	Args       amqp091.Table
}

func WithAmqpTopology(config AmqpTopologyConsumer) ConsumerTopology {
	return func() TopologyConsumer {
		return TopologyConsumer{Amqp: config}
	}
}

package wrap

import (
	"github.com/rabbitmq/amqp091-go"
	"github.com/triasbrata/higo/messagebroker/manager/connections"
)

type Connection struct {
	*amqp091.Connection
}

func (c *Connection) Channel() (connections.ChannelAMQP, error) {
	ch, err := c.Connection.Channel()
	return ch, err
}

type Channel struct {
	amqp091.Channel
}

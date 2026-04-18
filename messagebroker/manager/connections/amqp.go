package connections

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection is the interface you depend on in your code.
type ConnectionAMQP interface {
	Channel() (ChannelAMQP, error)
	Close() error
	IsClosed() bool
	NotifyClose(c chan *amqp.Error) chan *amqp.Error
	NotifyBlocked(receiver chan amqp.Blocking) chan amqp.Blocking
}

// Channel is a subset of *amqp.Channel commonly used in apps.
// Add/remove methods to match your needs.
type ChannelAMQP interface {
	// lifecycle
	Close() error
	IsClosed() bool
	NotifyClose(c chan *amqp.Error) chan *amqp.Error
	NotifyReturn(c chan amqp.Return) chan amqp.Return

	// topology
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	Qos(prefetchCount, prefetchSize int, global bool) error

	// publish/confirm
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Confirm(noWait bool) error
	NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation

	// consume/acks
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	ConsumeWithContext(ctx context.Context, queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Cancel(consumer string, noWait bool) error
	Ack(tag uint64, multiple bool) error
	Nack(tag uint64, multiple, requeue bool) error
	Reject(tag uint64, requeue bool) error
}

package messagebroker

import "github.com/triasbrata/higo-framework/messagebroker/broker/impl"

type AmqpConfigProvider interface {
	GetAmqpConfig() impl.AmqpConfig
}

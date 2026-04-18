package impl

import (
	"slices"
	"time"
)

type AmqpConfigTls struct {
}
type AmqpConfig struct {
	ConnectionName       string
	URI                  string
	TLS                  AmqpConfigTls
	RetryConnectionTimer time.Duration
}

func WithAmqpBroker(config AmqpConfig) brokerConfig {
	return func(brk *brk) {
		brk.cfg.amqp = &brokerConAMQP{
			name: config.ConnectionName,
			uri:  config.URI,
		}
		brk.restartTimer = slices.Max([]time.Duration{config.RetryConnectionTimer, time.Second * 1})
	}
}

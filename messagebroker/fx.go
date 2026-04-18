package messagebroker

import (
	"github.com/triasbrata/higo-framework/messagebroker/broker"
	"github.com/triasbrata/higo-framework/messagebroker/broker/impl"
	"go.uber.org/fx"
)

func LoadMessageBrokerAmqp(staticCfg ...impl.AmqpConfig) fx.Option {
	provider := []fx.Option{}
	if len(staticCfg) == 1 {
		provider = append(provider, fx.Supply(staticCfg[0], fx.Private))
	} else {
		provider = append(provider, fx.Provide(func(prov AmqpConfigProvider) impl.AmqpConfig {
			return prov.GetAmqpConfig()
		}, fx.Private))
	}
	provider = append(provider, fx.Provide(func(cfg impl.AmqpConfig) (broker.Broker, error) {
		return impl.CreateNewBroker(impl.WithAmqpBroker(cfg))
	}))
	return fx.Module("pkg/messagebroker", provider...)
}

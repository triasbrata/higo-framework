package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/triasbrata/higo/instrumentation"
	"github.com/triasbrata/higo/messagebroker/broker"
	mbconsumer "github.com/triasbrata/higo/messagebroker/consumer"
	"github.com/triasbrata/higo/pyroscope"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

type ConsumerServerConfig struct {
	RestartTime time.Duration
}

type ConsumerConfigProvider interface {
	GetConsumerConfig() ConsumerServerConfig
}

type ConsumerRouting func(c mbconsumer.ConsumerBuilder)

type InvokeParam struct {
	fx.In
	Bk         broker.Broker
	Lc         fx.Lifecycle
	Cfg        ConsumerConfigProvider
	InsCfg     instrumentation.InstrumentationProvider
	Routing    ConsumerRouting
	Tp         trace.TracerProvider
	Mp         metric.MeterProvider
	Pyroscope  pyroscope.Profiler
	Shutdowner fx.Shutdowner
}

func InvokeConsumerServer(param InvokeParam) {
	otel.SetMeterProvider(param.Mp)
	otel.SetTracerProvider(param.Tp)
	instrumentation.SetTrace(param.Tp.Tracer(param.InsCfg.GetInstrumentationConfig().AppName))
	invCtx, cancel := context.WithCancel(context.Background())
	param.Lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			cfg := param.Cfg.GetConsumerConfig()
			c, err := param.Bk.Consumer(invCtx, broker.ConsumeWithAmqp(broker.AmqpConsumerConfig{
				RestartTime: cfg.RestartTime,
			}, broker.WithOtel(param.Tp, param.Mp)))
			if err != nil {
				return fmt.Errorf("error when try to consume %w", err)
			}
			param.Routing(c)
			go func() {
				if err := c.Start(invCtx); err != nil {
					param.Shutdowner.Shutdown(fx.ExitCode(1))
				}
			}()
			ok, errChan := c.Status()
			select {
			case err := <-errChan:
				return err
			case <-ctx.Done():
				cancel()
			case <-ok:
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			cancel()
			return param.Pyroscope.Stop()
		},
	})
}

func LoadConsumerServer() fx.Option {
	return fx.Module("bootstrap/consumer",
		fx.Invoke(InvokeConsumerServer),
	)
}

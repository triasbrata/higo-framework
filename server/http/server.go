package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/triasbrata/higo/instrumentation"
	"github.com/triasbrata/higo/middleware"
	"github.com/triasbrata/higo/pyroscope"
	"github.com/triasbrata/higo/routers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

type HttpServerConfig struct {
	Address string
	Port    string
}

type HttpConfigProvider interface {
	GetHttpConfig() HttpServerConfig
}

type InvokeParam struct {
	fx.In
	Lc         fx.Lifecycle
	App        *fiber.App
	Cfg        HttpConfigProvider
	InsCfg     instrumentation.InstrumentationProvider
	TraceProv  trace.TracerProvider
	MeterProv  metric.MeterProvider
	Router     routers.Router
	RouterBind RoutingBind
	Pyroscope  pyroscope.Profiler
	Shutdowner fx.Shutdowner
}
type NewFiberParam struct {
	fx.In
	Cfg       HttpConfigProvider
	TraceProv trace.TracerProvider
	MeterProv metric.MeterProvider
}
type RoutingBind func() error

func NewFiberServer(p NewFiberParam) *fiber.App {
	app := fiber.New(fiber.Config{
		JSONEncoder: func(v interface{}) ([]byte, error) {
			return sonic.Marshal(v)
		},
		JSONDecoder: func(data []byte, v interface{}) error {
			return sonic.Unmarshal(data, v)
		},
	})
	return app
}
func InvokeFiberServer(p InvokeParam) {
	otel.SetTracerProvider(p.TraceProv)
	otel.SetMeterProvider(p.MeterProv)
	instrumentation.SetTrace(p.TraceProv.Tracer(p.InsCfg.GetInstrumentationConfig().AppName))
	var running atomic.Bool
	p.Lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			middleware.OtelFiberGlobal(p.Router, p.TraceProv, p.MeterProv)
			p.RouterBind()
			go func() {
				httpCfg := p.Cfg.GetHttpConfig()
				running.Store(true)
				if err := p.App.Listen(fmt.Sprintf("%s:%s", httpCfg.Address, httpCfg.Port)); err != nil {
					running.Store(false)
					slog.ErrorContext(ctx, "fiber server failed", slog.Any("err", err))
					p.Shutdowner.Shutdown(fx.ExitCode(1))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if !running.Load() {
				return p.Pyroscope.Stop()
			}
			return errors.Join(p.App.ShutdownWithContext(ctx), p.Pyroscope.Stop())
		},
	})
}
func LoadHttpServer() fx.Option {
	return fx.Module("bootstrap/http",
		fx.Provide(NewFiberServer),
		fx.Invoke(InvokeFiberServer),
	)
}

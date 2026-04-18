package grpc

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/triasbrata/higo/instrumentation"
	"github.com/triasbrata/higo/pyroscope"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GrpcServerConfig struct {
	EnableReflection bool
	Address          string
	Port             string
}

type GrpcConfigProvider interface {
	GetGrpcConfig() GrpcServerConfig
}

type GrpcServerBinding func(s *grpc.Server)
type RoutingBind func() error

func LoadGrpcServer() fx.Option {
	return fx.Module("pkg/server/grpc",
		fx.Provide(func(cfgProv GrpcConfigProvider, tProvider trace.TracerProvider, mProvider metric.MeterProvider) *grpc.Server {
			cfg := cfgProv.GetGrpcConfig()
			server := grpc.NewServer(grpc.StatsHandler(
				otelgrpc.NewServerHandler(
					otelgrpc.WithTracerProvider(tProvider),
					otelgrpc.WithMeterProvider(mProvider),
					otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
				)))
			if cfg.EnableReflection {
				reflection.Register(server)
			}
			return server
		}),
		fx.Invoke(func(cfgProv GrpcConfigProvider, insProv instrumentation.InstrumentationProvider, lc fx.Lifecycle, server *grpc.Server, binder GrpcServerBinding, py pyroscope.Profiler, tProvider trace.TracerProvider, mProvider metric.MeterProvider, shutdowner fx.Shutdowner) error {
			cfg := cfgProv.GetGrpcConfig()
			otel.SetMeterProvider(mProvider)
			otel.SetTracerProvider(tProvider)
			instrumentation.SetTrace(tProvider.Tracer(insProv.GetInstrumentationConfig().AppName))
			var running atomic.Bool
			lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
				address := fmt.Sprintf("%s:%s", cfg.Address, cfg.Port)
				lis, err := net.Listen("tcp", address)
				if err != nil {
					return fmt.Errorf("got error when listen to network: %w", err)
				}
				binder(server)
				go func() {
					running.Store(true)
					if err := server.Serve(lis); err != nil {
						running.Store(false)
						shutdowner.Shutdown(fx.ExitCode(1))
					}
				}()
				return nil
			}, OnStop: func(ctx context.Context) error {
				if running.Load() {
					server.GracefulStop()
				}
				return py.Stop()
			}})
			return nil
		}))
}

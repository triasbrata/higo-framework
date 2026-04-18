# higo-framework

A batteries-included Go backend framework built on top of [Uber fx](https://github.com/uber-go/fx), providing HTTP, gRPC, and RabbitMQ consumer services with built-in observability.

## Features

- **HTTP** — [Fiber v2](https://github.com/gofiber/fiber) with OTel middleware
- **gRPC** — `google.golang.org/grpc` with `otelgrpc` interceptors
- **RabbitMQ Consumer** — `amqp091-go` with connection management and auto-restart
- **Tracing & Metrics** — OpenTelemetry (OTLP over gRPC or HTTP)
- **Profiling** — Grafana Pyroscope
- **Database** — PostgreSQL via `jmoiron/sqlx`
- **HTTP Client** — `httpFetcher` wrapper with middleware support
- **Config** — Environment variables, JSON, YAML via Viper
- **DI** — All modules expose `fx.Option` for clean wiring

## Installation

```bash
go get github.com/triasbrata/higo-framework@latest
```

## Quick Start

Use [higo-cli](https://github.com/triasbrata/higo-cli) to scaffold a new project:

```bash
go install github.com/triasbrata/higo-cli@latest
higo init my-project
```

## Package Overview

| Package | Description |
|---|---|
| `instrumentation` | OpenTelemetry setup (tracer, meter, OTLP exporters) |
| `server/http` | Fiber HTTP server with fx lifecycle |
| `server/grpc` | gRPC server with reflection support |
| `server/consumer` | RabbitMQ consumer server with auto-restart |
| `messagebroker` | Broker, publisher, and consumer abstractions |
| `routers` | Fiber router abstraction with fx |
| `middleware` | OTel propagation for HTTP, gRPC, and AMQP |
| `pyroscope` | Grafana Pyroscope profiling integration |
| `database` | PostgreSQL connection via sqlx |
| `httpFetcher` | HTTP client with middleware support |
| `secrets` | Config loaders: env, JSON, YAML |
| `log` | Structured logger (slog) |

## Usage Example

### HTTP Server

```go
func BootHttpServer() fx.Option {
    return fx.Options(
        log.LoadLoggerSlog(),
        config.LoadConfig(),
        instrumentation.OtelModule(...),
        delivery.ModuleHttp(),
        routersfx.LoadModuleRouter(),
        http.LoadHttpServer(),
    )
}

func main() {
    fx.New(BootHttpServer()).Run()
}
```

### gRPC Server

```go
func BootGRPC() fx.Option {
    return fx.Options(
        log.LoadLoggerSlog(),
        config.LoadConfig(),
        instrumentation.OtelModule(...),
        grpc.LoadGrpcServer(),
        delivery.ModuleGrpc(),
    )
}
```

### RabbitMQ Consumer

```go
func BootConsumer() fx.Option {
    return fx.Options(
        log.LoadLoggerSlog(),
        config.LoadConfig(),
        messagebroker.LoadMessageBrokerAmqp(),
        delivery.ModuleConsumer(),
        serverConsumer.LoadConsumerServer(),
    )
}
```

## Configuration

All configuration is loaded from environment variables via `secrets.Secret`:

| Env Var | Default | Description |
|---|---|---|
| `APP_NAME` | `app` | Application name for telemetry |
| `HTTP_PORT` | `8000` | HTTP server port |
| `GRPC_PORT` | `8001` | gRPC server port |
| `INS_ENDPOINT` | `localhost:4317` | OTLP collector endpoint |
| `INS_USE_GRPC` | `true` | Use gRPC for OTLP export |
| `AMQP_URI` | `amqp://guest:guest@localhost:5672` | RabbitMQ connection URI |
| `PYROSCOPE_ENABLED` | `false` | Enable Pyroscope profiling |
| `PYROSCOPE_SERVER_ADDRESS` | `http://localhost:9999` | Pyroscope server URL |

## Requirements

- Go 1.24+
- RabbitMQ (if using consumer/publisher)
- PostgreSQL (if using database package)
- OpenTelemetry Collector (optional, for tracing/metrics)

## License

MIT

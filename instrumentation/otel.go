package instrumentation

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/go-logr/logr"
	otelpyroscope "github.com/grafana/otel-profiling-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	ometric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	otrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"google.golang.org/grpc/credentials"
)

var gTracer otrace.Tracer

func SetTrace(tracer otrace.Tracer) {
	gTracer = tracer
}
func Tracer() otrace.Tracer {
	return gTracer
}

type InstrumentationConfig struct {
	AppName               string
	CaFile                []byte
	Endpoint              string
	UseGRPC               bool
	UseHttp               bool
	IntegrateWithPyroscope bool
}

type InstrumentationProvider interface {
	GetInstrumentationConfig() InstrumentationConfig
}

// PyroscopeChecker is an optional interface that lets instrumentation
// check whether pyroscope profiling is enabled, without importing the pyroscope package.
type PyroscopeChecker interface {
	IsPyroscopeEnabled() bool
}

type Params struct {
	fx.In
	Lc          fx.Lifecycle
	Cfg         InstrumentationProvider
	Logger      *slog.Logger
	ResourceIn  []InstrumentationIn `group:"otelattrs"`
	PyroChecker PyroscopeChecker    `optional:"true"`
}
type InstrumentationResult struct {
	fx.Out
	TraceProv otrace.TracerProvider
	MeterProv ometric.MeterProvider
}
type InstrumentationIn interface {
	Resource() []attribute.KeyValue
}

// The function type
type InstrumentationInFunc func() []attribute.KeyValue

// Implement the interface
func (f InstrumentationInFunc) Resource() []attribute.KeyValue {
	return f()
}

// Build x509.CertPool if caFile is present, or nil if not.
func buildCertPool(caFile []byte) (*x509.CertPool, error) {
	if len(caFile) == 0 {
		return nil, nil
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caFile) {
		return nil, fmt.Errorf("invalid CA file")
	}
	return certPool, nil
}

func OtelModule(factory ...interface{}) fx.Option {
	resourceAttributes := []attribute.KeyValue{}
	providers := []fx.Option{}
	for _, fc := range factory {
		switch arg := fc.(type) {
		case attribute.KeyValue:
			resourceAttributes = append(resourceAttributes, arg)
		default:
			rfc := reflect.TypeOf(fc)
			if rfc.Kind() != reflect.Func {
				continue
			}
			if rfc.NumOut() == 0 || rfc.NumOut() > 1 {
				panic("otel factory cant be out empty or more than 1")
			}
			if !rfc.Out(0).Implements(reflect.TypeOf((*InstrumentationIn)(nil)).Elem()) {
				panic(fmt.Sprintf("otel factory must have output %T ", new(InstrumentationIn)))
			}
			providers = append(providers,
				fx.Provide(
					fx.Private,
					fx.Annotate(fc,
						fx.ResultTags(`group:"otelattrs"`),
					),
				),
			)
		}
	}

	providers = append(providers,
		fx.Supply(
			fx.Private,
			fx.Annotate(
				InstrumentationInFunc(func() []attribute.KeyValue {
					return resourceAttributes
				}),
				fx.As(new(InstrumentationIn)),      // cast value to the interface
				fx.ResultTags(`group:"otelattrs"`), // put it into the group
			),
		),
	)
	providers = append(providers, fx.Provide(NewInstrumentation))
	return fx.Module("instrumentation/otel", providers...)
}

func NewInstrumentation(p Params) (InstrumentationResult, error) {
	cfg := p.Cfg.GetInstrumentationConfig()
	otel.SetLogger(logr.FromSlogHandler(p.Logger.Handler()))
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	attrs := []attribute.KeyValue{semconv.ServiceName(cfg.AppName)}
	for _, res := range p.ResourceIn {
		attrs = append(attrs, res.Resource()...)
	}
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL, attrs...),
	)
	if err != nil {
		return InstrumentationResult{}, fmt.Errorf("cant create otel resource because: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.Lc.Append(fx.StopHook(func(_ context.Context) error {
		cancel()
		return nil
	}))
	wrapWithPyroscope := cfg.IntegrateWithPyroscope && p.PyroChecker != nil && p.PyroChecker.IsPyroscopeEnabled()
	hTracer, err := NewTracerProvider(ctx, &cfg, wrapWithPyroscope, res)
	if err != nil {
		return InstrumentationResult{}, err
	}
	p.Lc.Append(hTracer.Hook)
	hMeter, err := NewMeterProvider(ctx, &cfg, res)
	if err != nil {
		return InstrumentationResult{}, err
	}
	p.Lc.Append(hMeter.Hook)
	return InstrumentationResult{
		TraceProv: hTracer.Provider,
		MeterProv: hMeter.Provider,
	}, nil
}

type HookMeterResult struct {
	Provider ometric.MeterProvider
	Hook     fx.Hook
}

type HookTracerResult struct {
	Provider otrace.TracerProvider
	Hook     fx.Hook
}

func NewMeterProvider(ctx context.Context, cfg *InstrumentationConfig, res *resource.Resource) (HookMeterResult, error) {
	var exporter metric.Exporter
	var err error

	certPool, err := buildCertPool(cfg.CaFile)
	if err != nil {
		return HookMeterResult{}, fmt.Errorf("failed to build TLS cert pool for metrics: %w", err)
	}

	switch {
	case cfg.UseGRPC:
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
			otlpmetricgrpc.WithCompressor("gzip"),
		}
		if certPool != nil {
			opts = append(opts, otlpmetricgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(certPool, "")))
		} else {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		exporter, err = otlpmetricgrpc.New(ctx, opts...)
	case cfg.UseHttp:
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
		}
		if cfg.Endpoint != "" {
			opts = append(opts, otlpmetrichttp.WithEndpoint(cfg.Endpoint))
		}
		if certPool != nil {
			tlsConfig := &tls.Config{RootCAs: certPool}
			opts = append(opts, otlpmetrichttp.WithTLSClientConfig(tlsConfig))
		} else {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		exporter, err = otlpmetrichttp.New(ctx, opts...)
	default:
		return HookMeterResult{}, fmt.Errorf("must set UseGRPC or UseHttp for instrumentation")
	}

	if err != nil {
		return HookMeterResult{}, fmt.Errorf("can't create otel metric exporter: %w", err)
	}

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exporter)),
	)

	hook := fx.Hook{
		OnStop: func(ctx context.Context) error {
			return provider.Shutdown(ctx)
		},
	}
	return HookMeterResult{
		Provider: provider,
		Hook:     hook,
	}, nil
}
func NewTracerProvider(ctx context.Context, cfg *InstrumentationConfig, wrapWithPyroscope bool, res *resource.Resource) (HookTracerResult, error) {
	var exporter *otlptrace.Exporter
	var err error

	certPool, err := buildCertPool(cfg.CaFile)
	if err != nil {
		return HookTracerResult{}, fmt.Errorf("failed to build TLS cert pool for traces: %w", err)
	}

	switch {
	case cfg.UseGRPC:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
			otlptracegrpc.WithCompressor("gzip"),
		}
		if certPool != nil {
			opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(certPool, "")))
		} else {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
	case cfg.UseHttp:
		opts := []otlptracehttp.Option{
			otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		}
		if cfg.Endpoint != "" {
			opts = append(opts, otlptracehttp.WithEndpoint(cfg.Endpoint))
		}
		if certPool != nil {
			tlsConfig := &tls.Config{RootCAs: certPool}
			opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsConfig))
		} else {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
	default:
		return HookTracerResult{}, fmt.Errorf("must set UseGRPC or UseHttp for instrumentation")
	}

	if err != nil {
		return HookTracerResult{}, fmt.Errorf("can't create otel trace exporter: %w", err)
	}

	oProvider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(res),
	)

	var provider otrace.TracerProvider = oProvider
	if wrapWithPyroscope {
		provider = otelpyroscope.NewTracerProvider(oProvider)
	}

	hook := fx.Hook{
		OnStop: func(ctx context.Context) error {
			return oProvider.Shutdown(ctx)
		},
	}

	return HookTracerResult{
		Provider: provider,
		Hook:     hook,
	}, nil
}

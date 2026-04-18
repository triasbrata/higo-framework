package impl

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	httpfetcher "github.com/triasbrata/higo/httpFetcher"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Options func(s *std)
type std struct {
	timeout    time.Duration
	client     *http.Client
	transports []httpfetcher.Middleware
}

// Delete implements httpfetcher.HttpFetcher.
func (s *std) Delete(ctx context.Context, uri string, data interface{}, opts ...httpfetcher.RequestOption) (*http.Response, error) {
	return s.doRequest(ctx, http.MethodDelete, data, uri, "delete", opts)
}

// Get implements httpfetcher.HttpFetcher.
func (s *std) Get(ctx context.Context, uri string, opts ...httpfetcher.RequestOption) (*http.Response, error) {

	return s.doRequest(ctx, http.MethodGet, nil, uri, "get", opts)
}

// Option implements httpfetcher.HttpFetcher.
func (s *std) Option(ctx context.Context, uri string, opts ...httpfetcher.RequestOption) (*http.Response, error) {
	return s.doRequest(ctx, http.MethodOptions, nil, uri, "options", opts)
}

// Patch implements httpfetcher.HttpFetcher.
func (s *std) Patch(ctx context.Context, uri string, data interface{}, opts ...httpfetcher.RequestOption) (*http.Response, error) {
	return s.doRequest(ctx, http.MethodPatch, nil, uri, "patch", opts)
}

// Post implements httpfetcher.HttpFetcher.
func (s *std) Post(ctx context.Context, uri string, data interface{}, opts ...httpfetcher.RequestOption) (*http.Response, error) {
	return s.doRequest(ctx, http.MethodPost, nil, uri, "post", opts)

}

// Put implements httpfetcher.HttpFetcher.
func (s *std) Put(ctx context.Context, uri string, data interface{}, opts ...httpfetcher.RequestOption) (*http.Response, error) {
	return s.doRequest(ctx, http.MethodPut, nil, uri, "put", opts)

}

func WithOtelInjector(options ...otelhttp.Option) Options {
	return func(s *std) {
		s.transports = append(s.transports, func(rt http.RoundTripper) http.RoundTripper {
			return otelhttp.NewTransport(rt, options...)
		})
	}
}
func WithTimeout(timeout time.Duration) Options {
	return func(s *std) {
		s.timeout = timeout
	}
}
func WithLogger(logger *slog.Logger, logOutput bool) Options {
	return func(s *std) {
		s.transports = append(s.transports, func(rt http.RoundTripper) http.RoundTripper {
			return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				start := time.Now()
				res, err := rt.RoundTrip(r)
				end := time.Since(start)
				go func() {
					attrs := []any{
						slog.Any("method", r.Method), slog.Any("uri", r.URL.String()),
					}
					logger.InfoContext(r.Context(), "Http Request", attrs...)
					attrs = append(attrs, slog.Any("execTime", end.Milliseconds()))
					if err != nil {
						attrs = append(attrs, slog.Any("isError", true), slog.Any("err", err))
					} else {
						attrs = append(attrs, slog.Any("isError", false))
					}
					logger.InfoContext(r.Context(), "Http Response", attrs...)
				}()
				return res, err
			})
		})
	}
}
func NewFetcher(options ...Options) httpfetcher.HttpFetcher {
	instance := &std{
		transports: []httpfetcher.Middleware{},
	}
	for _, opt := range options {
		opt(instance)
	}
	instance.client = &http.Client{
		Transport: middlewareChain(http.DefaultTransport, instance.transports...),
		Timeout:   instance.timeout,
	}
	return instance
}

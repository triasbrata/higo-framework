package httpfetcher

import (
	"context"
	"net/http"
)

type HttpFetcher interface {
	Get(ctx context.Context, uri string, opts ...RequestOption) (*http.Response, error)
	Post(ctx context.Context, uri string, data interface{}, opts ...RequestOption) (*http.Response, error)
	Patch(ctx context.Context, uri string, data interface{}, opts ...RequestOption) (*http.Response, error)
	Put(ctx context.Context, uri string, data interface{}, opts ...RequestOption) (*http.Response, error)
	Delete(ctx context.Context, uri string, data interface{}, opts ...RequestOption) (*http.Response, error)
	Option(ctx context.Context, uri string, opts ...RequestOption) (*http.Response, error)
}

type Middleware func(http.RoundTripper) http.RoundTripper
type RequestOption func(req *http.Request)

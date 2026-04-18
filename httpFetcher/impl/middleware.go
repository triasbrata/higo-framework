package impl

import (
	"net/http"

	httpfetcher "github.com/triasbrata/higo/httpFetcher"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
func middlewareChain(rt http.RoundTripper, middlewares ...httpfetcher.Middleware) http.RoundTripper {
	for i := len(middlewares) - 1; i >= 0; i-- {
		rt = middlewares[i](rt)
	}
	return rt
}

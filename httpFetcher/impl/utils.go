package impl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	httpfetcher "github.com/triasbrata/higo/httpFetcher"
)

func (s *std) doRequest(ctx context.Context, method string, data interface{}, uri string, msgAction string, opts []httpfetcher.RequestOption) (*http.Response, error) {
	body := io.Reader(http.NoBody)
	if data != nil {
		dataByte, err := sonic.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("cant marshal the request: %w ", err)
		}
		body = bytes.NewBuffer(dataByte)
	}
	req, err := http.NewRequestWithContext(ctx, method, uri, body)
	if err != nil {

		return nil, fmt.Errorf("cant create new %s request: %w ", msgAction, err)
	}
	for _, fx := range opts {
		fx(req)
	}
	return s.client.Do(req)
}

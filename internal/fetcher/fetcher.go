package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Result struct {
	URL        string
	StatusCode int
	Body       []byte
	Err        error
}

type Fetcher struct {
	client *http.Client
}

func New(requestTimeout time.Duration) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: requestTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (f *Fetcher) Fetch(ctx context.Context, rawURL string) Result {
	res := Result{URL: rawURL}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		res.Err = err
		return res
	}

	resp, err := f.client.Do(req)
	if err != nil {
		res.Err = err
		return res
	}
	defer resp.Body.Close()

	res.StatusCode = resp.StatusCode
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		res.Err = fmt.Errorf("unexpected status: %d", resp.StatusCode)
		return res
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		res.Err = fmt.Errorf("not html: %s", contentType)
		return res
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		res.Err = err
		return res
	}
	res.Body = body
	return res
}

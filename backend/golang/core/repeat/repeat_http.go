package repeat

import (
	"context"
	"net/http"
	"time"
)

type ClientWithRetry struct {
	client *http.Client
	config []OptionSetter
}

func NewClient(baseClient *http.Client, opts ...OptionSetter) *ClientWithRetry {
	if baseClient == nil {
		baseClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &ClientWithRetry{
		client: baseClient,
		config: opts,
	}
}

func (c *ClientWithRetry) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	op := func(ctx context.Context, retryCount int) error {

		newReq := req.Clone(ctx)
		r, err := c.client.Do(newReq)
		if err != nil {
			return err
		}

		if r.StatusCode >= 500 {
			_ = r.Body.Close()
			return &TemporaryError{StatusCode: r.StatusCode}
		}
		resp = r
		return nil
	}

	err := Exec(req.Context(), op, c.config...)
	return resp, err
}

type TemporaryError struct {
	StatusCode int
}

// TemporaryError represents a temporary error response.
type TemporaryError struct {
	StatusCode int
}

// Error implements the error interface.
func (e *TemporaryError) Error() string {
	return http.StatusText(e.StatusCode)
}

// Temporary returns true if the error is temporary.
func (e *TemporaryError) Temporary() bool { return true }

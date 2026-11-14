// Package dataset holds the plumbing shared by the sport packages, which all
// read their data from static CSV files published on GitHub releases.
package dataset

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ErrStatsNotAvailable = errors.New("stats not yet available for game")
	ErrGameNotFound      = errors.New("game not found")
	ErrNotFound          = errors.New("file not found")
)

// Fetch downloads a URL, retrying transient failures. It is a variable so tests
// can replace it (see Stub).
var Fetch = func(url string) (io.ReadCloser, error) {
	client := &http.Client{Timeout: 5 * time.Minute}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}

		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}

		switch {
		case resp.StatusCode == http.StatusOK:
			return resp.Body, nil
		case resp.StatusCode == http.StatusNotFound:
			resp.Body.Close()
			return nil, fmt.Errorf("%s: %w", url, ErrNotFound)
		default:
			resp.Body.Close()
			lastErr = fmt.Errorf("%s: unexpected status %d", url, resp.StatusCode)
		}
	}

	return nil, lastErr
}

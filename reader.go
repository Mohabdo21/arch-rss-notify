package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/mmcdole/gofeed"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
	},
}

var feedParser = gofeed.NewParser()

// httpStatusError wraps a non-2xx HTTP response status code.
type httpStatusError struct {
	Code int
}

func (e httpStatusError) Error() string {
	return fmt.Sprintf("unexpected HTTP status %d", e.Code)
}

func FetchFeed(ctx context.Context, url string) ([]*gofeed.Item, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError{Code: resp.StatusCode}
	}

	feed, err := feedParser.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	return feed.Items, nil
}

// isTransientError returns true if err is likely to be resolved by retrying
// (timeouts, DNS failures, connection refused/reset, TLS handshake failures,
// 5xx HTTP status codes).
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	if httpErr, ok := errors.AsType[httpStatusError](err); ok {
		return httpErr.Code == http.StatusBadGateway ||
			httpErr.Code == http.StatusServiceUnavailable ||
			httpErr.Code == http.StatusGatewayTimeout
	}

	if netErr, ok := errors.AsType[net.Error](err); ok {
		if netErr.Timeout() {
			return true
		}
	}

	if _, ok := errors.AsType[*net.DNSError](err); ok {
		return true
	}

	if opErr, ok := errors.AsType[*net.OpError](err); ok {
		if syscallErr, ok := opErr.Err.(syscall.Errno); ok {
			return syscallErr == syscall.ECONNREFUSED ||
				syscallErr == syscall.ECONNRESET ||
				syscallErr == syscall.ETIMEDOUT
		}
	}

	return false
}

// FetchFeedWithRetry calls FetchFeed and retries on transient errors using
// exponential backoff. The backoff doubles each attempt and includes +/-25%
// jitter. Non-transient errors and context cancellation are returned
// immediately.
func FetchFeedWithRetry(
	ctx context.Context,
	url string,
	retries int,
	backoff time.Duration,
) ([]*gofeed.Item, error) {
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			wait := backoff * (1 << uint(attempt-1))
			if wait > 0 {
				jitter := time.Duration(
					rand.Int63n(int64(wait/2)),
				) - wait/4
				wait += jitter
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}

		items, err := FetchFeed(ctx, url)
		if err == nil {
			return items, nil
		}

		lastErr = err
		if !isTransientError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf(
		"fetching %s failed after %d retries: %w",
		url, retries, lastErr,
	)
}

// FetchConfig controls retry behaviour for FetchFeeds.
type FetchConfig struct {
	Retries int
	Backoff time.Duration
}

// FetchFeeds fetches all RSS feeds concurrently, deduplicates by GUID, and
// returns the combined items. If every feed fails it returns a single error;
// partial failures are logged and the successful results are returned.
func FetchFeeds(
	ctx context.Context,
	urls []string,
	cfg FetchConfig,
) ([]*gofeed.Item, error) {
	type feedResult struct {
		items []*gofeed.Item
		err   error
		url   string
	}

	results := make(chan feedResult, len(urls))
	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Go(func() {
			items, err := FetchFeedWithRetry(
				ctx, url, cfg.Retries, cfg.Backoff,
			)
			results <- feedResult{items: items, err: err, url: url}
		})
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	seen := make(map[string]bool)
	var all []*gofeed.Item
	var lastErr error
	var failureCount int

	for r := range results {
		if r.err != nil {
			failureCount++
			lastErr = r.err
			log.Printf("error fetching %s: %v", r.url, r.err)
			continue
		}
		for _, item := range r.items {
			if item.GUID != "" {
				if seen[item.GUID] {
					continue
				}
				seen[item.GUID] = true
			}
			all = append(all, item)
		}
	}

	if failureCount == len(urls) {
		return nil, fmt.Errorf(
			"all %d feeds failed: %w", len(urls), lastErr,
		)
	}

	if failureCount > 0 {
		log.Printf(
			"warning: %d of %d feeds failed, returning partial results",
			failureCount, len(urls),
		)
	}

	return all, nil
}

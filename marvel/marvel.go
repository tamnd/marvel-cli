// Package marvel is the library behind the marvel command: the HTTP client,
// authentication, pacing, and the typed data models for the Marvel Comics API.
//
// The Marvel API at gateway.marvel.com/v1/public requires two credentials:
// MARVEL_PUBLIC_KEY and MARVEL_PRIVATE_KEY. Every request is signed with a
// timestamp and MD5 hash. See https://developer.marvel.com/documentation/authorization
package marvel

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	// Host is the main developer site hostname.
	Host = "developer.marvel.com"

	// GatewayHost is the API gateway hostname.
	GatewayHost = "gateway.marvel.com"

	// APIBaseURL is the root for all REST calls.
	APIBaseURL = "https://gateway.marvel.com/v1/public"

	// DefaultUserAgent identifies the CLI to the server.
	DefaultUserAgent = "marvel/dev (+https://github.com/tamnd/marvel-cli)"
)

// ErrNotFound is returned when the API returns 404.
var ErrNotFound = errors.New("not found")

// ErrRateLimited is returned when the API returns 429 and retries are exhausted.
var ErrRateLimited = errors.New("rate limited")

// ErrMissingKeys is returned when MARVEL_PUBLIC_KEY or MARVEL_PRIVATE_KEY are not set.
var ErrMissingKeys = errors.New("MARVEL_PUBLIC_KEY and MARVEL_PRIVATE_KEY must be set")

// Config holds constructor parameters for Client.
type Config struct {
	APIBaseURL string
	PublicKey  string
	PrivateKey string
	UserAgent  string
	Rate       time.Duration
	Retries    int
	Timeout    time.Duration
}

// DefaultConfig returns sensible defaults for the Marvel API.
func DefaultConfig() Config {
	return Config{
		APIBaseURL: APIBaseURL,
		UserAgent:  DefaultUserAgent,
		Rate:       500 * time.Millisecond,
		Retries:    3,
		Timeout:    30 * time.Second,
	}
}

// FromEnv overlays MARVEL_PUBLIC_KEY and MARVEL_PRIVATE_KEY onto the config.
func (c *Config) FromEnv() {
	if v := os.Getenv("MARVEL_PUBLIC_KEY"); v != "" {
		c.PublicKey = v
	}
	if v := os.Getenv("MARVEL_PRIVATE_KEY"); v != "" {
		c.PrivateKey = v
	}
}

// Client is a rate-limited, signed HTTP client for the Marvel REST API.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
// Returns ErrMissingKeys if either key is empty.
func NewClient(cfg Config) (*Client, error) {
	if cfg.PublicKey == "" || cfg.PrivateKey == "" {
		return nil, ErrMissingKeys
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}, nil
}

// authParams returns the (ts, apikey, hash) tuple for a request.
func (c *Client) authParams() (ts, apikey, hash string) {
	ts = strconv.FormatInt(time.Now().UnixMilli(), 10)
	raw := ts + c.cfg.PrivateKey + c.cfg.PublicKey
	h := md5.Sum([]byte(raw))
	hash = hex.EncodeToString(h[:])
	return ts, c.cfg.PublicKey, hash
}

// signURL appends authentication parameters to rawURL.
func (c *Client) signURL(rawURL string) string {
	ts, apikey, hash := c.authParams()
	sep := "?"
	for _, r := range rawURL {
		if r == '?' {
			sep = "&"
			break
		}
	}
	return rawURL + sep + "ts=" + ts + "&apikey=" + apikey + "&hash=" + hash
}

// get fetches a URL with authentication, pacing, and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	signed := c.signURL(rawURL)
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, signed)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, true, ErrRateLimited
	}
	if resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, ErrNotFound
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, false, fmt.Errorf("http %d: invalid or missing Marvel API keys", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// getJSON fetches a URL and JSON-decodes into v.
func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

// backoff returns the wait duration for a given retry attempt.
func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

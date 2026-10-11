// Package datafs calls a self-hosted DataFS install.
// Authorization is a bearer stats key or, for ServerEvent, an ingest token.
package datafs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

type Version struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

type ServerEvent struct {
	Domain         string            `json:"domain"`
	Name           string            `json:"name"`
	URL            string            `json:"url,omitempty"`
	Referrer       string            `json:"referrer,omitempty"`
	Props          map[string]string `json:"props,omitempty"`
	IP             string            `json:"ip,omitempty"`
	UserAgent      string            `json:"user_agent,omitempty"`
	VisitorKey     string            `json:"visitor_key,omitempty"`
	Timestamp      string            `json:"timestamp,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

type Totals struct {
	Visitors             int     `json:"visitors"`
	Pageviews            int     `json:"pageviews"`
	Visits               int     `json:"visits"`
	BounceRate           float64 `json:"bounce_rate"`
	VisitDurationSeconds float64 `json:"visit_duration_seconds"`
}

type Point struct {
	T                    string  `json:"t"`
	Visitors             int     `json:"visitors"`
	Pageviews            int     `json:"pageviews"`
	Visits               int     `json:"visits"`
	BounceRate           float64 `json:"bounce_rate"`
	VisitDurationSeconds float64 `json:"visit_duration_seconds"`
}

type Timeseries struct {
	From            string  `json:"from"`
	To              string  `json:"to"`
	Interval        string  `json:"interval"`
	Stale           bool    `json:"stale"`
	RollupFreshness string  `json:"rollup_freshness"`
	Totals          Totals  `json:"totals"`
	Points          []Point `json:"points"`
}

type BreakdownRow struct {
	Value     string `json:"value"`
	Visitors  int    `json:"visitors"`
	Pageviews int    `json:"pageviews"`
	Entrances int    `json:"entrances"`
	Exits     int    `json:"exits"`
}

type Breakdown struct {
	From      string         `json:"from"`
	To        string         `json:"to"`
	Dimension string         `json:"dimension"`
	Rows      []BreakdownRow `json:"rows"`
}

type Realtime struct {
	Visitors int `json:"visitors"`
}

type FunnelStep struct {
	Kind     string `json:"kind"`
	Value    string `json:"value"`
	Visitors int    `json:"visitors,omitempty"`
}

type Funnel struct {
	Day   string       `json:"day"`
	Steps []FunnelStep `json:"steps"`
}

func (c *Client) Version(ctx context.Context) (Version, error) {
	var out Version
	err := c.do(ctx, http.MethodGet, "/api/v1/version", nil, nil, &out)
	return out, err
}

func (c *Client) ServerEvent(ctx context.Context, ev ServerEvent) (string, error) {
	var out struct {
		EventID string `json:"event_id"`
	}
	err := c.do(ctx, http.MethodPost, "/api/v1/server-event", nil, ev, &out)
	return out.EventID, err
}

func (c *Client) Timeseries(ctx context.Context, siteID, from, to, interval string) (Timeseries, error) {
	var out Timeseries
	err := c.do(ctx, http.MethodGet, "/api/v1/sites/"+url.PathEscape(siteID)+"/timeseries", map[string]string{
		"from": from, "to": to, "interval": interval,
	}, nil, &out)
	return out, err
}

func (c *Client) Breakdown(ctx context.Context, siteID, from, to, dimension string, limit int, country string) (Breakdown, error) {
	q := map[string]string{"from": from, "to": to, "dimension": dimension}
	if limit > 0 {
		q["limit"] = fmt.Sprintf("%d", limit)
	}
	if country != "" {
		q["country"] = country
	}
	var out Breakdown
	err := c.do(ctx, http.MethodGet, "/api/v1/sites/"+url.PathEscape(siteID)+"/breakdown", q, nil, &out)
	return out, err
}

func (c *Client) Realtime(ctx context.Context, siteID string) (Realtime, error) {
	var out Realtime
	err := c.do(ctx, http.MethodGet, "/api/v1/sites/"+url.PathEscape(siteID)+"/realtime", nil, nil, &out)
	return out, err
}

func (c *Client) Funnel(ctx context.Context, siteID string, funnel Funnel) (Funnel, error) {
	var out Funnel
	err := c.do(ctx, http.MethodPost, "/api/v1/sites/"+url.PathEscape(siteID)+"/funnel", nil, funnel, &out)
	return out, err
}

func (c *Client) do(ctx context.Context, method, path string, query map[string]string, body any, dest any) error {
	u, err := url.Parse(strings.TrimRight(c.BaseURL, "/") + path)
	if err != nil {
		return err
	}
	if len(query) > 0 {
		q := u.Query()
		for k, v := range query {
			if v != "" {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return err
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		apiErr := &Error{Status: res.StatusCode}
		var payload struct {
			Code    string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &payload) == nil {
			apiErr.Code = payload.Code
			apiErr.Message = payload.Message
		}
		if apiErr.Code == "" {
			apiErr.Code = res.Status
		}
		return apiErr
	}
	if dest == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	return json.Unmarshal(raw, dest)
}

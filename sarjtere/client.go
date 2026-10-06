package main

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAppBase    = "https://sarjtr.epdk.gov.tr/sarjet/api"
	defaultGatewayURL = "https://apigateway.epdk.gov.tr/sarjIstasyonlari/"
	defaultMirrorURL  = "https://alperkonan.github.io/prizy-data/stations.json"
	defaultUserAgent  = "Dart/3.8 (dart:io)"
)

type Client struct {
	HTTP       *http.Client
	AppBase    string
	GatewayURL string
	MirrorURL  string
	UserAgent  string
}

func NewClient() *Client {
	return &Client{
		HTTP:       &http.Client{Timeout: 2 * time.Minute},
		AppBase:    defaultAppBase,
		GatewayURL: defaultGatewayURL,
		MirrorURL:  defaultMirrorURL,
		UserAgent:  defaultUserAgent,
	}
}

type HTTPError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	msg := fmt.Sprintf("%s %s: %s", e.Method, e.URL, http.StatusText(e.StatusCode))
	if e.StatusCode == 0 {
		msg = fmt.Sprintf("%s %s: status %d", e.Method, e.URL, e.StatusCode)
	}
	if e.Body != "" {
		msg += ": " + e.Body
	}
	return msg
}

func (c *Client) Stations(ctx context.Context) ([]StationSummary, error) {
	var out []StationSummary
	if err := c.appGet(ctx, "/stations", &out); err != nil {
		return nil, fmt.Errorf("stations: %w", err)
	}
	return out, nil
}

// Station returns the detail for the given day: sockets, prices, availability.
func (c *Client) Station(ctx context.Context, id int64, day time.Time) (*Station, error) {
	stamp := day.In(Istanbul).Format("2006-01-02 15:04:05")
	path := fmt.Sprintf("/stations/id/%d/%s", id, url.PathEscape(stamp))

	var out Station
	if err := c.appGet(ctx, path, &out); err != nil {
		return nil, fmt.Errorf("station %d: %w", id, err)
	}
	return &out, nil
}

func (c *Client) Cities(ctx context.Context) ([]City, error) {
	var out []City
	if err := c.appGet(ctx, "/cities/list", &out); err != nil {
		return nil, fmt.Errorf("cities: %w", err)
	}
	return out, nil
}

func (c *Client) Registry(ctx context.Context) (*Registry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.GatewayURL, strings.NewReader("{}"))
	if err != nil {
		return nil, fmt.Errorf("registry: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	var out Registry
	if err := c.do(req, &out); err != nil {
		return nil, fmt.Errorf("registry: %w", err)
	}
	return &out, nil
}

func (c *Client) Mirror(ctx context.Context) (*Mirror, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.MirrorURL, nil)
	if err != nil {
		return nil, fmt.Errorf("mirror: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	var out Mirror
	if err := c.do(req, &out); err != nil {
		return nil, fmt.Errorf("mirror: %w", err)
	}
	return &out, nil
}

func (c *Client) appGet(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.AppBase+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")

	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return &HTTPError{
			Method:     req.Method,
			URL:        req.URL.String(),
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(bytes.ToValidUTF8(body, nil))),
		}
	}

	// App backend sends zone-less "2006-01-02T15:04:05" in Istanbul time.
	unmarshalTime := json.UnmarshalFunc(func(b []byte, t *time.Time) error {
		s, err := strconv.Unquote(string(b))
		if err != nil {
			return err
		}

		if parsed, err := time.ParseInLocation("2006-01-02T15:04:05", s, Istanbul); err == nil {
			*t = parsed
			return nil
		}

		parsed, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return err
		}
		*t = parsed
		return nil
	})

	if err := json.UnmarshalRead(resp.Body, out, json.WithUnmarshalers(unmarshalTime)); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

// distanceKM is the haversine great-circle distance.
func distanceKM(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371.0
	rad := func(d float64) float64 { return d * math.Pi / 180 }

	dlat := rad(lat2 - lat1)
	dlng := rad(lng2 - lng1)
	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dlng/2)*math.Sin(dlng/2)

	return 2 * r * math.Asin(math.Sqrt(a))
}

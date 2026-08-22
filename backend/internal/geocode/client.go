// Package geocode reverse-geocodes coordinates to a town/city name using
// OpenStreetMap's Nominatim service, so routes can be named after where
// they are rather than an arbitrary number. Nominatim's free usage policy
// requires a descriptive User-Agent and a max of 1 request/second, both of
// which this client enforces; results are cached in-process since a given
// route's location won't change between syncs.
package geocode

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	reverseURL  = "https://nominatim.openstreetmap.org/reverse"
	minInterval = 1100 * time.Millisecond
)

type Client struct {
	httpClient *http.Client
	userAgent  string

	mu       sync.Mutex
	lastCall time.Time
	cache    map[string]string
}

func NewClient(userAgent string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		userAgent:  userAgent,
		cache:      make(map[string]string),
	}
}

type reverseResponse struct {
	Address struct {
		City    string `json:"city"`
		Town    string `json:"town"`
		Village string `json:"village"`
		Suburb  string `json:"suburb"`
		County  string `json:"county"`
		State   string `json:"state"`
	} `json:"address"`
}

// TownName reverse-geocodes a coordinate to a town/city name, falling back
// through progressively broader administrative areas if a precise one
// isn't available.
func (c *Client) TownName(lat, lng float64) (string, error) {
	key := fmt.Sprintf("%.2f,%.2f", lat, lng)

	c.mu.Lock()
	if name, ok := c.cache[key]; ok {
		c.mu.Unlock()
		return name, nil
	}
	if wait := minInterval - time.Since(c.lastCall); wait > 0 {
		time.Sleep(wait)
	}
	c.lastCall = time.Now()
	c.mu.Unlock()

	req, err := http.NewRequest(http.MethodGet, reverseURL, nil)
	if err != nil {
		return "", err
	}
	q := req.URL.Query()
	q.Set("lat", strconv.FormatFloat(lat, 'f', 6, 64))
	q.Set("lon", strconv.FormatFloat(lng, 'f', 6, 64))
	q.Set("format", "json")
	q.Set("zoom", "14")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("nominatim reverse geocode failed: %s: %s", resp.Status, string(body))
	}

	var rr reverseResponse
	if err := json.Unmarshal(body, &rr); err != nil {
		return "", err
	}
	name := firstNonEmpty(rr.Address.City, rr.Address.Town, rr.Address.Village, rr.Address.Suburb, rr.Address.County, rr.Address.State)

	c.mu.Lock()
	c.cache[key] = name
	c.mu.Unlock()

	return name, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

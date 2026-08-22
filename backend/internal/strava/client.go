package strava

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrAPIInactive is returned when Strava rejects the request because the
// app's API access is inactive (requires a paid subscriber account).
var ErrAPIInactive = errors.New("strava API access inactive — a Strava subscription is required")

const (
	authorizeURL = "https://www.strava.com/oauth/authorize"
	tokenURL     = "https://www.strava.com/oauth/token"
	apiBase      = "https://www.strava.com/api/v3"
)

type Client struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	httpClient   *http.Client
}

func NewClient(clientID, clientSecret, redirectURI string) *Client {
	return &Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) AuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", c.RedirectURI)
	q.Set("response_type", "code")
	q.Set("approval_prompt", "auto")
	q.Set("scope", "activity:read_all")
	q.Set("state", state)
	return authorizeURL + "?" + q.Encode()
}

func (c *Client) ExchangeCode(code string) (TokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", c.ClientID)
	form.Set("client_secret", c.ClientSecret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	return c.postToken(form)
}

func (c *Client) RefreshToken(refreshToken string) (TokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", c.ClientID)
	form.Set("client_secret", c.ClientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")
	return c.postToken(form)
}

func (c *Client) postToken(form url.Values) (TokenResponse, error) {
	var tr TokenResponse
	resp, err := c.httpClient.PostForm(tokenURL, form)
	if err != nil {
		return tr, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tr, err
	}
	if resp.StatusCode != http.StatusOK {
		return tr, fmt.Errorf("strava token request failed: %s: %s", resp.Status, string(body))
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return tr, err
	}
	return tr, nil
}

// ListAllActivities fetches the athlete's entire activity history,
// paginating until Strava returns a short page.
func (c *Client) ListAllActivities(accessToken string) ([]Activity, error) {
	var all []Activity
	const perPage = 200
	for page := 1; ; page++ {
		q := url.Values{}
		q.Set("per_page", strconv.Itoa(perPage))
		q.Set("page", strconv.Itoa(page))

		req, err := http.NewRequest(http.MethodGet, apiBase+"/athlete/activities?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusForbidden && strings.Contains(string(body), "Inactive") {
				return nil, ErrAPIInactive
			}
			return nil, fmt.Errorf("strava activities request failed: %s: %s", resp.Status, string(body))
		}

		var pageActivities []Activity
		if err := json.Unmarshal(body, &pageActivities); err != nil {
			return nil, err
		}
		all = append(all, pageActivities...)
		if len(pageActivities) < perPage {
			break
		}
	}
	return all, nil
}

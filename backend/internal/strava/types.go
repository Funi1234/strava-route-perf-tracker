package strava

import "time"

type Athlete struct {
	ID        int64  `json:"id"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
}

type TokenResponse struct {
	TokenType    string  `json:"token_type"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	ExpiresAt    int64   `json:"expires_at"`
	ExpiresIn    int64   `json:"expires_in"`
	Athlete      Athlete `json:"athlete"`
}

// Activity mirrors the fields we need from Strava's
// GET /athlete/activities summary response.
type Activity struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	Distance         float64   `json:"distance"`    // meters
	MovingTime       int       `json:"moving_time"` // seconds
	Type             string    `json:"type"`
	SportType        string    `json:"sport_type"`
	StartDate        time.Time `json:"start_date"`
	StartLatLng      []float64 `json:"start_latlng"` // [lat, lng], omitted/null for indoor activities
	EndLatLng        []float64 `json:"end_latlng"`
	AverageSpeed     float64   `json:"average_speed"` // m/s
	HasHeartrate     bool      `json:"has_heartrate"`
	AverageHeartrate float64   `json:"average_heartrate"`
}

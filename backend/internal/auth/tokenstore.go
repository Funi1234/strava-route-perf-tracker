package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Tokens holds a single user's Strava OAuth tokens. This app is single-user,
// so tokens are persisted to one local file rather than a database.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"` // unix seconds
}

func (t Tokens) Valid() bool {
	return t.AccessToken != "" && t.RefreshToken != ""
}

// Expired reports whether the access token is expired or about to expire.
func (t Tokens) Expired() bool {
	return time.Now().Unix() >= t.ExpiresAt-60
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() (Tokens, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var t Tokens
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return t, nil
	}
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(data, &t); err != nil {
		return t, err
	}
	return t, nil
}

func (s *Store) Save(t Tokens) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

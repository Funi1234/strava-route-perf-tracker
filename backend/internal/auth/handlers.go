package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"strava-routes/backend/internal/strava"
)

var errNotAuthenticated = errors.New("not authenticated with strava")

type Handler struct {
	Client      *strava.Client
	Store       *Store
	FrontendURL string
}

func NewHandler(client *strava.Client, store *Store, frontendURL string) *Handler {
	return &Handler{Client: client, Store: store, FrontendURL: frontendURL}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.Client.AuthorizeURL(randomState()), http.StatusFound)
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		http.Redirect(w, r, h.FrontendURL+"?strava_error="+errParam, http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	tr, err := h.Client.ExchangeCode(code)
	if err != nil {
		log.Printf("strava code exchange failed: %v", err)
		http.Error(w, "failed to authenticate with strava", http.StatusBadGateway)
		return
	}

	if err := h.Store.Save(Tokens{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    tr.ExpiresAt,
	}); err != nil {
		log.Printf("failed to save tokens: %v", err)
		http.Error(w, "failed to persist session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, h.FrontendURL, http.StatusFound)
}

func (h *Handler) Session(w http.ResponseWriter, r *http.Request) {
	t, err := h.Store.Load()
	authenticated := err == nil && t.Valid()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"authenticated": authenticated})
}

// EnsureAccessToken returns a valid access token for calling the Strava API,
// refreshing and persisting a new one first if the current one has expired.
func (h *Handler) EnsureAccessToken() (string, error) {
	t, err := h.Store.Load()
	if err != nil {
		return "", err
	}
	if !t.Valid() {
		return "", errNotAuthenticated
	}
	if !t.Expired() {
		return t.AccessToken, nil
	}

	tr, err := h.Client.RefreshToken(t.RefreshToken)
	if err != nil {
		return "", err
	}
	newTokens := Tokens{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    tr.ExpiresAt,
	}
	if err := h.Store.Save(newTokens); err != nil {
		return "", err
	}
	return newTokens.AccessToken, nil
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

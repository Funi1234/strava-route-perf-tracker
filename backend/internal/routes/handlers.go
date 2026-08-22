package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"strava-routes/backend/internal/archive"
	"strava-routes/backend/internal/strava"
)

type TokenProvider interface {
	EnsureAccessToken() (string, error)
}

// Handler serves route/activity data computed from the most recent sync.
// Data is cached in memory since it's cheap to recompute from a full
// history fetch — no persistence needed beyond the Strava tokens.
type Handler struct {
	StravaClient *strava.Client
	Tokens       TokenProvider
	Geocoder     TownNamer

	mu     sync.RWMutex
	routes []Route
	synced bool
}

func NewHandler(client *strava.Client, tokens TokenProvider, geocoder TownNamer) *Handler {
	return &Handler{StravaClient: client, Tokens: tokens, Geocoder: geocoder}
}

type routeSummary struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	DistanceKm    float64 `json:"distanceKm"`
	ActivityCount int     `json:"activityCount"`
	LastRunAt     string  `json:"lastRunAt"`
}

type activityDTO struct {
	ID            int64    `json:"id"`
	Label         string   `json:"label"`
	StartDate     string   `json:"startDate"`
	DistanceKm    float64  `json:"distanceKm"`
	MovingTimeSec int      `json:"movingTimeSec"`
	AvgHeartrate  *float64 `json:"avgHeartrate"`
	PaceSecPerKm  *float64 `json:"paceSecPerKm"`
	StravaURL     string   `json:"stravaUrl"`
}

func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	accessToken, err := h.Tokens.EnsureAccessToken()
	if err != nil {
		http.Error(w, "not authenticated with strava", http.StatusUnauthorized)
		return
	}

	stravaActivities, err := h.StravaClient.ListAllActivities(accessToken)
	if err != nil {
		log.Printf("failed to fetch strava activities: %v", err)
		if errors.Is(err, strava.ErrAPIInactive) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			_ = json.NewEncoder(w).Encode(map[string]string{"code": "api_inactive"})
			return
		}
		http.Error(w, "failed to fetch activities from strava", http.StatusBadGateway)
		return
	}

	clustered := NameRoutes(Cluster(FromStravaActivities(stravaActivities)), h.Geocoder)

	h.mu.Lock()
	h.routes = clustered
	h.synced = true
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"routeCount": len(clustered)})
}

func (h *Handler) ListRoutes(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.synced {
		http.Error(w, "no data yet, call /api/sync first", http.StatusPreconditionRequired)
		return
	}

	// Routes run only once don't have a trend to show and would otherwise
	// flood this list once a full activity history is synced, so only
	// repeated routes are surfaced here.
	out := make([]routeSummary, 0)
	for _, rt := range h.routes {
		if len(rt.Activities) < 2 {
			continue
		}
		last := rt.Activities[0].StartDate
		var totalDist float64
		for _, a := range rt.Activities {
			if a.StartDate.After(last) {
				last = a.StartDate
			}
			totalDist += a.DistanceMeters
		}
		out = append(out, routeSummary{
			ID:            rt.ID,
			Name:          rt.Name,
			DistanceKm:    round2(totalDist / float64(len(rt.Activities)) / 1000),
			ActivityCount: len(rt.Activities),
			LastRunAt:     last.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) ListRouteActivities(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid route id", http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.synced {
		http.Error(w, "no data yet, call /api/sync first", http.StatusPreconditionRequired)
		return
	}

	var route *Route
	for i := range h.routes {
		if h.routes[i].ID == id {
			route = &h.routes[i]
			break
		}
	}
	if route == nil {
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}

	sorted := make([]RunActivity, len(route.Activities))
	copy(sorted, route.Activities)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartDate.Before(sorted[j].StartDate) })

	out := make([]activityDTO, len(sorted))
	for i, a := range sorted {
		dto := activityDTO{
			ID:            a.ID,
			Label:         a.StartDate.Format("Jan 2, 2006 · 3:04 PM"),
			StartDate:     a.StartDate.Format(time.RFC3339),
			DistanceKm:    round2(a.DistanceMeters / 1000),
			MovingTimeSec: a.MovingTimeSec,
			StravaURL:     fmt.Sprintf("https://www.strava.com/activities/%d", a.ID),
		}
		if a.HasHeartrate {
			hr := round1(a.AverageHeartrate)
			dto.AvgHeartrate = &hr
		}
		if a.AverageSpeed > 0 {
			pace := round1(1000 / a.AverageSpeed)
			dto.PaceSecPerKm = &pace
		}
		out[i] = dto
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	const maxSize = 500 << 20 // 500 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "failed to parse upload", http.StatusBadRequest)
		return
	}

	f, _, err := r.FormFile("archive")
	if err != nil {
		http.Error(w, "missing archive file", http.StatusBadRequest)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "failed to read upload", http.StatusBadRequest)
		return
	}

	archiveActivities, err := archive.ParseZip(data)
	if err != nil {
		log.Printf("failed to parse archive: %v", err)
		http.Error(w, "failed to parse archive zip", http.StatusBadRequest)
		return
	}

	clustered := NameRoutes(Cluster(FromArchiveActivities(archiveActivities)), h.Geocoder)

	h.mu.Lock()
	h.routes = clustered
	h.synced = true
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"routeCount": len(clustered)})
}

// FromArchiveActivities converts parsed archive activities into RunActivity,
// dropping any without GPS coordinates.
func FromArchiveActivities(activities []archive.Activity) []RunActivity {
	var out []RunActivity
	for _, a := range activities {
		if a.StartLat == 0 && a.StartLng == 0 {
			continue
		}
		out = append(out, RunActivity{
			ID:               a.ID,
			Name:             a.Name,
			StartDate:        a.StartDate,
			DistanceMeters:   a.DistanceMeters,
			MovingTimeSec:    a.MovingTimeSec,
			AverageSpeed:     a.AverageSpeed,
			HasHeartrate:     a.HasHeartrate,
			AverageHeartrate: a.AverageHeartrate,
			StartLat:         a.StartLat,
			StartLng:         a.StartLng,
			MidLat:           a.MidLat,
			MidLng:           a.MidLng,
			EndLat:           a.EndLat,
			EndLng:           a.EndLng,
		})
	}
	return out
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
func round1(f float64) float64 { return math.Round(f*10) / 10 }

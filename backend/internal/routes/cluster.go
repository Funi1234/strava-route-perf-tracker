package routes

import (
	"math"
	"sort"
	"time"

	"strava-routes/backend/internal/strava"
)

// Strava's API has no native "same route" field — only per-segment matching
// via segment_efforts. Routes here are inferred with a GPS proximity
// heuristic: activities are grouped together when their start points, end
// points, and overall distance are all close to an existing group's running
// average. These thresholds are deliberately simple and tunable.
const (
	startEndThresholdMeters = 150.0
	distanceTolerancePct    = 0.20
	earthRadiusMeters       = 6371000.0
)

type RunActivity struct {
	ID                 int64
	Name               string
	StartDate          time.Time
	DistanceMeters     float64
	MovingTimeSec      int
	AverageSpeed       float64 // m/s
	HasHeartrate       bool
	AverageHeartrate   float64
	StartLat, StartLng float64
	MidLat, MidLng     float64 // midpoint of GPS track; zero if unavailable
	EndLat, EndLng     float64
}

type Route struct {
	ID         int
	Name       string
	Activities []RunActivity

	startCentroid [2]float64
	midCentroid   [2]float64
	endCentroid   [2]float64
	avgDistance   float64
	hasMid        bool // true once a midpoint has been recorded
}

// FromStravaActivities converts raw Strava activities into RunActivity,
// dropping any without GPS start/end coordinates (e.g. indoor/trainer
// activities), since those can't be assigned to a route.
func FromStravaActivities(activities []strava.Activity) []RunActivity {
	var out []RunActivity
	for _, a := range activities {
		if len(a.StartLatLng) != 2 || len(a.EndLatLng) != 2 {
			continue
		}
		out = append(out, RunActivity{
			ID:               a.ID,
			Name:             a.Name,
			StartDate:        a.StartDate,
			DistanceMeters:   a.Distance,
			MovingTimeSec:    a.MovingTime,
			AverageSpeed:     a.AverageSpeed,
			HasHeartrate:     a.HasHeartrate,
			AverageHeartrate: a.AverageHeartrate,
			StartLat:         a.StartLatLng[0],
			StartLng:         a.StartLatLng[1],
			EndLat:           a.EndLatLng[0],
			EndLng:           a.EndLatLng[1],
		})
	}
	return out
}

func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// Cluster groups activities into inferred routes. Activities are processed
// oldest-first and greedily assigned to the first matching cluster, whose
// centroid is then updated with a running average.
func Cluster(activities []RunActivity) []Route {
	sorted := make([]RunActivity, len(activities))
	copy(sorted, activities)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartDate.Before(sorted[j].StartDate) })

	var clusters []*Route
	nextID := 1

	for _, act := range sorted {
		hasMid := act.MidLat != 0 || act.MidLng != 0

		var match *Route
		for _, cl := range clusters {
			startDist := haversineMeters(act.StartLat, act.StartLng, cl.startCentroid[0], cl.startCentroid[1])
			endDist := haversineMeters(act.EndLat, act.EndLng, cl.endCentroid[0], cl.endCentroid[1])
			distDiff := math.Abs(act.DistanceMeters-cl.avgDistance) / cl.avgDistance
			if startDist > startEndThresholdMeters || endDist > startEndThresholdMeters || distDiff > distanceTolerancePct {
				continue
			}
			// If both this activity and the cluster have a midpoint, check it
			// too. This prevents activities that start/end at the same place
			// (e.g. home) but follow completely different paths from merging.
			if hasMid && cl.hasMid {
				midDist := haversineMeters(act.MidLat, act.MidLng, cl.midCentroid[0], cl.midCentroid[1])
				if midDist > startEndThresholdMeters {
					continue
				}
			}
			match = cl
			break
		}
		if match == nil {
			match = &Route{
				ID:            nextID,
				startCentroid: [2]float64{act.StartLat, act.StartLng},
				midCentroid:   [2]float64{act.MidLat, act.MidLng},
				endCentroid:   [2]float64{act.EndLat, act.EndLng},
				avgDistance:   act.DistanceMeters,
				hasMid:        hasMid,
			}
			nextID++
			clusters = append(clusters, match)
		}

		match.Activities = append(match.Activities, act)
		n := float64(len(match.Activities))
		match.startCentroid[0] += (act.StartLat - match.startCentroid[0]) / n
		match.startCentroid[1] += (act.StartLng - match.startCentroid[1]) / n
		match.endCentroid[0] += (act.EndLat - match.endCentroid[0]) / n
		match.endCentroid[1] += (act.EndLng - match.endCentroid[1]) / n
		match.avgDistance += (act.DistanceMeters - match.avgDistance) / n
		if hasMid {
			if !match.hasMid {
				match.midCentroid = [2]float64{act.MidLat, act.MidLng}
				match.hasMid = true
			} else {
				match.midCentroid[0] += (act.MidLat - match.midCentroid[0]) / n
				match.midCentroid[1] += (act.MidLng - match.midCentroid[1]) / n
			}
		}
	}

	sort.Slice(clusters, func(i, j int) bool { return len(clusters[i].Activities) > len(clusters[j].Activities) })

	result := make([]Route, len(clusters))
	for i, cl := range clusters {
		result[i] = *cl
	}
	return result
}

// StartCentroid returns the running-average start coordinate for this
// route's activities, used to reverse-geocode a name for it.
func (r Route) StartCentroid() (lat, lng float64) {
	return r.startCentroid[0], r.startCentroid[1]
}

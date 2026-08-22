# Strava Route Performance Tracker

Pulls your full Strava activity history, groups activities into routes by
GPS start/end proximity (Strava's API has no native "same route" field —
see note below), names each route after the town it's in, and charts pace +
heart rate trend per route. Each activity links back to its page on Strava.

## Setup

### 1. Strava API app

Already done for this project — credentials are in `backend/.env`
(gitignored). If you need to recreate it: register an app at
[strava.com/settings/api](https://www.strava.com/settings/api) with
Authorization Callback Domain `localhost`, then fill in
`backend/.env` (see `backend/.env.example`) with `STRAVA_CLIENT_ID` /
`STRAVA_CLIENT_SECRET`.

### 2. Run the backend

```bash
cd backend
go mod tidy   # first time only
go run .
```

Listens on `http://localhost:8080`.

### 3. Run the frontend

```bash
cd frontend
npm install   # first time only
npm run dev
```

Open `http://localhost:3000`.

### 4. Connect your Strava account

Click "Connect with Strava" and log in — this step is yours to do since it
needs your real Strava credentials. You'll be redirected back once
authorized, and the app will sync your entire activity history. For a long
history this can take a little while (many pages of activities to fetch,
plus rate-limited reverse-geocoding — see below).

## How "route" grouping works

Strava's API doesn't have a `route_id` linking activities together — the
only cross-activity matching it provides is per-segment (`segment_efforts`),
not whole-route. So this app infers routes itself: activities are grouped
together when their start point, end point, and total distance are all
close to an existing group's running average (150m proximity, ±20% distance
tolerance — see `backend/internal/routes/cluster.go`). Activities without
GPS data (e.g. indoor/trainer) are excluded since they can't be matched.

## How route naming works

Strava doesn't reliably expose a town/city name per activity either, so
each route's name is derived by reverse-geocoding its start location via
OpenStreetMap's free Nominatim API (`backend/internal/geocode`), then
formatted as `{Town} Route {n}` — e.g. "Cupertino Route 1". Numbering is
per-town, with the most-run route in a town becoming Route 1. Lookups are
cached and rate-limited to one request/second per Nominatim's usage policy.

## Project layout

- `backend/` — Go API server: Strava OAuth, activity fetching, route
  clustering, reverse-geocoded naming, JSON endpoints (`internal/auth`,
  `internal/strava`, `internal/routes`, `internal/geocode`).
- `frontend/` — Next.js App Router UI: route list (`app/page.tsx`) and
  per-route activity list + trend chart (`app/routes/[id]/page.tsx`).

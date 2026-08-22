# Strava Route Performance Tracker

Pulls your full Strava activity history, groups activities into routes by
GPS start/end proximity (Strava's API has no native "same route" field —
see note below), names each route after the town it's in, and charts pace +
heart rate trend per route. Each activity links back to its page on Strava.

## Setup

### 1. Strava API app

Register an app at [strava.com/settings/api](https://www.strava.com/settings/api)
with Authorization Callback Domain `localhost`. Copy your **Client ID** and
**Client Secret** — you'll need them in the next step.

> **Note:** Strava's API now requires a paid subscription to fetch activity
> data. If you don't have one, the app will prompt you to upload your Strava
> data archive instead (see [Using your data archive](#using-your-data-archive)).

---

## Running with Docker

```bash
cp .env.example .env
# fill in STRAVA_CLIENT_ID and STRAVA_CLIENT_SECRET
docker compose up --build
```

Open `http://localhost:3000`. The backend and frontend start together.
Strava OAuth tokens are persisted in a named Docker volume so you stay
logged in across restarts.

---

## Running locally

### 2. Configure credentials

```bash
cp backend/.env.example backend/.env
# fill in STRAVA_CLIENT_ID and STRAVA_CLIENT_SECRET
```

### 3. Run the backend

```bash
cd backend
go mod tidy   # first time only
go run .
```

Listens on `http://localhost:8080`.

### 4. Run the frontend

```bash
cd frontend
npm install   # first time only
npm run dev
```

Open `http://localhost:3000`.

---

### 5. Connect your Strava account

Click **Connect with Strava** and log in. You'll be redirected back once
authorized. If your account has API access the app syncs your full activity
history automatically. For a long history this can take a little while (many
pages of activities to fetch, plus rate-limited reverse-geocoding — see below).

## Using your data archive

If Strava's API is inactive on your account, the app will offer a second
option: upload your full data archive.

1. Strava → Settings → My Account → **Download or Delete Your Account**
2. Request your archive and wait for the email
3. Upload the `.zip` file in the app

The archive contains your complete activity history including GPX tracks
(older activities recorded via phone) and FIT files (newer activities,
including Apple Watch). Both formats are supported.

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
  `internal/strava`, `internal/routes`, `internal/geocode`,
  `internal/archive`).
- `frontend/` — Next.js App Router UI: route list (`app/page.tsx`) and
  per-route activity list + trend chart (`app/routes/[id]/page.tsx`).
- `docker-compose.yml` — runs backend + frontend together with a named
  volume for token persistence.

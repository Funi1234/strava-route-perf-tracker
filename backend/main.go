package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"strava-routes/backend/internal/auth"
	"strava-routes/backend/internal/geocode"
	"strava-routes/backend/internal/routes"
	"strava-routes/backend/internal/strava"
)

func main() {
	_ = godotenv.Load()

	clientID := requireEnv("STRAVA_CLIENT_ID")
	clientSecret := requireEnv("STRAVA_CLIENT_SECRET")
	redirectURI := getEnv("STRAVA_REDIRECT_URI", "http://localhost:8080/auth/callback")
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3000")
	port := getEnv("PORT", "8080")

	stravaClient := strava.NewClient(clientID, clientSecret, redirectURI)
	tokenStore := auth.NewStore("data/tokens.json")
	authHandler := auth.NewHandler(stravaClient, tokenStore, frontendURL)
	geocoder := geocode.NewClient("strava-route-tracker/1.0 (personal use)")
	routesHandler := routes.NewHandler(stravaClient, authHandler, geocoder)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/login", authHandler.Login)
	mux.HandleFunc("GET /auth/callback", authHandler.Callback)
	mux.HandleFunc("GET /api/session", authHandler.Session)
	mux.HandleFunc("POST /api/sync", routesHandler.Sync)
	mux.HandleFunc("POST /api/upload", routesHandler.Upload)
	mux.HandleFunc("GET /api/routes", routesHandler.ListRoutes)
	mux.HandleFunc("GET /api/routes/{id}/activities", routesHandler.ListRouteActivities)

	log.Printf("backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, withCORS(mux, frontendURL)); err != nil {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var %s", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export class APIInactiveError extends Error {
  constructor() {
    super("Strava API access requires a paid subscription");
    this.name = "APIInactiveError";
  }
}

export type RouteSummary = {
  id: number;
  name: string;
  distanceKm: number;
  activityCount: number;
  lastRunAt: string;
};

export type RouteActivity = {
  id: number;
  label: string;
  startDate: string;
  distanceKm: number;
  movingTimeSec: number;
  avgHeartrate: number | null;
  paceSecPerKm: number | null;
  stravaUrl: string;
};

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, init);
  if (!res.ok) {
    throw new Error(`${init?.method ?? "GET"} ${path} failed: ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export function getSession() {
  return apiFetch<{ authenticated: boolean }>("/api/session");
}

export async function syncActivities(): Promise<{ routeCount: number }> {
  const res = await fetch(`${API_BASE}/api/sync`, { method: "POST" });
  if (res.status === 402) throw new APIInactiveError();
  if (!res.ok) throw new Error(`POST /api/sync failed: ${res.status}`);
  return res.json() as Promise<{ routeCount: number }>;
}

export async function uploadArchive(file: File): Promise<{ routeCount: number }> {
  const form = new FormData();
  form.append("archive", file);
  const res = await fetch(`${API_BASE}/api/upload`, { method: "POST", body: form });
  if (!res.ok) throw new Error(`POST /api/upload failed: ${res.status}`);
  return res.json() as Promise<{ routeCount: number }>;
}

export function listRoutes() {
  return apiFetch<RouteSummary[]>("/api/routes");
}

export function listRouteActivities(routeId: string | number) {
  return apiFetch<RouteActivity[]>(`/api/routes/${routeId}/activities`);
}

export function loginUrl() {
  return `${API_BASE}/auth/login`;
}

export function formatPace(secPerKm: number | null): string {
  if (secPerKm == null) return "—";
  const min = Math.floor(secPerKm / 60);
  const sec = Math.round(secPerKm % 60);
  return `${min}:${sec.toString().padStart(2, "0")}/km`;
}

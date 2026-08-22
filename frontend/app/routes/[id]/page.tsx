"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import ActivityList from "@/components/ActivityList";
import PerformanceChart from "@/components/PerformanceChart";
import { listRouteActivities, listRoutes, type RouteActivity } from "@/lib/api";

export default function RoutePage() {
  const params = useParams<{ id: string }>();
  const [activities, setActivities] = useState<RouteActivity[] | null>(null);
  const [routeName, setRouteName] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([listRouteActivities(params.id), listRoutes()])
      .then(([activityData, routes]) => {
        setActivities(activityData);
        const match = routes.find((r) => String(r.id) === params.id);
        setRouteName(match?.name ?? null);
      })
      .catch((e) => setError(e instanceof Error ? e.message : String(e)));
  }, [params.id]);

  return (
    <main className="max-w-xl mx-auto w-full px-6 py-16 flex-1">
      <Link href="/" className="text-sm text-neutral-500 hover:underline">
        ← All routes
      </Link>
      <h1 className="text-2xl font-semibold mt-2 mb-6">{routeName ?? `Route ${params.id}`}</h1>

      {error && <p className="text-sm text-red-600">Something went wrong: {error}</p>}
      {!activities && !error && <p className="text-sm text-neutral-500">Loading…</p>}

      {activities && (
        <>
          <PerformanceChart activities={activities} />
          <ActivityList activities={activities} />
        </>
      )}
    </main>
  );
}

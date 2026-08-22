"use client";

import { useEffect, useState } from "react";
import RouteList from "@/components/RouteList";
import { getSession, listRoutes, loginUrl, syncActivities, type RouteSummary } from "@/lib/api";

type Status = "checking" | "loggedOut" | "syncing" | "ready" | "error";

export default function Home() {
  const [status, setStatus] = useState<Status>("checking");
  const [routes, setRoutes] = useState<RouteSummary[]>([]);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    try {
      const session = await getSession();
      if (!session.authenticated) {
        setStatus("loggedOut");
        return;
      }
      setStatus("syncing");
      await syncActivities();
      setRoutes(await listRoutes());
      setStatus("ready");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setStatus("error");
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- fetch-on-mount; every setState call below happens after an await
    load();
  }, []);

  return (
    <main className="max-w-xl mx-auto w-full px-6 py-16 flex-1">
      <h1 className="text-2xl font-semibold mb-1">Routes</h1>
      <p className="text-sm text-neutral-500 mb-8">Your full activity history, grouped by route.</p>

      {status === "checking" && <p className="text-sm text-neutral-500">Loading…</p>}

      {status === "loggedOut" && (
        <a
          href={loginUrl()}
          className="inline-block rounded-lg bg-orange-500 text-white px-4 py-2 text-sm font-medium hover:bg-orange-600 transition-colors"
        >
          Connect with Strava
        </a>
      )}

      {status === "syncing" && (
        <p className="text-sm text-neutral-500">
          Syncing your full history with Strava — this can take a bit longer for a long activity
          history…
        </p>
      )}

      {status === "error" && (
        <div className="text-sm text-red-600">
          <p>Something went wrong: {error}</p>
          <button onClick={load} className="underline mt-2">
            Try again
          </button>
        </div>
      )}

      {status === "ready" && (
        <>
          <RouteList routes={routes} />
          <button
            onClick={load}
            className="mt-6 text-sm text-neutral-500 underline hover:text-neutral-700 dark:hover:text-neutral-300"
          >
            Sync again
          </button>
        </>
      )}
    </main>
  );
}

"use client";

import Link from "next/link";
import type { RouteSummary } from "@/lib/api";

export default function RouteList({ routes }: { routes: RouteSummary[] }) {
  if (routes.length === 0) {
    return (
      <p className="text-sm text-neutral-500">
        No repeated routes found in the last week. Go for a run somewhere you&apos;ve run
        before, then sync again.
      </p>
    );
  }

  return (
    <ul className="divide-y divide-neutral-200 dark:divide-neutral-800">
      {routes.map((route) => (
        <li key={route.id}>
          <Link
            href={`/routes/${route.id}`}
            className="flex items-center justify-between py-4 hover:bg-neutral-50 dark:hover:bg-neutral-900 -mx-2 px-2 rounded-lg transition-colors"
          >
            <div>
              <p className="font-medium">{route.name}</p>
              <p className="text-sm text-neutral-500">
                {route.distanceKm.toFixed(1)} km · {route.activityCount}{" "}
                {route.activityCount === 1 ? "run" : "runs"}
              </p>
            </div>
            <span className="text-sm text-neutral-400">
              {new Date(route.lastRunAt).toLocaleDateString(undefined, {
                month: "short",
                day: "numeric",
              })}
            </span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

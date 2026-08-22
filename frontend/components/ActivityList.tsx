import type { RouteActivity } from "@/lib/api";
import { formatPace } from "@/lib/api";

export default function ActivityList({ activities }: { activities: RouteActivity[] }) {
  const sorted = [...activities].sort(
    (a, b) => new Date(b.startDate).getTime() - new Date(a.startDate).getTime()
  );

  return (
    <ul className="divide-y divide-neutral-200 dark:divide-neutral-800">
      {sorted.map((activity) => (
        <li key={activity.id}>
          <a
            href={activity.stravaUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center justify-between py-3 hover:bg-neutral-50 dark:hover:bg-neutral-900 -mx-2 px-2 rounded-lg transition-colors"
          >
            <div>
              <p className="font-medium">{activity.label}</p>
              <p className="text-sm text-neutral-500">{activity.distanceKm.toFixed(2)} km</p>
            </div>
            <div className="text-right text-sm text-neutral-500">
              <p>{formatPace(activity.paceSecPerKm)}</p>
              <p>{activity.avgHeartrate != null ? `${Math.round(activity.avgHeartrate)} bpm` : "—"}</p>
            </div>
          </a>
        </li>
      ))}
    </ul>
  );
}

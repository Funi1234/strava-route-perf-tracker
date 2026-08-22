"use client";

import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { RouteActivity } from "@/lib/api";
import { formatPace } from "@/lib/api";

export default function PerformanceChart({ activities }: { activities: RouteActivity[] }) {
  const data = [...activities]
    .sort((a, b) => new Date(a.startDate).getTime() - new Date(b.startDate).getTime())
    .map((a) => ({
      date: new Date(a.startDate).toLocaleDateString(undefined, { month: "short", day: "numeric" }),
      pace: a.paceSecPerKm,
      heartrate: a.avgHeartrate,
    }));

  if (data.length < 2) {
    return (
      <p className="text-sm text-neutral-500 mb-6">
        Need at least two runs on this route to show a trend.
      </p>
    );
  }

  return (
    <div className="h-64 w-full mb-8">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 16, bottom: 0, left: 0 }}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-neutral-200 dark:stroke-neutral-800" />
          <XAxis dataKey="date" tick={{ fontSize: 12 }} />
          <YAxis
            yAxisId="pace"
            tick={{ fontSize: 12 }}
            tickFormatter={(v) => formatPace(v)}
            width={60}
            reversed
            domain={[225, "dataMax"]}
          />
          <YAxis
            yAxisId="hr"
            orientation="right"
            tick={{ fontSize: 12 }}
            width={40}
            reversed
            domain={[120, "dataMax"]}
          />
          <Tooltip
            formatter={(value, name) =>
              name === "pace"
                ? [formatPace(Number(value)), "Pace"]
                : [`${Math.round(Number(value))} bpm`, "Heart rate"]
            }
          />
          <Line
            yAxisId="pace"
            type="monotone"
            dataKey="pace"
            stroke="#f97316"
            strokeWidth={2}
            dot={{ r: 3 }}
            connectNulls
            name="pace"
          />
          <Line
            yAxisId="hr"
            type="monotone"
            dataKey="heartrate"
            stroke="#3b82f6"
            strokeWidth={2}
            dot={{ r: 3 }}
            connectNulls
            name="heartrate"
          />
        </LineChart>
      </ResponsiveContainer>
      <div className="flex gap-4 text-xs text-neutral-500 mt-2">
        <span className="flex items-center gap-1">
          <span className="w-2 h-2 rounded-full bg-orange-500 inline-block" /> Pace
        </span>
        <span className="flex items-center gap-1">
          <span className="w-2 h-2 rounded-full bg-blue-500 inline-block" /> Heart rate
        </span>
        <span className="text-neutral-400">(both axes reversed — up the chart means better)</span>
      </div>
    </div>
  );
}

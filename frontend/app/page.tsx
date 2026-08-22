"use client";

import { useEffect, useRef, useState } from "react";
import RouteList from "@/components/RouteList";
import {
  APIInactiveError,
  getSession,
  listRoutes,
  loginUrl,
  syncActivities,
  uploadArchive,
  type RouteSummary,
} from "@/lib/api";

type Status = "checking" | "loggedOut" | "syncing" | "ready" | "error" | "upload" | "uploading";

export default function Home() {
  const [status, setStatus] = useState<Status>("checking");
  const [routes, setRoutes] = useState<RouteSummary[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

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
      if (e instanceof APIInactiveError) {
        setStatus("upload");
        return;
      }
      setError(e instanceof Error ? e.message : String(e));
      setStatus("error");
    }
  }

  async function handleUpload() {
    if (!uploadFile) return;
    setStatus("uploading");
    try {
      await uploadArchive(uploadFile);
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

      {status === "upload" && (
        <div className="space-y-6">
          <div className="space-y-3">
            <h2 className="text-sm font-medium">Sync with Strava</h2>
            <p className="text-sm text-neutral-500">
              Strava&apos;s API requires a paid subscription. If you have one, try syncing directly.
            </p>
            <button
              onClick={load}
              className="inline-block rounded-lg bg-orange-500 text-white px-4 py-2 text-sm font-medium hover:bg-orange-600 transition-colors"
            >
              Try Sync with Strava
            </button>
          </div>

          <div className="border-t border-neutral-200 dark:border-neutral-700 pt-6 space-y-3">
            <h2 className="text-sm font-medium">Upload your Strava archive</h2>
            <p className="text-sm text-neutral-500">
              No subscription? Download your data from Strava and upload it here.
            </p>
            <ol className="text-sm text-neutral-500 list-decimal list-inside space-y-1">
              <li>Strava → Settings → My Account</li>
              <li>Download or Delete Your Account → Request Your Archive</li>
              <li>Upload the zip file you receive by email</li>
            </ol>
            <div className="flex items-center gap-3">
              <input
                ref={fileInputRef}
                type="file"
                accept=".zip"
                className="hidden"
                onChange={(e) => setUploadFile(e.target.files?.[0] ?? null)}
              />
              <button
                onClick={() => fileInputRef.current?.click()}
                className="rounded-lg border border-neutral-300 dark:border-neutral-600 px-4 py-2 text-sm hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
              >
                {uploadFile ? uploadFile.name : "Choose archive zip"}
              </button>
              <button
                onClick={handleUpload}
                disabled={!uploadFile}
                className="rounded-lg bg-orange-500 text-white px-4 py-2 text-sm font-medium hover:bg-orange-600 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                Upload
              </button>
            </div>
          </div>
        </div>
      )}

      {status === "uploading" && (
        <p className="text-sm text-neutral-500">
          Processing your archive — parsing GPX files and grouping routes…
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

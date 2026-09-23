"use client";

import { useEffect, useState } from "react";
import { StatCard } from "@/components/stat-card";
import { AchievementCard } from "@/components/achievement-card";
import { QualityChart } from "@/components/quality-chart";
import { HookLeaderboard } from "@/components/hook-leaderboard";
import { RecentRunsTable } from "@/components/recent-runs-table";
import { Separator } from "@/components/ui/separator";
import type { StatsResponse } from "@/lib/types";
import { formatMinutesSaved } from "@/lib/format";

export default function Home() {
  const [data, setData] = useState<StatsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/stats")
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then(setData)
      .catch((e) => setError(String(e)));
  }, []);

  if (error) {
    return (
      <main className="flex-1 flex items-center justify-center p-8">
        <p className="text-destructive">Failed to load data: {error}</p>
      </main>
    );
  }

  if (!data) {
    return (
      <main className="flex-1 flex items-center justify-center p-8">
        <p className="text-muted-foreground">Loading…</p>
      </main>
    );
  }

  const { stats, achievements, daily, recent_runs, hook_stats } = data;

  return (
    <main className="flex-1 mx-auto w-full max-w-6xl px-4 sm:px-6 py-8 space-y-8">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold">Quality Gate Dashboard</h1>
        <p className="text-sm text-muted-foreground">
          Local execution history — data comes from{" "}
          <code className="text-xs">.git/quality-gate/history.jsonl</code>
        </p>
      </header>

      <section className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          icon="🔥"
          label="Streak"
          value={
            stats.streak_broken_today
              ? "0 days"
              : `${stats.streak_days} days`
          }
          hint={
            stats.streak_broken_today
              ? `Latest ${stats.streak_hook_type} build failed`
              : `without breaking ${stats.streak_hook_type}`
          }
        />
        <StatCard
          icon="📈"
          label="Success rate"
          value={`${stats.success_rate.toFixed(0)}%`}
          hint={`${stats.total_runs} total runs`}
        />
        <StatCard
          icon="⚡"
          label="Saved this month"
          value={formatMinutesSaved(stats.time_saved_this_month_minutes)}
          hint={`${stats.fixes_this_month} auto-fixes`}
        />
        <StatCard
          icon="🛠️"
          label="Total auto-fixes"
          value={String(stats.total_auto_fixes)}
          hint="since history began"
        />
      </section>

      <QualityChart daily={daily} />

      <HookLeaderboard hookStats={hook_stats} />

      <section className="space-y-3">
        <h2 className="text-lg font-semibold">🏅 Achievements</h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {achievements.map((a) => (
            <AchievementCard key={a.id} achievement={a} />
          ))}
        </div>
      </section>

      <Separator />

      <RecentRunsTable runs={recent_runs} />
    </main>
  );
}

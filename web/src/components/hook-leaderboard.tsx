import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import type { HookStat } from "@/lib/types";
import { formatDuration } from "@/lib/format";

function LeaderboardRow({
  name,
  value,
  pct,
}: {
  name: string;
  value: string;
  pct: number;
}) {
  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-sm">
        <span className="font-mono text-xs truncate pr-2">{name}</span>
        <span className="text-muted-foreground tabular-nums shrink-0">{value}</span>
      </div>
      <Progress value={pct} className="h-1.5" />
    </div>
  );
}

export function HookLeaderboard({ hookStats }: { hookStats: HookStat[] }) {
  const slowest = [...hookStats]
    .sort((a, b) => b.avg_duration_ms - a.avg_duration_ms)
    .slice(0, 5);
  const maxDuration = slowest[0]?.avg_duration_ms || 1;

  const flakiest = [...hookStats]
    .filter((h) => h.total_failures > 0)
    .sort((a, b) => b.failure_rate - a.failure_rate)
    .slice(0, 5);
  const maxFailureRate = Math.max(...flakiest.map((h) => h.failure_rate), 1);

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <Card>
        <CardHeader>
          <CardTitle>🐢 Slowest hooks</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {slowest.length === 0 && (
            <p className="text-sm text-muted-foreground">No hook durations recorded yet.</p>
          )}
          {slowest.map((h) => (
            <LeaderboardRow
              key={h.name}
              name={h.name}
              value={formatDuration(h.avg_duration_ms)}
              pct={(h.avg_duration_ms / maxDuration) * 100}
            />
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>🎲 Flakiest hooks</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {flakiest.length === 0 && (
            <p className="text-sm text-muted-foreground">No failures recorded yet — nice!</p>
          )}
          {flakiest.map((h) => (
            <LeaderboardRow
              key={h.name}
              name={h.name}
              value={`${h.failure_rate.toFixed(0)}% (${h.total_failures}/${h.total_runs})`}
              pct={(h.failure_rate / maxFailureRate) * 100}
            />
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

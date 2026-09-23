export interface Achievement {
  id: string;
  name: string;
  description: string;
  unlocked: boolean;
  progress: number;
  target: number;
}

export interface DailySummary {
  date: string; // YYYY-MM-DD
  total: number;
  failed: number;
  success_rate: number;
}

export interface HookOutcome {
  name: string;
  success: boolean;
  duration_ms: number;
}

export interface RunRecord {
  timestamp: string; // RFC3339
  hook_type: string;
  success: boolean;
  duration_ms: number;
  hooks: HookOutcome[];
}

export interface StatsSummary {
  total_runs: number;
  total_failed_runs: number;
  success_rate: number;
  streak_days: number;
  streak_hook_type: string;
  streak_broken_today: boolean;
  total_auto_fixes: number;
  fixes_this_month: number;
  time_saved_this_month_minutes: number;
}

export interface HookStat {
  name: string;
  total_runs: number;
  total_failures: number;
  failure_rate: number; // 0-100
  avg_duration_ms: number;
}

export interface StatsResponse {
  stats: StatsSummary;
  achievements: Achievement[];
  daily: DailySummary[];
  recent_runs: RunRecord[];
  hook_stats: HookStat[];
}

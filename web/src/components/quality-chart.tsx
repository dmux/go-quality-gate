"use client";

import { useState } from "react";
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import type { DailySummary } from "@/lib/types";
import { formatDayLabel } from "@/lib/format";

const chartConfig = {
  success_rate: {
    label: "Taxa de sucesso",
    color: "var(--chart-1)",
  },
} satisfies ChartConfig;

// Extracted (rather than defined inline in JSX) so they're directly unit
// testable without needing Recharts to actually render and trigger a
// tooltip in jsdom.
export function formatTooltipLabel(_value: unknown, payload?: readonly { payload?: unknown }[]): string {
  const d = payload?.[0]?.payload as { date?: string } | undefined;
  return d?.date ?? "";
}

export function formatTooltipValue(value: unknown, _name: unknown, item?: { payload?: unknown }): string {
  const p = item?.payload as DailySummary | undefined;
  const failed = p?.failed ?? 0;
  const total = p?.total ?? 0;
  return `${value}% (${total - failed}/${total} runs)`;
}

export function QualityChart({ daily }: { daily: DailySummary[] }) {
  const [range, setRange] = useState<"7" | "30">("30");
  const days = Number(range);
  const data = daily.slice(-days).map((d) => ({
    ...d,
    label: formatDayLabel(d.date),
  }));

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between space-y-0">
        <CardTitle>📈 Quality evolution</CardTitle>
        <Tabs value={range} onValueChange={(v) => setRange(v as "7" | "30")}>
          <TabsList>
            <TabsTrigger value="7">7 days</TabsTrigger>
            <TabsTrigger value="30">30 days</TabsTrigger>
          </TabsList>
        </Tabs>
      </CardHeader>
      <CardContent>
        <ChartContainer config={chartConfig} className="aspect-auto h-[260px] w-full">
          <AreaChart data={data} margin={{ left: 12, right: 12 }}>
            <defs>
              <linearGradient id="fillSuccessRate" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--color-success_rate)" stopOpacity={0.8} />
                <stop offset="95%" stopColor="var(--color-success_rate)" stopOpacity={0.05} />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} />
            <XAxis
              dataKey="label"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              minTickGap={24}
            />
            <YAxis
              domain={[0, 100]}
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              width={32}
            />
            <ChartTooltip
              content={
                <ChartTooltipContent
                  labelFormatter={formatTooltipLabel}
                  formatter={formatTooltipValue}
                />
              }
            />
            <Area
              dataKey="success_rate"
              type="monotone"
              fill="url(#fillSuccessRate)"
              stroke="var(--color-success_rate)"
            />
          </AreaChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}

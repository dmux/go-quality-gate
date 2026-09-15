import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import type { RunRecord } from "@/lib/types";
import { formatDuration, formatTimestamp } from "@/lib/format";

export function RecentRunsTable({ runs }: { runs: RunRecord[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>🕓 Recent runs</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>When</TableHead>
              <TableHead>Hook</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Duration</TableHead>
              <TableHead className="text-right">Checks</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {runs.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-muted-foreground">
                  No runs recorded yet.
                </TableCell>
              </TableRow>
            )}
            {runs.map((run, i) => (
              <TableRow key={`${run.timestamp}-${i}`}>
                <TableCell className="text-muted-foreground">
                  {formatTimestamp(run.timestamp)}
                </TableCell>
                <TableCell className="font-mono text-xs">{run.hook_type}</TableCell>
                <TableCell>
                  <Badge variant={run.success ? "default" : "destructive"}>
                    {run.success ? "✅ passed" : "❌ failed"}
                  </Badge>
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {formatDuration(run.duration_ms)}
                </TableCell>
                <TableCell className="text-right tabular-nums text-muted-foreground">
                  {run.hooks?.filter((h) => h.success).length ?? 0}/{run.hooks?.length ?? 0}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import type { Achievement } from "@/lib/types";

export function AchievementCard({ achievement }: { achievement: Achievement }) {
  const pct = achievement.target > 0
    ? Math.min(100, (achievement.progress / achievement.target) * 100)
    : 0;

  return (
    <Card className={achievement.unlocked ? "border-primary/40" : "opacity-70"}>
      <CardHeader className="pb-2 flex-row items-center justify-between space-y-0">
        <CardTitle className="text-sm font-semibold">
          {achievement.unlocked ? "🏆" : "🔒"} {achievement.name}
        </CardTitle>
        <Badge variant={achievement.unlocked ? "default" : "secondary"}>
          {achievement.unlocked ? "Unlocked" : `${achievement.progress}/${achievement.target}`}
        </Badge>
      </CardHeader>
      <CardContent className="space-y-2">
        <p className="text-xs text-muted-foreground">{achievement.description}</p>
        {!achievement.unlocked && <Progress value={pct} className="h-1.5" />}
      </CardContent>
    </Card>
  );
}

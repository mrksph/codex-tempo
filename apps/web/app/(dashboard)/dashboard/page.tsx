import type { ActivityRange } from "@/components/activity-range-select";
import { ActivityMetrics } from "@/components/activity-metrics";
import { ActivityTimelinePanel } from "@/components/activity-timeline-panel";
import { PageHeader } from "@/components/page-header";
import { ProjectTimeSection } from "@/components/project-time-section";
import { tempoFetch, calendarRecentRange, recentRange } from "@/lib/api/client";
import type { Project, Run, Summary } from "@/lib/api/types";

export const dynamic = "force-dynamic";

const ACTIVITY_RANGES: Record<ActivityRange, { days: number; title: string }> = {
  "24h": { days: 1, title: "Activity in the last 24 hours" },
  "7d": { days: 7, title: "Activity in the last 7 days" },
  "30d": { days: 30, title: "Activity in the last 30 days" },
  "90d": { days: 90, title: "Activity in the last 90 days" },
};

export default async function DashboardPage({ searchParams }: { searchParams: Promise<{ range?: string | string[] }> }) {
  const params = await searchParams;
  const requestedRange = Array.isArray(params.range) ? params.range[0] : params.range;
  const selectedRange: ActivityRange = requestedRange && requestedRange in ACTIVITY_RANGES ? requestedRange as ActivityRange : "24h";
  const activityRangeConfig = ACTIVITY_RANGES[selectedRange];
  const activityRange = selectedRange === "24h"
    ? recentRange(1)
    : calendarRecentRange(activityRangeConfig.days);
  const query = new URLSearchParams(activityRange).toString();
  const activityQuery = new URLSearchParams(activityRange).toString();
  const [summaryData, projectData, timelineData] = await Promise.all([
    tempoFetch<Summary>(`/api/v1/reports/summary?${query}`),
    tempoFetch<{ projects: Project[] }>("/api/v1/projects"),
    tempoFetch<{ from: string; to: string; runs: Run[] }>(`/api/v1/reports/timeline?${activityQuery}`),
  ]);
  const names = new Map(projectData.projects.map((project) => [project.id, project.name]));
  const activityProjectCount = new Set(timelineData.runs.map((run) => run.project_id)).size;
  const chartValues = Object.entries(summaryData.project_span_seconds)
    .map(([id, seconds]) => ({ id, name: names.get(id) || id.slice(0, 8), seconds }))
    .sort((a, b) => b.seconds - a.seconds);

  return <>
    <PageHeader
      title={selectedRange === "24h" ? "Today's summary" : `Summary · ${activityRangeConfig.title.replace("Activity in the ", "")}`}
      subtitle="Aggregated activity across all sessions"
      period={formatPeriod(activityRange.from, activityRange.to)}
    />
    <ActivityMetrics
      parallelismPeak={summaryData.parallelism_peak}
      periodLabel={selectedRange === "24h" ? "Today" : activityRangeConfig.title.replace("Activity in the ", "").replace("Activity in ", "")}
      projectTime={summaryData.agent_seconds}
      runs={summaryData.run_count}
      tokens={summaryData.input_tokens + summaryData.output_tokens}
      wallClock={summaryData.wall_clock_seconds}
    />
    <ActivityTimelinePanel
      from={timelineData.from}
      key={selectedRange}
      meta={`${activityProjectCount} projects`}
      note="Projects with recorded work, ordered by recent activity"
      projects={projectData.projects}
      range={selectedRange}
      runs={timelineData.runs}
      title={activityRangeConfig.title}
      to={timelineData.to}
    />
    <ProjectTimeSection periodLabel={selectedRange === "24h" ? "today" : activityRangeConfig.title.replace("Activity in the ", "")} values={chartValues}/>
  </>;
}

function formatPeriod(from: string, to: string) {
  const formatter = new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", timeZone: "Europe/Madrid" });
  return `${formatter.format(new Date(from))} – ${formatter.format(new Date(new Date(to).getTime() - 1))}`;
}

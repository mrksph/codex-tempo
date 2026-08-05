import type { ActivityRange } from "@/components/activity-range-select";
import { ActivityTimelinePanel } from "@/components/activity-timeline-panel";
import { AutoRefresh } from "@/components/auto-refresh";
import { Metric } from "@/components/metric";
import { PageHeader } from "@/components/page-header";
import { ProjectChart } from "@/components/project-chart";
import { tempoFetch, calendarRecentRange, recentRange, todayRange } from "@/lib/api/client";
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
  const range = todayRange(); const query = new URLSearchParams(range).toString();
  const activityRange = selectedRange === "24h"
    ? recentRange(1)
    : calendarRecentRange(activityRangeConfig.days);
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
      title="Today's summary"
      subtitle="Aggregated activity across all sessions"
      period={new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", year: "numeric" }).format(new Date())}
    />
    <section className="mb-5 grid min-w-0 grid-cols-[repeat(5,minmax(0,1fr))] border border-[var(--line)] bg-[var(--surface)]" aria-label="Primary metrics">
      <Metric label="Agent time" value={formatDuration(summaryData.agent_seconds)} meta="Overlap counts too"/>
      <Metric label="Wall-clock time" value={formatDuration(summaryData.wall_clock_seconds)} meta="Union of intervals"/>
      <Metric label="Peak parallelism" value={String(summaryData.parallelism_peak)} meta="Today's peak, not real time"/>
      <Metric label="Runs" value={String(summaryData.run_count)} meta="Runs started today"/>
      <Metric label="Tokens" value={compact(summaryData.input_tokens + summaryData.output_tokens)} meta={`${compact(summaryData.output_tokens)} output`}/>
    </section>
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
    <section className="border border-[var(--line)] bg-[var(--surface)]">
      <AutoRefresh />
      <div className="flex min-h-[52px] items-center justify-between gap-3 border-b border-[var(--line)] px-4 py-3.5">
        <div>
          <h2 className="text-sm font-semibold tracking-normal">Time by project</h2>
          <p className="mt-1 text-xs text-[var(--muted)]">Accumulated agent time today</p>
        </div>
        <span className="text-xs text-[var(--muted)]">{chartValues.length} projects</span>
      </div>
      <ProjectChart values={chartValues} />
    </section>
  </>;
}

function formatDuration(seconds: number) {
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} min`;
  return `${Math.floor(minutes / 60)} h ${String(minutes % 60).padStart(2, "0")} min`;
}

function compact(value: number) {
  return new Intl.NumberFormat("en-GB", { notation: "compact", maximumFractionDigits: 1 }).format(value);
}

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
  "24h": { days: 1, title: "Actividad de las últimas 24 horas" },
  "7d": { days: 7, title: "Actividad de los últimos 7 días" },
  "30d": { days: 30, title: "Actividad de los últimos 30 días" },
  "90d": { days: 90, title: "Actividad de los últimos 90 días" },
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
  const chartValues = Object.entries(summaryData.project_span_seconds).map(([id, seconds]) => ({ id, name: names.get(id) || id.slice(0, 8), seconds })).sort((a,b) => b.seconds-a.seconds);
  return <>
    <PageHeader title="Resumen de hoy" subtitle="Actividad consolidada en todas las sesiones" period={new Intl.DateTimeFormat("es", {day:"2-digit",month:"short",year:"numeric"}).format(new Date())}/>
    <section className="metrics" aria-label="Métricas principales">
      <Metric label="Tiempo de agente" value={formatDuration(summaryData.agent_seconds)} meta="Los solapamientos suman"/>
      <Metric label="Tiempo real" value={formatDuration(summaryData.wall_clock_seconds)} meta="Unión de intervalos"/>
      <Metric label="Máx. simultáneos" value={String(summaryData.parallelism_peak)} meta="Pico de hoy, no tiempo real"/>
      <Metric label="Intervalos" value={String(summaryData.run_count)} meta="Tramos iniciados hoy"/>
      <Metric label="Tokens" value={compact(summaryData.input_tokens + summaryData.output_tokens)} meta={`${compact(summaryData.output_tokens)} de salida`}/>
    </section>
    <ActivityTimelinePanel
      from={timelineData.from}
      key={selectedRange}
      meta={`${activityProjectCount} proyectos`}
      note="Proyectos con trabajo registrado, ordenados por actividad reciente"
      projects={projectData.projects}
      range={selectedRange}
      runs={timelineData.runs}
      title={activityRangeConfig.title}
      to={timelineData.to}
    />
    <section className="panel"><AutoRefresh/><div className="panel-head"><div><h2 className="panel-title">Tiempo por proyecto</h2><p className="panel-note">Tiempo de agente acumulado hoy</p></div><span className="panel-subtitle">{chartValues.length} proyectos</span></div><ProjectChart values={chartValues}/></section>
  </>;
}

function formatDuration(seconds:number){const minutes=Math.round(seconds/60);if(minutes<60)return `${minutes} min`;return `${Math.floor(minutes/60)} h ${String(minutes%60).padStart(2,"0")} min`}
function compact(value:number){return new Intl.NumberFormat("es",{notation:"compact",maximumFractionDigits:1}).format(value)}

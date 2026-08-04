import { ChevronLeft } from "lucide-react";
import Link from "next/link";
import { notFound } from "next/navigation";
import {
  ActivityRangeSelect,
  CustomActivityRange,
  type ActivityRangeSelection,
} from "@/components/activity-range-select";
import { ActivityTimelinePanel } from "@/components/activity-timeline-panel";
import { Metric } from "@/components/metric";
import { PageHeader } from "@/components/page-header";
import { tempoFetch, calendarRecentRange, recentRange } from "@/lib/api/client";
import type { Project, Run, Summary } from "@/lib/api/types";

export const dynamic = "force-dynamic";

const PROJECT_RANGES: Record<Exclude<ActivityRangeSelection, "custom">, { days: number; activityTitle: string; label: string }> = {
  "24h": { days: 1, activityTitle: "Actividad de las últimas 24 horas", label: "Últimas 24 horas" },
  "7d": { days: 7, activityTitle: "Actividad de los últimos 7 días", label: "Últimos 7 días" },
  "30d": { days: 30, activityTitle: "Actividad de los últimos 30 días", label: "Últimos 30 días" },
  "90d": { days: 90, activityTitle: "Actividad de los últimos 90 días", label: "Últimos 90 días" },
};

export default async function ProjectDetailPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{
    range?: string | string[];
    from?: string | string[];
    to?: string | string[];
    statsRange?: string | string[];
    statsFrom?: string | string[];
    statsTo?: string | string[];
  }>;
}) {
  const { id } = await params;
  const queryParams = await searchParams;
  const { projects } = await tempoFetch<{ projects: Project[] }>("/api/v1/projects");
  const project = projects.find((value) => value.id === id);
  if (!project) notFound();

  const selectedRange = selectedProjectRange(firstValue(queryParams.range), "7d");
  const selectedStatsRange = selectedProjectRange(firstValue(queryParams.statsRange), "30d");
  const timelineRange = resolveDateRange(selectedRange, firstValue(queryParams.from), firstValue(queryParams.to), 7);
  const statsRange = resolveDateRange(selectedStatsRange, firstValue(queryParams.statsFrom), firstValue(queryParams.statsTo), 30);
  const recentActivityRange = recentRange(30);
  const statsQuery = new URLSearchParams({ from: statsRange.from, to: statsRange.to, project_id: id }).toString();
  const timelineQuery = new URLSearchParams({ from: timelineRange.from, to: timelineRange.to, project_id: id }).toString();
  const recentQuery = new URLSearchParams({ ...recentActivityRange, project_id: id }).toString();
  const [summary, timeline, recentTimeline] = await Promise.all([
    tempoFetch<Summary>(`/api/v1/reports/summary?${statsQuery}`),
    tempoFetch<{ from: string; to: string; runs: Run[] }>(`/api/v1/reports/timeline?${timelineQuery}`),
    tempoFetch<{ runs: Run[] }>(`/api/v1/reports/timeline?${recentQuery}`),
  ]);
  const recentRuns = recentTimeline.runs.slice(0, 20);
  const tokens = summary.input_tokens + summary.output_tokens;
  const statsPeriodLabel = selectedStatsRange === "custom"
    ? statsRange.inputFrom === statsRange.inputTo
      ? statsRange.displayFrom
      : `${statsRange.displayFrom} - ${statsRange.displayTo}`
    : PROJECT_RANGES[selectedStatsRange].label;
  const timelineTitle = selectedRange === "custom"
    ? `Actividad del ${timelineRange.displayFrom} al ${timelineRange.displayTo}`
    : PROJECT_RANGES[selectedRange].activityTitle;

  return <>
    <Link className="back-link" href="/projects"><ChevronLeft size={15}/>Proyectos</Link>
    <PageHeader
      title={project.name}
      subtitle="Detalle de actividad del proyecto"
      periodControl={<ActivityRangeSelect
        allowCustom
        ariaLabel="Rango de métricas"
        customFrom={statsRange.inputFrom}
        customTo={statsRange.inputTo}
        defaultValue="30d"
        fromParam="statsFrom"
        rangeParam="statsRange"
        toParam="statsTo"
        value={selectedStatsRange}
      />}
    />
    {selectedStatsRange === "custom" && <CustomActivityRange
      className="stats-custom-range"
      from={statsRange.inputFrom}
      fromParam="statsFrom"
      key={`${statsRange.inputFrom}-${statsRange.inputTo}`}
      rangeParam="statsRange"
      to={statsRange.inputTo}
      toParam="statsTo"
    />}
    <section className="metrics" aria-label="Métricas del proyecto">
      <Metric label="Total histórico" value={formatDuration(project.agent_seconds)} meta={`${project.run_count} intervalos registrados`}/>
      <Metric label="Tiempo de agente" value={formatDuration(summary.agent_seconds)} meta={statsPeriodLabel}/>
      <Metric label="Tiempo real" value={formatDuration(summary.wall_clock_seconds)} meta="Sin solapamientos"/>
      <Metric label="Intervalos" value={String(summary.run_count)} meta={statsPeriodLabel}/>
      <Metric label="Tokens" value={compact(tokens)} meta={`${compact(summary.output_tokens)} de salida`}/>
    </section>
    <ActivityTimelinePanel
      allowCustomRange
      customFrom={timelineRange.inputFrom}
      customTo={timelineRange.inputTo}
      defaultRange="7d"
      from={timeline.from}
      key={`${selectedRange}-${timelineRange.inputFrom}-${timelineRange.inputTo}`}
      meta={`${timeline.runs.length} intervalos`}
      note={`Intervalos registrados para ${project.name}`}
      projects={[project]}
      range={selectedRange}
      runs={timeline.runs}
      title={timelineTitle}
      to={timeline.to}
    />
    <section className="panel">
      <div className="panel-head"><h2 className="panel-title">Actividad reciente</h2><span className="panel-subtitle">Últimos 20 intervalos</span></div>
      {recentRuns.length ? <div className="table-scroll"><table>
        <thead><tr><th>Inicio</th><th>Duración</th><th>Origen</th><th>Modelo</th><th>Estado</th></tr></thead>
        <tbody>{recentRuns.map((run) => <tr key={run.id}>
          <td>{formatDate(run.started_at)}</td>
          <td className="numeric">{formatRunDuration(run)}</td>
          <td><span className={`badge source-${source(run.session_id)}`}>{sourceLabel(run.session_id)}</span></td>
          <td>{run.model || "-"}</td>
          <td><span className="badge">{statusLabel(run.status)}</span></td>
        </tr>)}</tbody>
      </table></div> : <div className="empty activity-empty">No hay actividad reciente para este proyecto.</div>}
    </section>
  </>;
}

function selectedProjectRange(value: string | undefined, fallback: Exclude<ActivityRangeSelection, "custom">): ActivityRangeSelection {
  return value && (value === "custom" || value in PROJECT_RANGES) ? value as ActivityRangeSelection : fallback;
}

function resolveDateRange(selection: ActivityRangeSelection, fromValue: string | undefined, toValue: string | undefined, defaultDays: number) {
  if (selection !== "custom") {
    const config = PROJECT_RANGES[selection];
    const range = selection === "24h" ? recentRange(1) : calendarRecentRange(config.days);
    const rangeFrom = new Date(range.from);
    const rangeTo = new Date(range.to);
    return {
      ...range,
      displayFrom: formatRangeDate(rangeFrom),
      displayTo: formatRangeDate(rangeTo),
      inputFrom: localDateValue(rangeFrom),
      inputTo: localDateValue(rangeTo),
    };
  }

  const now = new Date();
  const today = startOfLocalDay(now);
  const defaultFrom = new Date(today);
  defaultFrom.setDate(defaultFrom.getDate() - Math.max(0, defaultDays - 1));
  let customFrom = parseLocalDate(fromValue) || defaultFrom;
  let customTo = parseLocalDate(toValue) || today;
  if (customTo > today) customTo = today;
  if (customFrom > customTo) {
    customFrom = defaultFrom;
    customTo = today;
  }
  const endOfFinalDay = new Date(customTo);
  endOfFinalDay.setDate(endOfFinalDay.getDate() + 1);
  endOfFinalDay.setMilliseconds(-1);
  const finish = new Date(Math.min(endOfFinalDay.getTime(), now.getTime()));
  return {
    from: customFrom.toISOString(),
    to: finish.toISOString(),
    displayFrom: formatRangeDate(customFrom),
    displayTo: formatRangeDate(customTo),
    inputFrom: localDateValue(customFrom),
    inputTo: localDateValue(customTo),
  };
}

function formatRangeDate(date: Date) {
  return new Intl.DateTimeFormat("es", { day: "2-digit", month: "short", year: "numeric" }).format(date);
}

function firstValue(value?: string | string[]) {
  return Array.isArray(value) ? value[0] : value;
}

function parseLocalDate(value?: string) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value || "");
  if (!match) return null;
  const date = new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]));
  if (date.getFullYear() !== Number(match[1]) || date.getMonth() !== Number(match[2]) - 1 || date.getDate() !== Number(match[3])) return null;
  return startOfLocalDay(date);
}

function startOfLocalDay(value: Date) {
  const date = new Date(value);
  date.setHours(0, 0, 0, 0);
  return date;
}

function localDateValue(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function formatDuration(seconds: number) {
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} min`;
  return `${Math.floor(minutes / 60)} h ${String(minutes % 60).padStart(2, "0")} min`;
}

function formatRunDuration(run: Run) {
  const end = new Date(run.ended_at || run.last_activity_at).getTime();
  const seconds = Math.max(0, (end - new Date(run.started_at).getTime()) / 1000);
  return seconds < 60 ? "< 1 min" : formatDuration(seconds);
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("es", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(value));
}

function compact(value: number) {
  return new Intl.NumberFormat("es", { notation: "compact", maximumFractionDigits: 1 }).format(value);
}

function source(sessionID: string) {
  return sessionID.startsWith("wakapi:") ? "wakapi" : sessionID.startsWith("hook:") ? "hook" : "codex";
}

function sourceLabel(sessionID: string) {
  const value = source(sessionID);
  return value === "wakapi" ? "Wakapi" : value === "hook" ? "Hooks" : "Codex";
}

function statusLabel(value: string) {
  const labels: Record<string, string> = { completed: "Completado", abandoned: "Cerrado por inactividad", superseded: "Sustituido", running: "En curso" };
  return labels[value] || value;
}

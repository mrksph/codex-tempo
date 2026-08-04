import { ChevronLeft } from "lucide-react";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Metric } from "@/components/metric";
import { PageHeader } from "@/components/page-header";
import { Timeline } from "@/components/timeline";
import { tempoFetch, recentRange } from "@/lib/api/client";
import type { Project, Run, Summary } from "@/lib/api/types";

export const dynamic = "force-dynamic";

export default async function ProjectDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const { projects } = await tempoFetch<{ projects: Project[] }>("/api/v1/projects");
  const project = projects.find((value) => value.id === id);
  if (!project) notFound();

  const monthRange = recentRange(30);
  const weekRange = recentRange(7);
  const monthQuery = new URLSearchParams({ ...monthRange, project_id: id }).toString();
  const weekQuery = new URLSearchParams({ ...weekRange, project_id: id }).toString();
  const [summary, timeline] = await Promise.all([
    tempoFetch<Summary>(`/api/v1/reports/summary?${monthQuery}`),
    tempoFetch<{ from: string; to: string; runs: Run[] }>(`/api/v1/reports/timeline?${weekQuery}`),
  ]);
  const recentRuns = timeline.runs.slice(0, 20);
  const tokens = summary.input_tokens + summary.output_tokens;

  return <>
    <Link className="back-link" href="/projects"><ChevronLeft size={15}/>Proyectos</Link>
    <PageHeader title={project.name} subtitle="Detalle de actividad del proyecto" period="Últimos 30 días"/>
    <section className="metrics" aria-label="Métricas del proyecto">
      <Metric label="Total histórico" value={formatDuration(project.agent_seconds)} meta={`${project.run_count} intervalos registrados`}/>
      <Metric label="Tiempo de agente" value={formatDuration(summary.agent_seconds)} meta="Últimos 30 días"/>
      <Metric label="Tiempo real" value={formatDuration(summary.wall_clock_seconds)} meta="Sin solapamientos"/>
      <Metric label="Intervalos" value={String(summary.run_count)} meta="Últimos 30 días"/>
      <Metric label="Tokens" value={compact(tokens)} meta={`${compact(summary.output_tokens)} de salida`}/>
    </section>
    <section className="panel activity-panel">
      <div className="panel-head">
        <div><h2 className="panel-title">Actividad de los últimos 7 días</h2><p className="panel-note">Intervalos registrados para {project.name}</p></div>
        <span className="panel-subtitle">{timeline.runs.length} intervalos</span>
      </div>
      <Timeline runs={timeline.runs} projects={[project]} from={timeline.from} to={timeline.to}/>
    </section>
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

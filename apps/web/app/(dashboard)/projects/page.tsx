import { ArrowUpRight, FolderKanban } from "lucide-react";
import Link from "next/link";
import { PageHeader } from "@/components/page-header";
import { tempoFetch, recentRange } from "@/lib/api/client";
import type { Project, Summary } from "@/lib/api/types";

export const dynamic = "force-dynamic";

export default async function ProjectsPage() {
  const range = recentRange(30);
  const query = new URLSearchParams(range).toString();
  const [{ projects }, summary] = await Promise.all([
    tempoFetch<{ projects: Project[] }>("/api/v1/projects"),
    tempoFetch<Summary>(`/api/v1/reports/summary?${query}`),
  ]);

  return <>
    <PageHeader title="Projects" subtitle="Recent and historical activity by repository" />
    <div className="table-wrap">{projects.length ? <table>
      <thead><tr><th>Project</th><th>Last 30 days</th><th>Total tracked</th><th>Runs</th><th>Last activity</th></tr></thead>
      <tbody>{projects.map((project) => <tr key={project.id}>
        <td><Link className="project-link" href={`/projects/${project.id}`}><FolderKanban size={15}/><strong>{project.name}</strong><ArrowUpRight size={13}/></Link></td>
        <td className="numeric"><strong>{recentDuration(summary.project_span_seconds[project.id] || 0)}</strong></td>
        <td className="numeric">{duration(project.agent_seconds)}</td>
        <td className="numeric">{project.run_count}</td>
        <td>{project.last_active_at ? date(project.last_active_at) : "-"}</td>
      </tr>)}</tbody>
    </table> : <div className="empty">No projects have been recorded yet.</div>}</div>
  </>;
}

function duration(seconds: number) {
  const minutes = Math.round(seconds / 60);
  return minutes < 60 ? `${minutes} min` : `${Math.floor(minutes / 60)} h ${String(minutes % 60).padStart(2, "0")} min`;
}

function recentDuration(seconds: number) {
  return seconds > 0 ? duration(seconds) : "-";
}

function date(value: string) {
  return new Intl.DateTimeFormat("en-GB", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(value));
}

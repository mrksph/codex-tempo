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
    <div className="overflow-x-auto border border-[var(--line)] bg-[var(--surface)]">
      {projects.length ? (
        <table className="w-full border-collapse text-xs">
          <thead>
            <tr>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Project</th>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Last 30 days</th>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Total tracked</th>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Runs</th>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Last activity</th>
            </tr>
          </thead>
          <tbody>
            {projects.map((project) => <tr key={project.id}>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px]"><Link className="inline-flex items-center gap-1.5 text-[var(--ink)] hover:text-[var(--accent)]" href={`/projects/${project.id}`}><FolderKanban size={15} /><strong>{project.name}</strong><ArrowUpRight size={13} /></Link></td>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px] [font-variant-numeric:tabular-nums]"><strong>{recentDuration(summary.project_span_seconds[project.id] || 0)}</strong></td>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px] [font-variant-numeric:tabular-nums]">{duration(project.agent_seconds)}</td>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px] [font-variant-numeric:tabular-nums]">{project.run_count}</td>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px]">{project.last_active_at ? date(project.last_active_at) : "-"}</td>
            </tr>)}
          </tbody>
        </table>
      ) : (
        <div className="grid min-h-[220px] place-items-center p-8 text-center text-[13px] text-[var(--muted)]">No projects have been recorded yet.</div>
      )}
    </div>
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

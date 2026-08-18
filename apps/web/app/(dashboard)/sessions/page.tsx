import { ArrowUpRight, TerminalSquare } from "lucide-react";
import Link from "next/link";
import { PageHeader } from "@/components/page-header";
import { TablePagination } from "@/components/table-pagination";
import { tempoFetch } from "@/lib/api/client";
import { paginate } from "@/lib/pagination";
import type { Machine, Project, Session } from "@/lib/api/types";

export const dynamic = "force-dynamic";

export default async function SessionsPage({ searchParams }: { searchParams: Promise<{ page?: string | string[] }> }) {
  const queryParams = await searchParams;
  const [{ sessions }, { projects }, { machines }] = await Promise.all([
    tempoFetch<{ sessions: Session[] }>("/api/v1/sessions"),
    tempoFetch<{ projects: Project[] }>("/api/v1/projects"),
    tempoFetch<{ machines: Machine[] }>("/api/v1/machines"),
  ]);
  const projectNames = new Map(projects.map((project) => [project.id, project.name]));
  const machineNames = new Map(machines.map((machine) => [machine.id, machine.name]));
  const pagination = paginate(sessions, queryParams.page);

  return <>
    <PageHeader title="Sessions" subtitle="Native Codex conversations; imported Wakapi activity does not create sessions here" />
    <div className="border border-[var(--line)] bg-[var(--surface)]" id="sessions-table">
      {sessions.length ? (
        <><div className="overflow-x-auto"><table className="w-full border-collapse text-xs">
          <thead>
            <tr>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Project</th>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Source</th>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Runs</th>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Machine</th>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Start</th>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Last activity</th>
              <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Session</th>
            </tr>
          </thead>
          <tbody>
            {pagination.items.map((session) => <tr key={session.id}>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px]"><Link className="inline-flex items-center gap-1.5 text-[var(--ink)] hover:text-[var(--accent)]" href={`/projects/${session.project_id}`}><strong>{projectNames.get(session.project_id) || short(session.project_id)}</strong><ArrowUpRight size={13} /></Link></td>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px]">{sessionBadge(session.source)}</td>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px] [font-variant-numeric:tabular-nums]">{session.run_count}</td>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px]">{machineNames.get(session.machine_id) || short(session.machine_id)}</td>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px]">{date(session.started_at)}</td>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px]">{date(session.last_activity_at)}</td>
              <td className="border-b border-[var(--line)] px-3.5 py-[13px]"><span className="inline-flex items-center gap-1.5 text-xs text-[var(--muted)] font-mono" title={session.id}><TerminalSquare size={13} />{short(stripPrefix(session.id))}</span></td>
            </tr>)}
          </tbody>
        </table></div><TablePagination anchor="sessions-table" {...pagination}/></>
      ) : (
        <div className="grid min-h-[220px] place-items-center p-8 text-center text-[13px] text-[var(--muted)]">No native sessions have been recorded yet.</div>
      )}
    </div>
  </>;
}

function stripPrefix(value: string) {
  const separator = value.indexOf(":");
  return separator >= 0 ? value.slice(separator + 1) : value;
}

function short(value: string) {
  return value.length > 18 ? `${value.slice(0, 15)}...` : value;
}

function date(value: string) {
  return new Intl.DateTimeFormat("en-GB", { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(value));
}

function sessionBadge(source: string) {
  if (source === "hook") {
    return <span className="inline-flex min-h-[22px] items-center rounded bg-[#dff2e9] px-2 text-[11px] font-semibold text-[var(--accent)]">Hooks</span>;
  }
  if (source === "codex") {
    return <span className="inline-flex min-h-[22px] items-center rounded bg-[#e8eef4] px-2 text-[11px] font-semibold text-[#526f8a]">Codex history</span>;
  }
  return <span className="inline-flex min-h-[22px] items-center rounded bg-[var(--surface-muted)] px-2 text-[11px] font-semibold text-[var(--muted)]">{source}</span>;
}

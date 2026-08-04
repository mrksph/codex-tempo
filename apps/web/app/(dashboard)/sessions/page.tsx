import { ArrowUpRight, TerminalSquare } from "lucide-react";
import Link from "next/link";
import { PageHeader } from "@/components/page-header";
import { tempoFetch } from "@/lib/api/client";
import type { Machine, Project, Session } from "@/lib/api/types";

export const dynamic = "force-dynamic";

export default async function SessionsPage() {
  const [{ sessions }, { projects }, { machines }] = await Promise.all([
    tempoFetch<{ sessions: Session[] }>("/api/v1/sessions"),
    tempoFetch<{ projects: Project[] }>("/api/v1/projects"),
    tempoFetch<{ machines: Machine[] }>("/api/v1/machines"),
  ]);
  const projectNames = new Map(projects.map((project) => [project.id, project.name]));
  const machineNames = new Map(machines.map((machine) => [machine.id, machine.name]));

  return <>
    <PageHeader title="Sessions" subtitle="Native Codex conversations; imported Wakapi activity does not create sessions here" />
    <div className="table-wrap">{sessions.length ? <table>
      <thead><tr><th>Project</th><th>Source</th><th>Runs</th><th>Machine</th><th>Start</th><th>Last activity</th><th>Session</th></tr></thead>
      <tbody>{sessions.map((session) => <tr key={session.id}>
        <td><Link className="project-link" href={`/projects/${session.project_id}`}><strong>{projectNames.get(session.project_id) || short(session.project_id)}</strong><ArrowUpRight size={13}/></Link></td>
        <td><span className={`badge source-${session.source}`}>{sourceLabel(session.source)}</span></td>
        <td className="numeric">{session.run_count}</td>
        <td>{machineNames.get(session.machine_id) || short(session.machine_id)}</td>
        <td>{date(session.started_at)}</td>
        <td>{date(session.last_activity_at)}</td>
        <td><span className="session-id" title={session.id}><TerminalSquare size={13}/>{short(stripPrefix(session.id))}</span></td>
      </tr>)}</tbody>
    </table> : <div className="empty">No native sessions have been recorded yet.</div>}</div>
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

function sourceLabel(value: string) {
  return value === "hook" ? "Hooks" : value === "codex" ? "Codex history" : value;
}

import { PageHeader } from "@/components/page-header";
import { tempoFetch } from "@/lib/api/client";
import type { Machine } from "@/lib/api/types";

export const dynamic = "force-dynamic";

export default async function MachinesPage() {
  const { machines } = await tempoFetch<{ machines: Machine[] }>("/api/v1/machines");
  return (
    <>
      <PageHeader title="Machines" subtitle="Authorized agents and last sync" />
      <div className="overflow-x-auto border border-[var(--line)] bg-[var(--surface)]">
        {machines.length ? (
          <table className="w-full border-collapse text-xs">
            <thead>
              <tr>
                <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Name</th>
                <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">ID</th>
                <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Created</th>
                <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Last seen</th>
                <th className="border-b border-[var(--line)] bg-[#fafbfa] px-3.5 py-[11px] text-left text-[10px] uppercase tracking-wide text-[var(--muted)]">Status</th>
              </tr>
            </thead>
            <tbody>
              {machines.map((machine) => {
                const online = machine.last_seen_at && Date.now() - new Date(machine.last_seen_at).getTime() < 120000;
                return (
                  <tr key={machine.id}>
                    <td className="border-b border-[var(--line)] px-3.5 py-[13px]"><strong>{machine.name}</strong></td>
                    <td className="border-b border-[var(--line)] px-3.5 py-[13px]">{machine.id.slice(0, 13)}...</td>
                    <td className="border-b border-[var(--line)] px-3.5 py-[13px]">{date(machine.created_at)}</td>
                    <td className="border-b border-[var(--line)] px-3.5 py-[13px]">{machine.last_seen_at ? date(machine.last_seen_at) : "Never"}</td>
                    <td className="border-b border-[var(--line)] px-3.5 py-[13px]">
                      <span className={online
                        ? "inline-flex min-h-[22px] items-center rounded bg-[#dff2e9] px-2 text-[11px] font-semibold text-[var(--accent)]"
                        : "inline-flex min-h-[22px] items-center rounded bg-[var(--surface-muted)] px-2 text-[11px] font-semibold text-[var(--muted)]"}>
                        {online ? "Online" : "Offline"}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          <div className="grid min-h-[220px] place-items-center p-8 text-center text-[13px] text-[var(--muted)]">No machines have been registered yet.</div>
        )}
      </div>
    </>
  );
}

function date(value: string) {
  return new Intl.DateTimeFormat("en-GB", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

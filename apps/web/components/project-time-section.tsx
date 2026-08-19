import { AutoRefresh } from "@/components/auto-refresh";
import { ProjectChart } from "@/components/project-chart";

export function ProjectTimeSection({
  periodLabel,
  values,
}: {
  periodLabel: string;
  values: { id: string; name: string; seconds: number }[];
}) {
  return <section className="border border-[var(--line)] bg-[var(--surface)]">
    <AutoRefresh />
    <div className="flex min-h-[52px] items-center justify-between gap-3 border-b border-[var(--line)] px-4 py-3.5">
      <div>
        <h2 className="text-sm font-semibold tracking-normal">Time by project</h2>
        <p className="mt-1 text-xs text-[var(--muted)]">Unique project time · {periodLabel}</p>
      </div>
      <span className="text-xs text-[var(--muted)]">{values.length} projects</span>
    </div>
    <ProjectChart values={values}/>
  </section>;
}

import { Metric } from "@/components/metric";

export function ActivityMetrics({
  projectTime,
  wallClock,
  parallelismPeak,
  runs,
  tokens,
  periodLabel,
  scopeLabel = "Overlap counted once per project",
  lifetime,
}: {
  projectTime: number;
  wallClock: number;
  parallelismPeak: number;
  runs: number;
  tokens: number;
  periodLabel: string;
  scopeLabel?: string;
  lifetime?: { value: string; meta: string };
}) {
  return <section className={`mb-5 grid min-w-0 border border-[var(--line)] bg-[var(--surface)] ${lifetime ? "grid-cols-[repeat(6,minmax(0,1fr))]" : "grid-cols-[repeat(5,minmax(0,1fr))]"}`} aria-label="Activity metrics">
    {lifetime && <Metric label="Lifetime total" value={lifetime.value} meta={lifetime.meta}/>}
    <Metric label="Project time" value={formatDuration(projectTime)} meta={scopeLabel}/>
    <Metric label="Wall-clock time" value={formatDuration(wallClock)} meta="Union of intervals"/>
    <Metric label="Peak parallelism" value={String(parallelismPeak)} meta={`${periodLabel} peak`}/>
    <Metric label="Runs" value={String(runs)} meta={`${periodLabel} runs`}/>
    <Metric label="Tokens" value={compact(tokens)} meta="Output included"/>
  </section>;
}

function formatDuration(seconds: number) {
  const minutes = Math.round(seconds / 60);
  return minutes < 60 ? `${minutes} min` : `${Math.floor(minutes / 60)} h ${String(minutes % 60).padStart(2, "0")} min`;
}

function compact(value: number) {
  return new Intl.NumberFormat("en-GB", { notation: "compact", maximumFractionDigits: 1 }).format(value);
}

export function Metric({ label, value, meta }: { label: string; value: string; meta: string }) {
  return <div className="min-w-0 border-r border-[var(--line)] p-[18px]"><div className="text-[11px] font-bold uppercase tracking-normal text-[var(--muted)]">{label}</div><div className="mt-2.5 whitespace-nowrap text-[25px] leading-none font-bold [font-variant-numeric:tabular-nums]">{value}</div><div className="mt-1.5 text-[11px] text-[var(--muted)]">{meta}</div></div>;
}

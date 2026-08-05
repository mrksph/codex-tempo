import { CalendarDays } from "lucide-react";
import type { ReactNode } from "react";

export function PageHeader({
  title,
  subtitle,
  period,
  periodControl,
}: {
  title: string;
  subtitle: string;
  period?: string;
  periodControl?: ReactNode;
}) {
  const periodContent = periodControl || period;
  return <div className="mb-6 flex items-end justify-between gap-5 max-[760px]:flex-col max-[760px]:items-start"><div><h1 className="mb-1.5 text-[24px] leading-tight">{title}</h1><p className="m-0 text-xs text-[var(--muted)]">{subtitle}</p></div>{periodContent && <div className={`flex items-center gap-2 rounded border border-[var(--line)] bg-[var(--surface)] px-2.5 py-1.5 text-xs text-[var(--muted)] whitespace-nowrap ${periodControl ? "gap-0.5 pl-[9px]" : ""}`}><CalendarDays size={14} aria-hidden="true"/>{periodContent}</div>}</div>;
}

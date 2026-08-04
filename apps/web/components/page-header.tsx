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
  return <div className="page-head"><div><h1>{title}</h1><p>{subtitle}</p></div>{periodContent && <div className={`period${periodControl ? " period-control" : ""}`}><CalendarDays size={14} aria-hidden="true"/>{periodContent}</div>}</div>;
}

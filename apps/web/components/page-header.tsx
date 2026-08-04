import { CalendarDays } from "lucide-react";

export function PageHeader({ title, subtitle, period }: { title: string; subtitle: string; period?: string }) {
  return <div className="page-head"><div><h1>{title}</h1><p>{subtitle}</p></div>{period && <div className="period"><CalendarDays size={14}/>{period}</div>}</div>;
}

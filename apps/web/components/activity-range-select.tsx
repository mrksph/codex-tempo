"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useState, useTransition, type FormEvent } from "react";
import { ListFilter } from "lucide-react";

export type ActivityRange = "24h" | "7d" | "30d" | "90d";
export type ActivityRangeSelection = ActivityRange | "custom";

type RangeQueryParams = {
  rangeParam?: string;
  fromParam?: string;
  toParam?: string;
};

export function ActivityRangeSelect({
  value,
  allowCustom = false,
  customFrom,
  customTo,
  defaultValue = "24h",
  rangeParam = "range",
  fromParam = "from",
  toParam = "to",
  ariaLabel = "Activity range",
}: {
  value: ActivityRangeSelection;
  allowCustom?: boolean;
  customFrom?: string;
  customTo?: string;
  defaultValue?: ActivityRange;
  ariaLabel?: string;
} & RangeQueryParams) {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [pending, startTransition] = useTransition();

  function changeRange(nextRange: ActivityRangeSelection) {
    const params = new URLSearchParams(searchParams.toString());
    if (nextRange === defaultValue) params.delete(rangeParam);
    else params.set(rangeParam, nextRange);
    if (nextRange === "custom") {
      if (customFrom) params.set(fromParam, customFrom);
      if (customTo) params.set(toParam, customTo);
    } else {
      params.delete(fromParam);
      params.delete(toParam);
    }
    const query = params.toString();
    startTransition(() => router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false }));
  }

  return <select
    aria-label={ariaLabel}
    className="h-8 cursor-pointer rounded border border-[var(--line)] bg-[var(--surface)] px-3 text-[11px] text-[var(--ink)]"
    disabled={pending}
    onChange={(event) => changeRange(event.target.value as ActivityRangeSelection)}
    value={value}
  >
    <option value="24h">24 hours</option>
    <option value="7d">7 days</option>
    <option value="30d">30 days</option>
    <option value="90d">90 days</option>
    {allowCustom && <option value="custom">Custom</option>}
  </select>;
}

export function CustomActivityRange({
  from,
  to,
  className,
  rangeParam = "range",
  fromParam = "from",
  toParam = "to",
}: {
  from: string;
  to: string;
  className?: string;
} & RangeQueryParams) {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [startDate, setStartDate] = useState(from);
  const [endDate, setEndDate] = useState(to);
  const [pending, startTransition] = useTransition();
  const today = localDateValue(new Date());

  function applyRange(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!startDate || !endDate || startDate > endDate) return;
    const params = new URLSearchParams(searchParams.toString());
    params.set(rangeParam, "custom");
    params.set(fromParam, startDate);
    params.set(toParam, endDate);
    startTransition(() => router.replace(`${pathname}?${params.toString()}`, { scroll: false }));
  }

  return <form className={`min-h-[54px] border-b border-[var(--line)] bg-[#fafbfa] flex items-end justify-end gap-2.5 px-4 py-2.5 ${className || ""}`} onSubmit={applyRange}>
    <label className="grid gap-1 text-xs font-bold text-[var(--muted)]">
      <span>Start date</span>
      <input
        className="h-8 w-[142px] border border-[var(--line)] bg-[var(--surface)] px-2 py-0 text-[var(--ink)]"
        max={endDate || today}
        onChange={(event) => setStartDate(event.target.value)}
        required
        type="date"
        value={startDate}
        style={{ colorScheme: "light" }}
      />
    </label>
    <label className="grid gap-1 text-xs font-bold text-[var(--muted)]">
      <span>End date</span>
      <input
        className="h-8 w-[142px] border border-[var(--line)] bg-[var(--surface)] px-2 py-0 text-[var(--ink)]"
        max={today}
        min={startDate}
        onChange={(event) => setEndDate(event.target.value)}
        required
        type="date"
        value={endDate}
        style={{ colorScheme: "light" }}
      />
    </label>
    <button className={`inline-flex h-8 items-center justify-center gap-1.5 px-2.5 text-xs font-bold ${!startDate || !endDate || startDate > endDate || pending ? "cursor-wait opacity-60" : "cursor-pointer"} border border-[var(--accent)] bg-[var(--accent)] text-white`} disabled={pending || !startDate || !endDate || startDate > endDate} type="submit">
      <ListFilter size={14} aria-hidden="true" />
      <span>Apply</span>
    </button>
  </form>;
}

function localDateValue(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

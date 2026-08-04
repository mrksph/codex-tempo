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
    className="range-select"
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

  return <form className={`activity-custom-range${className ? ` ${className}` : ""}`} onSubmit={applyRange}>
    <label className="activity-date-field">
      <span>Start date</span>
      <input
        className="activity-date-input"
        max={endDate || today}
        onChange={(event) => setStartDate(event.target.value)}
        required
        type="date"
        value={startDate}
      />
    </label>
    <label className="activity-date-field">
      <span>End date</span>
      <input
        className="activity-date-input"
        max={today}
        min={startDate}
        onChange={(event) => setEndDate(event.target.value)}
        required
        type="date"
        value={endDate}
      />
    </label>
    <button className="activity-apply-range" disabled={pending || !startDate || !endDate || startDate > endDate} type="submit">
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

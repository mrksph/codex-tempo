"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useTransition } from "react";

export type ActivityRange = "24h" | "7d" | "30d" | "90d";

export function ActivityRangeSelect({ value }: { value: ActivityRange }) {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [pending, startTransition] = useTransition();

  function changeRange(nextRange: ActivityRange) {
    const params = new URLSearchParams(searchParams.toString());
    if (nextRange === "24h") params.delete("range");
    else params.set("range", nextRange);
    const query = params.toString();
    startTransition(() => router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false }));
  }

  return <select
    aria-label="Rango de actividad"
    className="range-select"
    disabled={pending}
    onChange={(event) => changeRange(event.target.value as ActivityRange)}
    value={value}
  >
    <option value="24h">24 horas</option>
    <option value="7d">7 días</option>
    <option value="30d">30 días</option>
    <option value="90d">90 días</option>
  </select>;
}

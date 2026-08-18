"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";

export function TablePagination({
  anchor,
  page,
  pageParam = "page",
  pageSize,
  totalItems,
  totalPages,
}: {
  anchor: string;
  page: number;
  pageParam?: string;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}) {
  const pathname = usePathname();
  const searchParams = useSearchParams();

  if (totalPages <= 1) return null;

  const first = (page - 1) * pageSize + 1;
  const last = Math.min(page * pageSize, totalItems);
  const href = (target: number) => {
    const params = new URLSearchParams(searchParams.toString());
    if (target === 1) params.delete(pageParam);
    else params.set(pageParam, String(target));
    const query = params.toString();
    return `${pathname}${query ? `?${query}` : ""}#${anchor}`;
  };

  return <nav className="flex min-h-[52px] items-center justify-between gap-3 border-t border-[var(--line)] px-3.5 py-2.5 text-xs text-[var(--muted)]" aria-label="Table pagination">
    <span>{first}-{last} of {totalItems}</span>
    <div className="flex items-center gap-2">
      {page > 1
        ? <Link className="inline-flex h-8 items-center gap-1 rounded border border-[var(--line)] bg-white px-2.5 font-semibold text-[var(--ink)] hover:bg-[var(--surface-muted)]" href={href(page - 1)}><ChevronLeft size={14}/>Previous</Link>
        : <span className="inline-flex h-8 items-center gap-1 rounded border border-[var(--line)] px-2.5 font-semibold opacity-40"><ChevronLeft size={14}/>Previous</span>}
      <span className="min-w-[72px] text-center [font-variant-numeric:tabular-nums]">Page {page} of {totalPages}</span>
      {page < totalPages
        ? <Link className="inline-flex h-8 items-center gap-1 rounded border border-[var(--line)] bg-white px-2.5 font-semibold text-[var(--ink)] hover:bg-[var(--surface-muted)]" href={href(page + 1)}>Next<ChevronRight size={14}/></Link>
        : <span className="inline-flex h-8 items-center gap-1 rounded border border-[var(--line)] px-2.5 font-semibold opacity-40">Next<ChevronRight size={14}/></span>}
    </div>
  </nav>;
}

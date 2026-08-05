"use client";

import { useState } from "react";
import { ZoomOut } from "lucide-react";
import {
  ActivityRangeSelect,
  CustomActivityRange,
  type ActivityRange,
  type ActivityRangeSelection,
} from "@/components/activity-range-select";
import { Timeline, formatCompactDateTime, type TimelineZoom } from "@/components/timeline";
import type { Project, Run } from "@/lib/api/types";

type ActivityTimelinePanelProps = {
  from: string;
  allowCustomRange?: boolean;
  customFrom?: string;
  customTo?: string;
  defaultRange?: ActivityRange;
  meta: string;
  note: string;
  projects: Project[];
  range?: ActivityRangeSelection;
  runs: Run[];
  title: string;
  to: string;
};

export function ActivityTimelinePanel({
  allowCustomRange = false,
  customFrom,
  customTo,
  defaultRange = "24h",
  from,
  meta,
  note,
  projects,
  range,
  runs,
  title,
  to,
}: ActivityTimelinePanelProps) {
  const [zoom, setZoom] = useState<TimelineZoom | null>(null);
  const isZoomed = Boolean(zoom);
  const zoomLabel = zoom ? `${formatCompactDateTime(zoom.start)} → ${formatCompactDateTime(zoom.end)}` : "";

  return <section className="mb-5 border border-[var(--line)] bg-[var(--surface)]" id="activity">
    <div className="flex min-h-[52px] items-center justify-between gap-3 border-b border-[var(--line)] px-4 py-[14px]">
      <div><h2 className="text-sm font-bold tracking-normal">{title}</h2><p className="mt-1 text-xs text-[var(--muted)]">{note}</p></div>
      <div className="flex items-center gap-2.5 max-[760px]:flex-col-reverse max-[760px]:items-end">
        <span className="text-xs text-[var(--muted)]">{meta}</span>
        <div className="h-8 w-[112px] flex-shrink-0 max-[760px]:w-full">
          <button
            aria-hidden={!isZoomed}
            className={`h-8 w-full inline-flex items-center justify-start gap-[7px] overflow-hidden whitespace-nowrap rounded border border-[var(--line)] px-2.5 text-[10px] font-bold ${!isZoomed ? "pointer-events-none invisible" : "bg-[var(--surface)] text-[var(--ink)] hover:bg-[var(--surface-muted)]"}`}
            disabled={!isZoomed}
            aria-label={isZoomed ? `Quitar zoom del rango ${zoomLabel}` : "Quitar zoom"}
            title={isZoomed ? `Quitar zoom: ${zoomLabel}` : "Quitar zoom"}
            onClick={() => setZoom(null)}
            tabIndex={isZoomed ? 0 : -1}
            type="button"
          >
            <ZoomOut size={14} aria-hidden="true" />
            <span className="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap [font-variant-numeric:tabular-nums]">{zoomLabel || "Quitar zoom"}</span>
          </button>
        </div>
        {range && <ActivityRangeSelect
          allowCustom={allowCustomRange}
          customFrom={customFrom}
          customTo={customTo}
          defaultValue={defaultRange}
          value={range}
        />}
      </div>
    </div>
    {allowCustomRange && range === "custom" && customFrom && customTo && <CustomActivityRange from={customFrom} to={customTo} />}
    <Timeline
      alignStartToDay={range !== "24h"}
      from={from}
      onZoomChange={setZoom}
      projects={projects}
      runs={runs}
      to={to}
      zoom={zoom}
    />
  </section>;
}

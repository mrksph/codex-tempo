"use client";

import { useState } from "react";
import { ZoomOut } from "lucide-react";
import {
  ActivityRangeSelect,
  CustomActivityRange,
  type ActivityRange,
  type ActivityRangeSelection,
} from "@/components/activity-range-select";
import { Timeline, type TimelineZoom } from "@/components/timeline";
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

  return <section className="panel activity-panel" id="activity">
    <div className="panel-head">
      <div><h2 className="panel-title">{title}</h2><p className="panel-note">{note}</p></div>
      <div className="panel-actions">
        <span className="panel-subtitle">{meta}</span>
        <div className="activity-zoom-slot">
          <button
            aria-hidden={!isZoomed}
            className={`activity-clear-zoom ${isZoomed ? "" : "is-hidden"}`}
            disabled={!isZoomed}
            onClick={() => setZoom(null)}
            tabIndex={isZoomed ? 0 : -1}
            type="button"
          >
            <ZoomOut size={14} aria-hidden="true" />
            <span>Quitar zoom</span>
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

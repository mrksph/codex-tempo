"use client";

import { useEffect, useMemo, useRef, useState, type PointerEvent } from "react";
import Link from "next/link";
import type { Project, Run } from "@/lib/api/types";

const PROJECT_COLORS = ["#167c5a", "#147c8d", "#b76b12", "#7857a4", "#b64c5b", "#526f8a", "#9a5f36", "#4f766b"];
const DISPLAY_TIME_ZONE = "Europe/Madrid";

type Interval = {
  start: number;
  end: number;
};

export type TimelineZoom = Interval;

type ProjectActivity = {
  id: string;
  name: string;
  color: string;
  intervals: Interval[];
  total: number;
  lastActivity: number;
};

export function Timeline({ runs, projects, from, to, zoom, onZoomChange, alignStartToDay }: {
  runs: Run[];
  projects: Project[];
  from: string;
  to: string;
  zoom: TimelineZoom | null;
  onZoomChange: (zoom: TimelineZoom | null) => void;
  alignStartToDay: boolean;
}) {
  const viewport = useRef<HTMLDivElement>(null);
  const plot = useRef<HTMLDivElement>(null);
  const hoverGuide = useRef<HTMLDivElement>(null);
  const hoverLabel = useRef<HTMLSpanElement>(null);
  const suppressClick = useRef(false);
  const baseStart = new Date(from).getTime();
  const baseFinish = new Date(to).getTime();
  const displayStart = alignStartToDay ? startOfLocalDay(baseStart) : baseStart;
  const [selection, setSelection] = useState<{ anchor: number; focus: number } | null>(null);
  const start = zoom?.start ?? displayStart;
  const finish = zoom?.end ?? baseFinish;
  const span = Math.max(1, finish - start);
  const activity = useMemo(() => groupByProject(runs, projects, start, finish), [runs, projects, start, finish]);
  const ticks = useMemo(() => ticksForRange(start, finish), [start, finish]);
  const selectionStart = selection ? Math.min(selection.anchor, selection.focus) : 0;
  const selectionEnd = selection ? Math.max(selection.anchor, selection.focus) : 0;

  useEffect(() => {
    if (!zoom || !viewport.current) return;
    viewport.current.scrollLeft = 0;
    if (hoverGuide.current) {
      hoverGuide.current.style.left = "20px";
      hoverGuide.current.style.opacity = "0";
    }
  }, [zoom]);

  function pointForClientX(clientX: number) {
    const element = plot.current;
    if (!element) return { time: start, left: 20, ratio: 0 };
    const rect = element.getBoundingClientRect();
    const trackStart = rect.left + 20;
    const trackWidth = Math.max(1, rect.width - 40);
    const ratio = Math.min(1, Math.max(0, (clientX - trackStart) / trackWidth));
    return {
      time: start + ratio * span,
      left: 20 + ratio * trackWidth,
      ratio,
    };
  }

  function updateHover(event: PointerEvent<HTMLDivElement>) {
    const point = pointForClientX(event.clientX);
    if (hoverGuide.current) {
      hoverGuide.current.style.left = `${point.left}px`;
      hoverGuide.current.style.opacity = "1";
      hoverGuide.current.classList.toggle("near-start", point.ratio < 0.08);
      hoverGuide.current.classList.toggle("near-end", point.ratio > 0.92);
    }
    if (hoverLabel.current) hoverLabel.current.textContent = formatHoverTime(point.time);
    return point.time;
  }

  function hideHover() {
    if (hoverGuide.current) hoverGuide.current.style.opacity = "0";
  }

  function beginSelection(event: PointerEvent<HTMLDivElement>) {
    if (event.button !== 0) return;
    const time = updateHover(event);
    event.currentTarget.setPointerCapture(event.pointerId);
    setSelection({ anchor: time, focus: time });
  }

  function updateInteraction(event: PointerEvent<HTMLDivElement>) {
    const time = updateHover(event);
    if (!selection) return;
    setSelection({ ...selection, focus: time });
  }

  function finishSelection(event: PointerEvent<HTMLDivElement>) {
    if (!selection) return;
    event.currentTarget.releasePointerCapture(event.pointerId);
    const nextStart = Math.min(selection.anchor, selection.focus);
    const nextEnd = Math.max(selection.anchor, selection.focus);
    setSelection(null);
    if (nextEnd - nextStart < 10 * 60 * 1000) return;
    suppressClick.current = true;
    window.setTimeout(() => { suppressClick.current = false; }, 0);
    onZoomChange({ start: nextStart, end: nextEnd });
  }

  return <div className="activity-timeline" role="region" aria-label="Activity by project in the selected period">
      <div className="activity-labels">
        <div className="activity-axis-spacer" />
        {activity.map((project) => <div className="activity-project" key={project.id}>
          <Link className="activity-project-name" href={`/projects/${project.id}`} title={`View ${project.name}`}>
            <span className="activity-project-swatch" style={{ backgroundColor: project.color }} />
            <span>{project.name}</span>
          </Link>
          <span className="activity-project-duration">{formatDuration(project.total)}</span>
        </div>)}
      </div>
      <div className="activity-plot-viewport" ref={viewport}>
        <div
          className="activity-plot"
          ref={plot}
          onClickCapture={(event) => {
            if (!suppressClick.current) return;
            event.preventDefault();
            event.stopPropagation();
          }}
          onPointerCancel={() => { setSelection(null); hideHover(); }}
          onPointerDown={beginSelection}
          onPointerEnter={updateHover}
          onPointerLeave={hideHover}
          onPointerMove={updateInteraction}
          onPointerUp={finishSelection}
        >
          <div className="activity-axis-track">
            {ticks.map((tick) => <div className="activity-tick" key={tick.value} style={{ left: `${((tick.value - start) / span) * 100}%` }}>
              {tick.showLabel && <span className="activity-tick-label">{formatTick(tick.value)}</span>}
            </div>)}
          </div>
          {activity.length === 0 && <div className="empty activity-empty">No activity has been recorded in this range.</div>}
          {activity.map((project) => <div className="activity-plot-row" key={project.id}>
            <div className="activity-track">
              {ticks.map((tick) => <span className={`activity-gridline ${tick.major ? "major" : ""}`} key={tick.value} style={{ left: `${((tick.value - start) / span) * 100}%` }} />)}
              {project.intervals.map((interval, index) => {
                const left = ((interval.start - start) / span) * 100;
                const width = ((interval.end - interval.start) / span) * 100;
                return <Link
                  className="activity-bar"
                  data-end={interval.end}
                  href={`/projects/${project.id}`}
                  key={`${interval.start}-${index}`}
                  style={{ backgroundColor: project.color, left: `${left}%`, width: `${width}%` }}
                  title={`${project.name} · ${formatInterval(interval)} · ${formatDuration(interval.end - interval.start)}`}
                  aria-label={`View ${project.name}: ${formatInterval(interval)}`}
                />;
              })}
            </div>
          </div>)}
          {selection && Math.abs(selectionEnd - selectionStart) > 0 && <div
            className="activity-selection"
            style={{
              left: selectionPosition(selectionStart, start, span),
              width: selectionWidth(selectionEnd - selectionStart, span),
            }}
          />}
          <div className="activity-hover-guide" ref={hoverGuide} aria-hidden="true">
            <span className="activity-hover-label" ref={hoverLabel} />
          </div>
        </div>
      </div>
    </div>;
}

function selectionPosition(value: number, start: number, span: number) {
  const ratio = (value - start) / span;
  return `calc(20px + ${ratio * 100}% - ${ratio * 40}px)`;
}

function selectionWidth(value: number, span: number) {
  const ratio = value / span;
  return `calc(${ratio * 100}% - ${ratio * 40}px)`;
}

function ticksForRange(start: number, finish: number) {
  const tickMap = new Map<number, { value: number; showLabel: boolean; major: boolean }>();
  tickMap.set(start, { value: start, showLabel: true, major: false });

  const day = new Date(start);
  day.setTime(startOfLocalDay(start));
  if (day.getTime() < start) day.setDate(day.getDate() + 1);
  for (let tick = day.getTime(); tick < finish; tick = nextLocalDay(tick)) {
    tickMap.set(tick, { value: tick, showLabel: true, major: true });
  }

  const days = (finish - start) / (24 * 60 * 60 * 1000);
  const labelInterval = days <= 8 ? 1 : days <= 32 ? 5 : 14;
  const ticks = [...tickMap.values()].sort((a, b) => a.value - b.value);
  ticks.forEach((tick, index) => { tick.showLabel = index % labelInterval === 0; });
  ticks.push({ value: finish, showLabel: true, major: false });
  return ticks;
}

function nextLocalDay(value: number) {
  const day = new Date(value);
  day.setDate(day.getDate() + 1);
  day.setHours(0, 0, 0, 0);
  return day.getTime();
}

function startOfLocalDay(value: number) {
  const day = new Date(value);
  day.setHours(0, 0, 0, 0);
  return day.getTime();
}

function groupByProject(runs: Run[], projects: Project[], rangeStart: number, rangeEnd: number): ProjectActivity[] {
  const names = new Map(projects.map((project) => [project.id, project.name]));
  const grouped = new Map<string, Interval[]>();

  for (const run of runs) {
    const startedAt = new Date(run.started_at).getTime();
    const lastActivityAt = new Date(run.last_activity_at).getTime();
    const endedAt = run.ended_at ? new Date(run.ended_at).getTime() : lastActivityAt;
    const start = Math.max(rangeStart, startedAt);
    const end = Math.min(rangeEnd, endedAt);
    if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) continue;
    grouped.set(run.project_id, [...(grouped.get(run.project_id) || []), { start, end }]);
  }

  const colors = new Map([...grouped.keys()].sort().map((id, index) => [id, PROJECT_COLORS[index % PROJECT_COLORS.length]]));
  return [...grouped.entries()]
    .map(([id, intervals]) => {
      const merged = mergeIntervals(intervals);
      return {
        id,
        name: names.get(id) || id.slice(0, 8),
        color: colors.get(id) || PROJECT_COLORS[0],
        intervals: merged,
        total: merged.reduce((sum, interval) => sum + interval.end - interval.start, 0),
        lastActivity: Math.max(...merged.map((interval) => interval.end)),
      };
    })
    .sort((a, b) => b.lastActivity - a.lastActivity);
}

function mergeIntervals(intervals: Interval[]) {
  const sorted = [...intervals].sort((a, b) => a.start - b.start);
  const merged: Interval[] = [];
  for (const interval of sorted) {
    const previous = merged.at(-1);
    if (previous && interval.start <= previous.end) {
      previous.end = Math.max(previous.end, interval.end);
    } else {
      merged.push({ ...interval });
    }
  }
  return merged;
}

function formatTick(value: number) {
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: DISPLAY_TIME_ZONE,
  }).format(new Date(value));
}

function formatHoverTime(value: number) {
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: DISPLAY_TIME_ZONE,
  }).format(new Date(value));
}

function formatInterval(interval: Interval) {
  const format = new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit", timeZone: DISPLAY_TIME_ZONE });
  return `${format.format(new Date(interval.start))} - ${format.format(new Date(interval.end))}`;
}

function formatDuration(milliseconds: number) {
  const minutes = Math.max(1, Math.round(milliseconds / 60000));
  if (minutes < 60) return `${minutes} min`;
  return `${Math.floor(minutes / 60)} h ${String(minutes % 60).padStart(2, "0")} min`;
}

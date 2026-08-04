"use client";

import { useEffect, useMemo, useRef } from "react";
import Link from "next/link";
import type { Project, Run } from "@/lib/api/types";

const PROJECT_COLORS = ["#167c5a", "#147c8d", "#b76b12", "#7857a4", "#b64c5b", "#526f8a", "#9a5f36", "#4f766b"];
const DISPLAY_TIME_ZONE = "Europe/Madrid";

type Interval = {
  start: number;
  end: number;
};

type ProjectActivity = {
  id: string;
  name: string;
  color: string;
  intervals: Interval[];
  total: number;
  lastActivity: number;
};

export function Timeline({ runs, projects, from, to }: { runs: Run[]; projects: Project[]; from: string; to: string }) {
  const viewport = useRef<HTMLDivElement>(null);
  const start = new Date(from).getTime();
  const finish = new Date(to).getTime();
  const span = Math.max(1, finish - start);
  const activity = useMemo(() => groupByProject(runs, projects, start, finish), [runs, projects, start, finish]);
  const axisSteps = axisStepsForSpan(span);
  const ticks = Array.from({ length: axisSteps + 1 }, (_, index) => start + (span * index) / axisSteps);
  const plotWidth = plotWidthForSpan(span);

  useEffect(() => {
    const element = viewport.current;
    if (!element) return;

    const latestBar = Array.from(element.querySelectorAll<HTMLElement>(".activity-bar"))
      .reduce<HTMLElement | null>((latest, bar) => {
        if (!latest) return bar;
        return Number(bar.dataset.end) > Number(latest.dataset.end) ? bar : latest;
      }, null);
    if (!latestBar) return;

    const viewportRect = element.getBoundingClientRect();
    const barRect = latestBar.getBoundingClientRect();
    const targetX = viewportRect.left + element.clientWidth * 0.8;
    element.scrollLeft = Math.max(0, element.scrollLeft + barRect.right - targetX);
  }, [activity]);

  if (!activity.length) {
    return <div className="empty activity-empty">No hay actividad registrada en este periodo.</div>;
  }

  return <div className="activity-timeline" role="region" aria-label="Actividad por proyecto en el periodo seleccionado">
    <div className="activity-labels">
      <div className="activity-axis-spacer" />
      {activity.map((project) => <div className="activity-project" key={project.id}>
        <Link className="activity-project-name" href={`/projects/${project.id}`} title={`Ver ${project.name}`}>
          <span className="activity-project-swatch" style={{ backgroundColor: project.color }} />
          <span>{project.name}</span>
        </Link>
        <span className="activity-project-duration">{formatDuration(project.total)}</span>
      </div>)}
    </div>
    <div className="activity-plot-viewport" ref={viewport}>
      <div className="activity-plot" style={{ minWidth: plotWidth }}>
        <div className="activity-axis-track">
          {ticks.map((tick, index) => <div className="activity-tick" key={tick} style={{ left: `${(index / axisSteps) * 100}%` }}>
            <span className="activity-tick-label">{formatTick(tick, index === 0 || index === axisSteps || span > 48 * 60 * 60 * 1000)}</span>
          </div>)}
        </div>
        {activity.map((project) => <div className="activity-plot-row" key={project.id}>
          <div className="activity-track">
            {ticks.map((tick, index) => <span className="activity-gridline" key={tick} style={{ left: `${(index / axisSteps) * 100}%` }} />)}
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
                aria-label={`Ver ${project.name}: ${formatInterval(interval)}`}
              />;
            })}
          </div>
        </div>)}
      </div>
    </div>
  </div>;
}

function axisStepsForSpan(span: number) {
  const days = span / (24 * 60 * 60 * 1000);
  if (days <= 1.5) return 6;
  if (days <= 8) return 7;
  if (days <= 32) return 10;
  return 9;
}

function plotWidthForSpan(span: number) {
  const days = span / (24 * 60 * 60 * 1000);
  if (days <= 1.5) return 1410;
  if (days <= 8) return 2520;
  if (days <= 32) return 3600;
  return 4500;
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

function formatTick(value: number, includeDay: boolean) {
  return new Intl.DateTimeFormat("es", includeDay
    ? { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit", timeZone: DISPLAY_TIME_ZONE }
    : { hour: "2-digit", minute: "2-digit", timeZone: DISPLAY_TIME_ZONE },
  ).format(new Date(value));
}

function formatInterval(interval: Interval) {
  const format = new Intl.DateTimeFormat("es", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit", timeZone: DISPLAY_TIME_ZONE });
  return `${format.format(new Date(interval.start))} - ${format.format(new Date(interval.end))}`;
}

function formatDuration(milliseconds: number) {
  const minutes = Math.max(1, Math.round(milliseconds / 60000));
  if (minutes < 60) return `${minutes} min`;
  return `${Math.floor(minutes / 60)} h ${String(minutes % 60).padStart(2, "0")} min`;
}

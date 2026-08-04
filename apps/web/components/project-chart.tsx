"use client";

import * as echarts from "echarts";
import type { ECElementEvent } from "echarts";
import { useRouter } from "next/navigation";
import { useEffect, useRef } from "react";

export function ProjectChart({ values }: { values: { id: string; name: string; seconds: number }[] }) {
  const ref = useRef<HTMLDivElement>(null);
  const chartRef = useRef<echarts.ECharts | null>(null);
  const valuesRef = useRef(values);
  const hasRendered = useRef(false);
  const router = useRouter();

  valuesRef.current = values;

  useEffect(() => {
    if (!ref.current) return;
    const chart = echarts.init(ref.current);
    chartRef.current = chart;
    const openProject = (event: ECElementEvent) => {
      const currentValues = valuesRef.current;
      const index = event.componentType === "series"
        ? event.dataIndex
        : currentValues.findIndex((item) => item.name === String(event.value));
      if (typeof index === "number" && index >= 0 && currentValues[index]) router.push(`/projects/${currentValues[index].id}`);
    };
    chart.on("click", openProject);
    const resize = () => chart.resize(); window.addEventListener("resize", resize);
    return () => {
      window.removeEventListener("resize", resize);
      chart.off("click", openProject);
      chart.dispose();
      chartRef.current = null;
      hasRendered.current = false;
    };
  }, [router]);

  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;
    const initialRender = !hasRendered.current;
    chart.setOption({
      animationDuration: initialRender ? 300 : 0,
      animationDurationUpdate: initialRender ? 0 : 140,
      animationEasingUpdate: "linear",
      grid: { left: 18, right: 28, top: 24, bottom: 42, containLabel: true },
      tooltip: { trigger: "axis", valueFormatter: (value: unknown) => formatDuration(Number(value)) },
      xAxis: { type: "value", axisLabel: { color: "#66716d", formatter: (value: number) => value < 3600 ? `${Math.round(value / 60)}m` : `${Math.round(value / 3600)}h` }, splitLine: { lineStyle: { color: "#edf0ee" } } },
      yAxis: { type: "category", inverse: true, data: values.map((item) => item.name), triggerEvent: true, axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: "#3b4642", width: 150, overflow: "truncate", cursor: "pointer" } },
      series: [{
        id: "project-time",
        type: "bar",
        data: values.map((item) => ({ id: item.id, name: item.name, value: item.seconds })),
        barWidth: 18,
        cursor: "pointer",
        itemStyle: { color: "#167c5a", borderRadius: 2 },
      }],
    }, { lazyUpdate: true });
    hasRendered.current = true;
  }, [values]);

  return <div className="chart-shell">
    <div ref={ref} className="chart" role="img" aria-label="Agent time by project" />
    {!values.length && <div className="empty chart-empty">No activity has been recorded in this period yet.</div>}
  </div>;
}

function formatDuration(seconds: number) { const hours = Math.floor(seconds / 3600); const minutes = Math.round((seconds % 3600) / 60); return hours ? `${hours} h ${minutes} min` : `${minutes} min`; }

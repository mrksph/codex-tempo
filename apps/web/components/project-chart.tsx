"use client";

import * as echarts from "echarts";
import type { ECElementEvent } from "echarts";
import { useRouter } from "next/navigation";
import { useEffect, useRef } from "react";

export function ProjectChart({ values }: { values: { id: string; name: string; seconds: number }[] }) {
  const ref = useRef<HTMLDivElement>(null);
  const router = useRouter();
  useEffect(() => {
    if (!ref.current || values.length === 0) return;
    const chart = echarts.init(ref.current);
    chart.setOption({
      animationDuration: 450,
      grid: { left: 18, right: 28, top: 24, bottom: 42, containLabel: true },
      tooltip: { trigger: "axis", valueFormatter: (value: unknown) => formatDuration(Number(value)) },
      xAxis: { type: "value", axisLabel: { color: "#66716d", formatter: (value: number) => value < 3600 ? `${Math.round(value / 60)}m` : `${Math.round(value / 3600)}h` }, splitLine: { lineStyle: { color: "#edf0ee" } } },
      yAxis: { type: "category", inverse: true, data: values.map((item) => item.name), triggerEvent: true, axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: "#3b4642", width: 150, overflow: "truncate", cursor: "pointer" } },
      series: [{ type: "bar", data: values.map((item) => item.seconds), barWidth: 18, cursor: "pointer", itemStyle: { color: "#167c5a", borderRadius: 2 } }],
    });
    const openProject = (event: ECElementEvent) => {
      const index = event.componentType === "series"
        ? event.dataIndex
        : values.findIndex((item) => item.name === String(event.value));
      if (typeof index === "number" && index >= 0 && values[index]) router.push(`/projects/${values[index].id}`);
    };
    chart.on("click", openProject);
    const resize = () => chart.resize(); window.addEventListener("resize", resize);
    return () => { window.removeEventListener("resize", resize); chart.off("click", openProject); chart.dispose(); };
  }, [router, values]);
  if (!values.length) return <div className="empty">Aún no hay actividad en este periodo.</div>;
  return <div ref={ref} className="chart" role="img" aria-label="Tiempo de agente por proyecto"/>;
}

function formatDuration(seconds: number) { const hours = Math.floor(seconds / 3600); const minutes = Math.round((seconds % 3600) / 60); return hours ? `${hours} h ${minutes} min` : `${minutes} min`; }

export function Metric({ label, value, meta }: { label: string; value: string; meta: string }) {
  return <div className="metric"><div className="metric-label">{label}</div><div className="metric-value">{value}</div><div className="metric-meta">{meta}</div></div>;
}

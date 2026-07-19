// Simple dependency-free funnel bar chart.

const STEPS: { key: string; label: string; color: string }[] = [
  { key: "sent", label: "Sent", color: "#6366f1" },
  { key: "open", label: "Opened", color: "#0ea5e9" },
  { key: "click", label: "Clicked", color: "#f59e0b" },
  { key: "submit", label: "Submitted", color: "#dc2626" },
  { key: "report", label: "Reported", color: "#16a34a" },
];

export function FunnelBars({ funnel }: { funnel: Record<string, number> }) {
  const total = funnel.targets || 0;
  const pct = (n: number) => (total ? Math.round((n / total) * 100) : 0);
  return (
    <div className="space-y-3">
      {STEPS.map((s) => {
        const n = funnel[s.key] ?? 0;
        return (
          <div key={s.key} className="flex items-center gap-3">
            <div className="w-24 text-sm font-medium">{s.label}</div>
            <div className="h-6 flex-1 overflow-hidden rounded-md" style={{ background: "#f1f5f9" }}>
              <div
                className="flex h-6 items-center justify-end rounded-md px-2 text-xs font-semibold text-white transition-all"
                style={{ width: `${Math.max(pct(n), n > 0 ? 8 : 0)}%`, background: s.color, minWidth: n > 0 ? 28 : 0 }}
              >
                {n > 0 ? n : ""}
              </div>
            </div>
            <div className="w-12 text-right text-sm muted">{pct(n)}%</div>
          </div>
        );
      })}
      <div className="text-xs muted">{total} targets total</div>
    </div>
  );
}

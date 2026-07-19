import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { api } from "../api";

const STEPS = [
  { key: "sent", label: "Sent" },
  { key: "open", label: "Opened" },
  { key: "click", label: "Clicked" },
  { key: "submit", label: "Submitted" },
  { key: "report", label: "Reported" },
];

export default function CampaignReport() {
  const { id } = useParams();
  const [report, setReport] = useState<any>(null);
  const [timeline, setTimeline] = useState<any[]>([]);
  const [err, setErr] = useState("");

  async function load() {
    try {
      const [r, t] = await Promise.all([api(`campaigns/${id}/report`), api(`campaigns/${id}/timeline`)]);
      setReport(r);
      setTimeline(t);
    } catch (e: any) {
      setErr(e.message);
    }
  }
  useEffect(() => {
    load();
    const iv = setInterval(load, 5000); // live-ish refresh
    return () => clearInterval(iv);
  }, [id]);

  if (err) return <div className="text-red-300">{err}</div>;
  if (!report) return <div className="text-slate-400">Loading…</div>;

  const f = report.funnel;
  const total = f.targets || 0;
  const pct = (n: number) => (total ? Math.round((n / total) * 100) : 0);

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link to={`/engagements/${report.campaign.engagement_id}`} className="text-slate-400 hover:text-slate-200">
          ← Engagement
        </Link>
        <h1 className="text-xl font-semibold">{report.campaign.name}</h1>
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
        {STEPS.map((s) => (
          <div key={s.key} className="card">
            <div className="text-2xl font-semibold">{f[s.key] ?? 0}</div>
            <div className="text-xs text-slate-400">{s.label}</div>
            <div className="mt-2 h-1.5 w-full rounded bg-slate-800">
              <div className="h-1.5 rounded bg-sky-500" style={{ width: `${pct(f[s.key] ?? 0)}%` }} />
            </div>
            <div className="mt-1 text-right text-[10px] text-slate-500">{pct(f[s.key] ?? 0)}%</div>
          </div>
        ))}
      </div>
      <p className="text-sm text-slate-400">{total} targets in this campaign.</p>

      <div className="card overflow-x-auto">
        <h2 className="mb-2 font-medium">Timeline</h2>
        <table className="data">
          <thead>
            <tr>
              <th>When</th>
              <th>Target</th>
              <th>Event</th>
              <th>IP</th>
            </tr>
          </thead>
          <tbody>
            {timeline.map((ev, i) => (
              <tr key={i}>
                <td className="text-slate-400">{new Date(ev.created_at).toLocaleString()}</td>
                <td>{ev.email}</td>
                <td>
                  <span className="badge bg-slate-700 text-slate-200">{ev.type}</span>
                </td>
                <td className="text-slate-500">{ev.ip}</td>
              </tr>
            ))}
            {timeline.length === 0 && (
              <tr>
                <td colSpan={4} className="text-center text-slate-500">
                  No events yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

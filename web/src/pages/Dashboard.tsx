import { useEffect, useState } from "react";
import { api } from "../api";
import { FunnelBars } from "../components/Funnel";

interface Stats {
  engagements_total: number;
  engagements_active: number;
  role: string;
  funnel: Record<string, number>;
}

export default function Dashboard() {
  const [s, setS] = useState<Stats | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    api<Stats>("dashboard").then(setS).catch((e) => setErr(e.message));
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      {err && <div className="text-sm" style={{ color: "#b91c1c" }}>{err}</div>}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Stat label="Engagements" value={s?.engagements_total ?? "—"} />
        <Stat label="Active" value={s?.engagements_active ?? "—"} />
        <Stat label="Targets contacted" value={s?.funnel?.targets ?? "—"} />
        <Stat label="Your role" value={s?.role ?? "—"} />
      </div>

      <div className="card">
        <div className="section-title mb-3">Organization funnel (all campaigns)</div>
        {s?.funnel ? <FunnelBars funnel={s.funnel} /> : <div className="muted text-sm">Loading…</div>}
      </div>

      <div className="card text-sm">
        <p className="font-semibold">Authorized use only</p>
        <p className="mt-1 muted">
          Every campaign runs inside an <b>engagement</b> that records the client, a written
          authorization reference, and a date window. Targets outside the engagement allowlist are
          rejected, and all actions are written to an append-only audit log.
        </p>
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: any }) {
  return (
    <div className="stat">
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
    </div>
  );
}

import { useEffect, useState } from "react";
import { api } from "../api";

interface Stats {
  engagements_total: number;
  engagements_active: number;
  role: string;
}

export default function Dashboard() {
  const [s, setS] = useState<Stats | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    api<Stats>("dashboard").then(setS).catch((e) => setErr(e.message));
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold">Dashboard</h1>
      {err && <div className="text-red-300">{err}</div>}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
        <Stat label="Engagements" value={s?.engagements_total ?? "—"} />
        <Stat label="Active" value={s?.engagements_active ?? "—"} />
        <Stat label="Your role" value={s?.role ?? "—"} />
      </div>
      <div className="card text-sm text-slate-300">
        <p className="font-medium text-slate-100">Authorized use only</p>
        <p className="mt-1 text-slate-400">
          Every campaign runs inside an <b>engagement</b> that records the client, a written
          authorization reference, and a date window. Targets outside the engagement
          allowlist are rejected, and all actions are written to an append-only audit log.
        </p>
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: any }) {
  return (
    <div className="card">
      <div className="text-3xl font-semibold">{value}</div>
      <div className="mt-1 text-xs uppercase tracking-wide text-slate-400">{label}</div>
    </div>
  );
}

import { useEffect, useState } from "react";
import { api } from "../api";

export default function Audit() {
  const [rows, setRows] = useState<any[]>([]);
  const [err, setErr] = useState("");
  useEffect(() => {
    api<any[]>("audit-log").then(setRows).catch((e) => setErr(e.message));
  }, []);
  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">Audit log</h1>
      <p className="text-sm text-slate-400">Append-only record of privileged actions within your organization.</p>
      {err && <div className="text-red-300">{err}</div>}
      <div className="card overflow-x-auto">
        <table className="data">
          <thead>
            <tr>
              <th>When</th>
              <th>Action</th>
              <th>Entity</th>
              <th>Detail</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.id}>
                <td className="text-slate-400">{new Date(r.created_at).toLocaleString()}</td>
                <td>
                  <span className="badge bg-slate-700 text-slate-200">{r.action}</span>
                </td>
                <td className="text-slate-400">{r.entity}</td>
                <td className="font-mono text-xs text-slate-500">{JSON.stringify(r.meta)}</td>
              </tr>
            ))}
            {rows.length === 0 && (
              <tr>
                <td colSpan={4} className="text-center text-slate-500">
                  No audit entries yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

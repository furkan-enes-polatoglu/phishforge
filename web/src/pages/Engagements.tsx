import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api";

interface Engagement {
  id: string;
  client_name: string;
  authz_ref: string;
  status: string;
  starts_at: string;
  ends_at: string;
}

export default function Engagements() {
  const [list, setList] = useState<Engagement[]>([]);
  const [err, setErr] = useState("");
  const [form, setForm] = useState({ client_name: "", authz_ref: "", starts_at: "", ends_at: "" });

  async function load() {
    try {
      setList(await api<Engagement[]>("engagements"));
    } catch (e: any) {
      setErr(e.message);
    }
  }
  useEffect(() => {
    load();
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setErr("");
    try {
      await api("engagements", {
        method: "POST",
        body: {
          client_name: form.client_name,
          authz_ref: form.authz_ref,
          starts_at: new Date(form.starts_at).toISOString(),
          ends_at: new Date(form.ends_at).toISOString(),
        },
      });
      setForm({ client_name: "", authz_ref: "", starts_at: "", ends_at: "" });
      load();
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Engagements</h1>
      {err && <div className="rounded-lg px-3 py-2 text-sm" style={{ background: "#fee2e2", color: "#991b1b" }}>{err}</div>}

      <form onSubmit={create} className="card grid gap-3 sm:grid-cols-2">
        <div className="sm:col-span-2 section-title">New engagement (authorization record)</div>
        <div>
          <label className="label">Client name</label>
          <input className="input" value={form.client_name} onChange={(e) => setForm({ ...form, client_name: e.target.value })} required />
        </div>
        <div>
          <label className="label">Authorization reference</label>
          <input className="input" placeholder="e.g. signed SoW #2026-07-19" value={form.authz_ref} onChange={(e) => setForm({ ...form, authz_ref: e.target.value })} required />
        </div>
        <div>
          <label className="label">Starts</label>
          <input className="input" type="datetime-local" value={form.starts_at} onChange={(e) => setForm({ ...form, starts_at: e.target.value })} required />
        </div>
        <div>
          <label className="label">Ends</label>
          <input className="input" type="datetime-local" value={form.ends_at} onChange={(e) => setForm({ ...form, ends_at: e.target.value })} required />
        </div>
        <div className="sm:col-span-2">
          <button className="btn">Create engagement</button>
        </div>
      </form>

      <div className="card overflow-x-auto">
        <table className="data">
          <thead>
            <tr>
              <th>Client</th>
              <th>Authz ref</th>
              <th>Window</th>
              <th>Status</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {list.map((e) => (
              <tr key={e.id}>
                <td className="font-semibold">{e.client_name}</td>
                <td className="muted">{e.authz_ref}</td>
                <td className="muted">
                  {new Date(e.starts_at).toLocaleDateString()} → {new Date(e.ends_at).toLocaleDateString()}
                </td>
                <td>
                  <StatusBadge status={e.status} />
                </td>
                <td>
                  <Link className="btn-ghost" to={`/engagements/${e.id}`}>
                    Open
                  </Link>
                </td>
              </tr>
            ))}
            {list.length === 0 && (
              <tr>
                <td colSpan={5} className="text-center muted">
                  No engagements yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export function StatusBadge({ status }: { status: string }) {
  const cls: Record<string, string> = {
    active: "badge-green",
    draft: "badge-gray",
    closed: "badge-gray",
    running: "badge-blue",
    scheduled: "badge-amber",
    completed: "badge-green",
  };
  return <span className={`badge ${cls[status] || "badge-gray"}`}>{status}</span>;
}

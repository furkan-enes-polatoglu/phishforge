import { useEffect, useState } from "react";
import { api } from "../api";

export default function Training() {
  const [modules, setModules] = useState<any[]>([]);
  const [assignments, setAssignments] = useState<any[]>([]);
  const [f, setF] = useState({ name: "", html: "<h1>Spotting phishing</h1>\n<p>Always check the sender, hover links, and never enter credentials from an emailed link.</p>" });
  const [msg, setMsg] = useState("");

  async function load() {
    const [m, a] = await Promise.all([api("training-modules"), api("training-assignments")]);
    setModules(m); setAssignments(a);
  }
  useEffect(() => { load().catch((e) => setMsg(e.message)); }, []);

  async function save(e: React.FormEvent) {
    e.preventDefault(); setMsg("");
    try { await api("training-modules", { method: "POST", body: f }); setMsg("Saved."); load(); }
    catch (e: any) { setMsg(e.message); }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Security awareness training</h1>
      <p className="text-sm muted">
        Targets who click or submit are automatically assigned the first training module and
        redirected to it. Viewing the module marks it completed.
      </p>

      <div className="grid gap-6 lg:grid-cols-2">
        <form onSubmit={save} className="card space-y-3">
          <div className="section-title">New training module</div>
          <input className="input" placeholder="Name" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required />
          <textarea className="input h-40 font-mono text-xs" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
          <button className="btn">Save module</button>
          {msg && <div className="text-sm" style={{ color: "#166534" }}>{msg}</div>}
        </form>
        <div className="space-y-4">
          <div>
            <div className="label">Preview</div>
            <iframe title="training-preview" srcDoc={f.html} className="h-56 w-full rounded-lg border bg-white" style={{ borderColor: "var(--pf-border)" }} />
          </div>
          <div className="card">
            <div className="section-title mb-2">Modules ({modules.length})</div>
            <ul className="text-sm">{modules.map((m) => <li key={m.id} className="border-b py-1" style={{ borderColor: "#eef1f7" }}>{m.name}</li>)}</ul>
          </div>
        </div>
      </div>

      <div className="card overflow-x-auto">
        <div className="section-title mb-2">Assignments &amp; completion</div>
        <table className="data">
          <thead><tr><th>Target</th><th>Module</th><th>Status</th><th>Assigned</th><th>Completed</th></tr></thead>
          <tbody>
            {assignments.map((a, i) => (
              <tr key={i}>
                <td>{a.email}</td>
                <td>{a.module}</td>
                <td><span className={`badge ${a.status === "completed" ? "badge-green" : "badge-amber"}`}>{a.status}</span></td>
                <td className="muted">{new Date(a.assigned_at).toLocaleString()}</td>
                <td className="muted">{a.completed_at ? new Date(a.completed_at).toLocaleString() : "—"}</td>
              </tr>
            ))}
            {assignments.length === 0 && <tr><td colSpan={5} className="text-center muted">No assignments yet.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}

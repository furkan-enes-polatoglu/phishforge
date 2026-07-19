import { useEffect, useState } from "react";
import { api } from "../api";

const EVENTS = ["open", "click", "submit", "report"];

export default function Settings() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>
      <Webhooks />
      <APIKeys />
    </div>
  );
}

function Webhooks() {
  const [list, setList] = useState<any[]>([]);
  const [f, setF] = useState<any>({ url: "", secret: "", events: [] as string[] });
  const [msg, setMsg] = useState("");
  const load = () => api("webhooks").then(setList);
  useEffect(() => { load(); }, []);
  function toggle(ev: string) {
    setF((s: any) => ({ ...s, events: s.events.includes(ev) ? s.events.filter((e: string) => e !== ev) : [...s.events, ev] }));
  }
  async function save(e: React.FormEvent) {
    e.preventDefault(); setMsg("");
    try { await api("webhooks", { method: "POST", body: f }); setF({ url: "", secret: "", events: [] }); load(); }
    catch (e: any) { setMsg(e.message); }
  }
  async function del(id: string) { await api(`webhooks/${id}`, { method: "DELETE" }); load(); }
  return (
    <div className="card space-y-3">
      <div className="section-title">Notifications (webhooks &amp; Slack/Teams)</div>
      <p className="text-xs muted">Real-time alerts on target actions. Slack/Teams incoming-webhook URLs are auto-formatted; other URLs get a signed JSON payload (HMAC in X-PhishForge-Signature).</p>
      <form onSubmit={save} className="grid gap-3 sm:grid-cols-2">
        <input className="input" placeholder="Webhook URL" value={f.url} onChange={(e) => setF({ ...f, url: e.target.value })} required />
        <input className="input" placeholder="Signing secret (optional)" value={f.secret} onChange={(e) => setF({ ...f, secret: e.target.value })} />
        <div className="sm:col-span-2 flex flex-wrap gap-3">
          {EVENTS.map((ev) => (
            <label key={ev} className="checkbox-row"><input type="checkbox" checked={f.events.includes(ev)} onChange={() => toggle(ev)} /> {ev}</label>
          ))}
          <span className="text-xs muted">(none selected = all events)</span>
        </div>
        <div><button className="btn">Add webhook</button></div>
      </form>
      {msg && <div className="text-sm" style={{ color: "#b91c1c" }}>{msg}</div>}
      <table className="data">
        <thead><tr><th>URL</th><th>Events</th><th></th></tr></thead>
        <tbody>
          {list.map((w) => (
            <tr key={w.id}>
              <td className="font-mono text-xs">{w.url}</td>
              <td className="muted">{w.events?.length ? w.events.join(", ") : "all"}</td>
              <td><button className="btn-danger btn-sm" onClick={() => del(w.id)}>Delete</button></td>
            </tr>
          ))}
          {list.length === 0 && <tr><td colSpan={3} className="text-center muted">No webhooks.</td></tr>}
        </tbody>
      </table>
    </div>
  );
}

function APIKeys() {
  const [list, setList] = useState<any[]>([]);
  const [f, setF] = useState({ name: "", role: "operator" });
  const [created, setCreated] = useState("");
  const [msg, setMsg] = useState("");
  const load = () => api("api-keys").then(setList);
  useEffect(() => { load(); }, []);
  async function create(e: React.FormEvent) {
    e.preventDefault(); setMsg(""); setCreated("");
    try { const r: any = await api("api-keys", { method: "POST", body: f }); setCreated(r.key); setF({ name: "", role: "operator" }); load(); }
    catch (e: any) { setMsg(e.message); }
  }
  async function revoke(id: string) { await api(`api-keys/${id}`, { method: "DELETE" }); load(); }
  return (
    <div className="card space-y-3">
      <div className="section-title">API keys (automation)</div>
      <p className="text-xs muted">Use in the <code>X-API-Key</code> header. The full key is shown once at creation.</p>
      {created && (
        <div className="rounded-lg px-3 py-2 text-sm" style={{ background: "#dcfce7", color: "#166534" }}>
          New key (copy now, shown once): <span className="font-mono font-semibold">{created}</span>
        </div>
      )}
      <form onSubmit={create} className="flex flex-wrap items-end gap-2">
        <div><label className="label">Name</label><input className="input" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required /></div>
        <div><label className="label">Role</label>
          <select className="input" value={f.role} onChange={(e) => setF({ ...f, role: e.target.value })}>
            <option value="viewer">viewer</option><option value="operator">operator</option><option value="admin">admin</option>
          </select>
        </div>
        <button className="btn">Create key</button>
      </form>
      {msg && <div className="text-sm" style={{ color: "#b91c1c" }}>{msg}</div>}
      <table className="data">
        <thead><tr><th>Name</th><th>Prefix</th><th>Role</th><th>Last used</th><th></th></tr></thead>
        <tbody>
          {list.map((k) => (
            <tr key={k.id}>
              <td>{k.name}</td>
              <td className="font-mono text-xs">{k.prefix}…</td>
              <td><span className="badge badge-blue">{k.role}</span></td>
              <td className="muted">{k.last_used_at ? new Date(k.last_used_at).toLocaleString() : "never"}</td>
              <td>{k.revoked ? <span className="badge badge-gray">revoked</span> : <button className="btn-danger btn-sm" onClick={() => revoke(k.id)}>Revoke</button>}</td>
            </tr>
          ))}
          {list.length === 0 && <tr><td colSpan={5} className="text-center muted">No API keys.</td></tr>}
        </tbody>
      </table>
    </div>
  );
}

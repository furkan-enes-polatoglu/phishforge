import { useEffect, useState } from "react";
import { api } from "../api";
import { useI18n } from "../i18n";

const EVENTS = ["open", "click", "submit", "report"];

export default function Settings() {
  const { t } = useI18n();
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("settings")}</h1>
      <Webhooks />
      <APIKeys />
    </div>
  );
}

function Webhooks() {
  const { t } = useI18n();
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
      <div className="section-title">{t("notifications")}</div>
      <p className="text-xs muted">{t("notif_help")}</p>
      <form onSubmit={save} className="grid gap-3 sm:grid-cols-2">
        <input className="input" placeholder={t("webhook_url")} value={f.url} onChange={(e) => setF({ ...f, url: e.target.value })} required />
        <input className="input" placeholder={t("signing_secret")} value={f.secret} onChange={(e) => setF({ ...f, secret: e.target.value })} />
        <div className="sm:col-span-2 flex flex-wrap gap-3">
          {EVENTS.map((ev) => (
            <label key={ev} className="checkbox-row"><input type="checkbox" checked={f.events.includes(ev)} onChange={() => toggle(ev)} /> {ev}</label>
          ))}
          <span className="text-xs muted">{t("none_all_events")}</span>
        </div>
        <div><button className="btn">{t("add_webhook")}</button></div>
      </form>
      {msg && <div className="text-sm" style={{ color: "#b91c1c" }}>{msg}</div>}
      <table className="data">
        <thead><tr><th>URL</th><th>{t("events")}</th><th></th></tr></thead>
        <tbody>
          {list.map((w) => (
            <tr key={w.id}>
              <td className="font-mono text-xs">{w.url}</td>
              <td className="muted">{w.events?.length ? w.events.join(", ") : "all"}</td>
              <td><button className="btn-danger btn-sm" onClick={() => del(w.id)}>Delete</button></td>
            </tr>
          ))}
          {list.length === 0 && <tr><td colSpan={3} className="text-center muted">{t("none_yet")}</td></tr>}
        </tbody>
      </table>
    </div>
  );
}

function APIKeys() {
  const { t } = useI18n();
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
      <div className="section-title">{t("api_keys")}</div>
      <p className="text-xs muted">{t("api_help")}</p>
      {created && (
        <div className="rounded-lg px-3 py-2 text-sm" style={{ background: "#dcfce7", color: "#166534" }}>
          {t("api_created_once")} <span className="font-mono font-semibold">{created}</span>
        </div>
      )}
      <form onSubmit={create} className="flex flex-wrap items-end gap-2">
        <div><label className="label">{t("name")}</label><input className="input" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required /></div>
        <div><label className="label">{t("role")}</label>
          <select className="input" value={f.role} onChange={(e) => setF({ ...f, role: e.target.value })}>
            <option value="viewer">viewer</option><option value="operator">operator</option><option value="admin">admin</option>
          </select>
        </div>
        <button className="btn">{t("create_key")}</button>
      </form>
      {msg && <div className="text-sm" style={{ color: "#b91c1c" }}>{msg}</div>}
      <table className="data">
        <thead><tr><th>{t("name")}</th><th>{t("prefix")}</th><th>{t("role")}</th><th>{t("last_used")}</th><th></th></tr></thead>
        <tbody>
          {list.map((k) => (
            <tr key={k.id}>
              <td>{k.name}</td>
              <td className="font-mono text-xs">{k.prefix}…</td>
              <td><span className="badge badge-blue">{k.role}</span></td>
              <td className="muted">{k.last_used_at ? new Date(k.last_used_at).toLocaleString() : t("never")}</td>
              <td>{k.revoked ? <span className="badge badge-gray">{t("revoked")}</span> : <button className="btn-danger btn-sm" onClick={() => revoke(k.id)}>{t("revoke")}</button>}</td>
            </tr>
          ))}
          {list.length === 0 && <tr><td colSpan={5} className="text-center muted">{t("none_yet")}</td></tr>}
        </tbody>
      </table>
    </div>
  );
}

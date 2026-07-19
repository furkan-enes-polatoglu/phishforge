import { useEffect, useState } from "react";
import { api } from "../api";
import { useI18n } from "../i18n";

export default function Training() {
  const { t } = useI18n();
  const [modules, setModules] = useState<any[]>([]);
  const [assignments, setAssignments] = useState<any[]>([]);
  const empty = { name: "", html: "<h1>Oltalamayı fark etmek</h1>\n<p>Göndereni kontrol edin, bağlantıların üzerine gelin ve e-postadaki bir bağlantıdan asla kimlik bilgisi girmeyin.</p>" };
  const [f, setF] = useState<any>(empty);
  const [editId, setEditId] = useState<string | null>(null);
  const [msg, setMsg] = useState("");

  async function load() {
    const [m, a] = await Promise.all([api("training-modules"), api("training-assignments")]);
    setModules(m); setAssignments(a);
  }
  useEffect(() => { load().catch((e) => setMsg(e.message)); }, []);

  async function save(e: React.FormEvent) {
    e.preventDefault(); setMsg("");
    try {
      if (editId) await api(`training-modules/${editId}`, { method: "PUT", body: f });
      else await api("training-modules", { method: "POST", body: f });
      setF(empty); setEditId(null); setMsg(t("saved")); load();
    } catch (e: any) { setMsg(e.message); }
  }
  function edit(x: any) { setEditId(x.id); setF({ name: x.name, html: x.html }); }
  async function del(id: string, name: string) {
    if (!confirm(`"${name}" ${t("confirm_delete")}`)) return;
    await api(`training-modules/${id}`, { method: "DELETE" }); load();
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("training_title")}</h1>
      <p className="text-sm muted">{t("training_help")}</p>

      <div className="grid gap-6 lg:grid-cols-2">
        <form onSubmit={save} className="card space-y-3">
          <div className="section-title">{editId ? t("edit") : t("new_module")}</div>
          <input className="input" placeholder={t("name")} value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required />
          <textarea className="input h-40 font-mono text-xs" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
          <div className="flex gap-2">
            <button className="btn">{editId ? t("update") : t("save")}</button>
            {editId && <button type="button" className="btn-ghost" onClick={() => { setEditId(null); setF(empty); }}>{t("cancel")}</button>}
          </div>
          {msg && <div className="text-sm" style={{ color: "#166534" }}>{msg}</div>}
        </form>
        <div className="space-y-4">
          <div>
            <div className="label">{t("preview")}</div>
            <iframe title="training-preview" srcDoc={f.html} className="h-56 w-full rounded-lg border bg-white" style={{ borderColor: "var(--pf-border)" }} />
          </div>
          <div className="card">
            <div className="section-title mb-2">{t("modules")} ({modules.length})</div>
            <table className="data">
              <tbody>
                {modules.map((m) => (
                  <tr key={m.id}>
                    <td>{m.name}</td>
                    <td className="text-right">
                      <button className="btn-ghost btn-sm" onClick={() => edit(m)}>{t("edit")}</button>{" "}
                      <button className="btn-danger btn-sm" onClick={() => del(m.id, m.name)}>{t("delete")}</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div className="card overflow-x-auto">
        <div className="section-title mb-2">{t("assignments")}</div>
        <table className="data">
          <thead><tr><th>{t("target")}</th><th>{t("module")}</th><th>{t("status")}</th><th>{t("assigned")}</th><th>{t("completed")}</th></tr></thead>
          <tbody>
            {assignments.map((a, i) => (
              <tr key={i}>
                <td>{a.email}</td>
                <td>{a.module}</td>
                <td><span className={`badge ${a.status === "completed" ? "badge-green" : "badge-amber"}`}>{a.status === "completed" ? t("completed") : t("assigned")}</span></td>
                <td className="muted">{new Date(a.assigned_at).toLocaleString()}</td>
                <td className="muted">{a.completed_at ? new Date(a.completed_at).toLocaleString() : "—"}</td>
              </tr>
            ))}
            {assignments.length === 0 && <tr><td colSpan={5} className="text-center muted">{t("none_yet")}</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}

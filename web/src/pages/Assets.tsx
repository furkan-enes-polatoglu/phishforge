import { useEffect, useState } from "react";
import { api } from "../api";
import { useI18n } from "../i18n";

type Tab = "email" | "landing" | "profiles";

export default function Assets() {
  const { t } = useI18n();
  const [tab, setTab] = useState<Tab>("email");
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("assets")}</h1>
      <div className="flex gap-2">
        {(["email", "landing", "profiles"] as Tab[]).map((tb) => (
          <button key={tb} className={tab === tb ? "tab tab-active" : "tab"} onClick={() => setTab(tb)}>
            {tb === "email" ? t("email_templates") : tb === "landing" ? t("landing_pages") : t("sending_profiles")}
          </button>
        ))}
      </div>
      {tab === "email" && <EmailTemplates />}
      {tab === "landing" && <LandingPages />}
      {tab === "profiles" && <SendingProfiles />}
    </div>
  );
}

function Preview({ html }: { html: string }) {
  const { t } = useI18n();
  return (
    <div>
      <div className="label">{t("live_preview")}</div>
      <iframe title="preview" srcDoc={html} className="h-64 w-full rounded-lg border bg-white" style={{ borderColor: "var(--pf-border)" }} />
    </div>
  );
}

// Reusable action buttons for a list row.
function RowActions({ onEdit, onDup, onDel, name }: { onEdit: () => void; onDup: () => void; onDel: () => void; name: string }) {
  const { t } = useI18n();
  return (
    <span className="flex gap-1">
      <button className="btn-ghost btn-sm" onClick={onEdit}>{t("edit")}</button>
      <button className="btn-ghost btn-sm" onClick={onDup}>{t("duplicate")}</button>
      <button className="btn-danger btn-sm" onClick={() => { if (confirm(`"${name}" ${t("confirm_delete")}`)) onDel(); }}>{t("delete")}</button>
    </span>
  );
}

function EmailTemplates() {
  const { t } = useI18n();
  const [list, setList] = useState<any[]>([]);
  const empty = { name: "", subject: "", html: '<p>Merhaba {{.FirstName}},</p>\n<p><a href="{{.TrackURL}}">Belgeyi inceleyin</a></p>', text: "" };
  const [f, setF] = useState<any>(empty);
  const [editId, setEditId] = useState<string | null>(null);
  const [msg, setMsg] = useState("");
  const load = () => api("email-templates").then(setList);
  useEffect(() => { load(); }, []);
  async function save(e: React.FormEvent) {
    e.preventDefault(); setMsg("");
    try {
      if (editId) await api(`email-templates/${editId}`, { method: "PUT", body: f });
      else await api("email-templates", { method: "POST", body: f });
      setF(empty); setEditId(null); setMsg(t("saved")); load();
    } catch (e: any) { setMsg(e.message); }
  }
  function edit(x: any) { setEditId(x.id); setF({ name: x.name, subject: x.subject, html: x.html, text: x.text || "" }); }
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <form onSubmit={save} className="card space-y-3">
        <div className="section-title">{editId ? t("edit") : t("new_module").replace("modülü", "şablon")} · {t("email_template")}</div>
        <p className="text-xs muted">{t("merge_tags")}: {"{{.FirstName}} {{.TrackURL}} {{.TrackPixel}} {{.ReportURL}}"}</p>
        <input className="input" placeholder={t("name")} value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required />
        <input className="input" placeholder={t("subject")} value={f.subject} onChange={(e) => setF({ ...f, subject: e.target.value })} required />
        <textarea className="input h-40 font-mono text-xs" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
        <div className="flex gap-2">
          <button className="btn">{editId ? t("update") : t("save")}</button>
          {editId && <button type="button" className="btn-ghost" onClick={() => { setEditId(null); setF(empty); }}>{t("cancel")}</button>}
        </div>
        {msg && <div className="text-sm" style={{ color: "#166534" }}>{msg}</div>}
      </form>
      <div className="space-y-4">
        <Preview html={f.html} />
        <div className="card">
          <div className="section-title mb-2">{t("existing")} ({list.length})</div>
          <table className="data">
            <tbody>
              {list.map((x) => (
                <tr key={x.id}>
                  <td className="font-medium">{x.name}<div className="muted text-xs">{x.subject}</div></td>
                  <td className="text-right"><RowActions name={x.name} onEdit={() => edit(x)} onDup={() => api(`email-templates/${x.id}/duplicate`, { method: "POST" }).then(load)} onDel={() => api(`email-templates/${x.id}`, { method: "DELETE" }).then(load)} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function LandingPages() {
  const { t } = useI18n();
  const [list, setList] = useState<any[]>([]);
  const empty = { name: "", html: '<h3>Giriş yap</h3>\n<form method="post" action="{{.SubmitURL}}">\n  <input name="username" value="{{.Email}}">\n  <input name="password" type="password">\n  <button>Giriş</button>\n</form>', capture_meta: false, capture_submitted_data: false, capture_passwords: false, redirect_url: "" };
  const [f, setF] = useState<any>(empty);
  const [editId, setEditId] = useState<string | null>(null);
  const [imp, setImp] = useState({ name: "", url: "" });
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);
  const load = () => api("landing-pages").then(setList);
  useEffect(() => { load(); }, []);
  async function save(e: React.FormEvent) {
    e.preventDefault(); setMsg("");
    try {
      if (editId) await api(`landing-pages/${editId}`, { method: "PUT", body: f });
      else await api("landing-pages", { method: "POST", body: f });
      setF(empty); setEditId(null); setMsg(t("saved")); load();
    } catch (e: any) { setMsg(e.message); }
  }
  async function importUrl(e: React.FormEvent) {
    e.preventDefault(); setMsg(""); setBusy(true);
    try { const r: any = await api("landing-pages/import", { method: "POST", body: imp }); setF({ ...empty, name: r.name, html: r.html }); setEditId(null); setMsg(t("saved")); load(); }
    catch (e: any) { setMsg(e.message); } finally { setBusy(false); }
  }
  function edit(x: any) { setEditId(x.id); setF({ name: x.name, html: x.html, capture_meta: x.capture_meta, capture_submitted_data: x.capture_submitted_data, capture_passwords: x.capture_passwords, redirect_url: x.redirect_url }); }
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <div className="space-y-4">
        <form onSubmit={importUrl} className="card space-y-2">
          <div className="section-title">{t("clone_from_url")}</div>
          <div className="flex gap-2">
            <input className="input" placeholder="example.com/login" value={imp.url} onChange={(e) => setImp({ ...imp, url: e.target.value })} />
            <button className="btn" disabled={busy}>{busy ? t("fetching") : t("import")}</button>
          </div>
          <p className="text-xs muted">{t("clone_help")}</p>
        </form>
        <form onSubmit={save} className="card space-y-3">
          <div className="section-title">{editId ? t("edit") : t("landing_page")}</div>
          <input className="input" placeholder={t("name")} value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required />
          <textarea className="input h-40 font-mono text-xs" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
          <input className="input" placeholder={t("redirect_after_submit")} value={f.redirect_url} onChange={(e) => setF({ ...f, redirect_url: e.target.value })} />
          <div className="rounded-lg border p-3 text-sm" style={{ borderColor: "var(--pf-border)", background: "#fafbff" }}>
            <div className="mb-2 font-semibold">{t("capture_settings")}</div>
            <label className="checkbox-row"><input type="checkbox" checked={f.capture_meta} onChange={(e) => setF({ ...f, capture_meta: e.target.checked })} /> {t("capture_field_names")}</label>
            <label className="checkbox-row mt-1"><input type="checkbox" checked={f.capture_submitted_data} onChange={(e) => setF({ ...f, capture_submitted_data: e.target.checked })} /> {t("capture_values")}</label>
            <label className="checkbox-row mt-1"><input type="checkbox" checked={f.capture_passwords} onChange={(e) => setF({ ...f, capture_passwords: e.target.checked })} /> {t("capture_passwords")}</label>
            {f.capture_passwords && <p className="mt-2 rounded px-2 py-1 text-xs" style={{ background: "#fee2e2", color: "#991b1b" }}>{t("capture_pw_warn")}</p>}
          </div>
          <div className="flex gap-2">
            <button className="btn">{editId ? t("update") : t("save")}</button>
            {editId && <button type="button" className="btn-ghost" onClick={() => { setEditId(null); setF(empty); }}>{t("cancel")}</button>}
          </div>
          {msg && <div className="text-sm" style={{ color: "#166534" }}>{msg}</div>}
        </form>
      </div>
      <div className="space-y-4">
        <Preview html={f.html} />
        <div className="card">
          <div className="section-title mb-2">{t("existing")} ({list.length})</div>
          <table className="data">
            <tbody>
              {list.map((x) => (
                <tr key={x.id}>
                  <td>{x.name}
                    <div className="mt-0.5 flex gap-1">
                      {x.capture_submitted_data && <span className="badge badge-amber">{t("captures_data")}</span>}
                      {x.capture_passwords && <span className="badge badge-red">{t("captures_pw")}</span>}
                    </div>
                  </td>
                  <td className="text-right"><RowActions name={x.name} onEdit={() => edit(x)} onDup={() => api(`landing-pages/${x.id}/duplicate`, { method: "POST" }).then(load)} onDel={() => api(`landing-pages/${x.id}`, { method: "DELETE" }).then(load)} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function SendingProfiles() {
  const { t } = useI18n();
  const [list, setList] = useState<any[]>([]);
  const empty = { name: "", smtp_host: "", smtp_port: 587, username: "", password: "", from_address: "", from_name: "", use_tls: true, dkim_domain: "", dkim_selector: "", sign_dkim: false };
  const [f, setF] = useState<any>(empty);
  const [editId, setEditId] = useState<string | null>(null);
  const [msg, setMsg] = useState("");
  const [dkim, setDkim] = useState<any>(null);
  const load = () => api("sending-profiles").then(setList);
  useEffect(() => { load(); }, []);
  async function save(e: React.FormEvent) {
    e.preventDefault(); setMsg("");
    try {
      if (editId) await api(`sending-profiles/${editId}`, { method: "PUT", body: f });
      else await api("sending-profiles", { method: "POST", body: f });
      setF(empty); setEditId(null); setMsg(t("saved")); load();
    } catch (e: any) { setMsg(e.message); }
  }
  function edit(x: any) { setEditId(x.id); setDkim(null); setF({ name: x.name, smtp_host: x.smtp_host, smtp_port: x.smtp_port, username: x.username, password: "", from_address: x.from_address, from_name: x.from_name, use_tls: x.use_tls, dkim_domain: x.dkim_domain || "", dkim_selector: x.dkim_selector || "", sign_dkim: x.sign_dkim }); }
  async function genDkim() {
    if (!editId) { setMsg("Önce profili kaydedin, sonra DKIM üretin."); return; }
    try { const r: any = await api(`sending-profiles/${editId}/dkim`, { method: "POST", body: { domain: f.dkim_domain, selector: f.dkim_selector } }); setDkim(r); setF({ ...f, sign_dkim: true }); load(); }
    catch (e: any) { setMsg(e.message); }
  }
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <form onSubmit={save} className="card grid gap-3 sm:grid-cols-2">
        <div className="col-span-2 section-title">{editId ? t("edit") : t("sending_profile")}</div>
        <input className="input" placeholder={t("name")} value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required />
        <input className="input" placeholder={t("from_address")} value={f.from_address} onChange={(e) => setF({ ...f, from_address: e.target.value })} required />
        <input className="input" placeholder={t("smtp_host")} value={f.smtp_host} onChange={(e) => setF({ ...f, smtp_host: e.target.value })} required />
        <input className="input" type="number" placeholder={t("port")} value={f.smtp_port} onChange={(e) => setF({ ...f, smtp_port: +e.target.value })} />
        <input className="input" placeholder={t("username")} value={f.username} onChange={(e) => setF({ ...f, username: e.target.value })} />
        <input className="input" type="password" placeholder={t("password") + (editId ? " " + t("leave_blank_keep") : "")} value={f.password} onChange={(e) => setF({ ...f, password: e.target.value })} />
        <input className="input" placeholder={t("from_name")} value={f.from_name} onChange={(e) => setF({ ...f, from_name: e.target.value })} />
        <label className="checkbox-row"><input type="checkbox" checked={f.use_tls} onChange={(e) => setF({ ...f, use_tls: e.target.checked })} /> STARTTLS</label>

        <div className="col-span-2 rounded-lg border p-3" style={{ borderColor: "var(--pf-border)", background: "#fafbff" }}>
          <div className="mb-1 font-semibold text-sm">{t("dkim_signing")}</div>
          <p className="mb-2 text-xs muted">{t("dkim_help")}</p>
          <div className="grid gap-2 sm:grid-cols-2">
            <input className="input" placeholder={t("dkim_domain")} value={f.dkim_domain} onChange={(e) => setF({ ...f, dkim_domain: e.target.value })} />
            <input className="input" placeholder={t("dkim_selector") + " (phishforge)"} value={f.dkim_selector} onChange={(e) => setF({ ...f, dkim_selector: e.target.value })} />
          </div>
          <label className="checkbox-row mt-2"><input type="checkbox" checked={f.sign_dkim} onChange={(e) => setF({ ...f, sign_dkim: e.target.checked })} /> {t("sign_with_dkim")}</label>
          <button type="button" className="btn-ghost btn-sm mt-2" onClick={genDkim}>{t("generate_dkim")}</button>
          {dkim && (
            <div className="mt-2 rounded px-2 py-2 text-xs" style={{ background: "#ecfdf5" }}>
              <div className="font-semibold">{t("dkim_dns_record")}</div>
              <div className="mt-1 font-mono break-all"><b>{dkim.dns_record_name}</b> TXT</div>
              <div className="font-mono break-all">{dkim.dns_record_value}</div>
            </div>
          )}
        </div>

        <div className="col-span-2 flex gap-2">
          <button className="btn">{editId ? t("update") : t("save")}</button>
          {editId && <button type="button" className="btn-ghost" onClick={() => { setEditId(null); setF(empty); setDkim(null); }}>{t("cancel")}</button>}
        </div>
        {msg && <div className="col-span-2 text-sm" style={{ color: "#166534" }}>{msg}</div>}
      </form>
      <div className="card">
        <div className="section-title mb-2">{t("existing")} ({list.length})</div>
        <table className="data">
          <tbody>
            {list.map((x) => (
              <tr key={x.id}>
                <td>{x.name}<div className="muted text-xs">{x.from_address} {x.sign_dkim && <span className="badge badge-green">DKIM</span>}</div></td>
                <td className="text-right"><RowActions name={x.name} onEdit={() => edit(x)} onDup={() => api(`sending-profiles/${x.id}/duplicate`, { method: "POST" }).then(load)} onDel={() => api(`sending-profiles/${x.id}`, { method: "DELETE" }).then(load)} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

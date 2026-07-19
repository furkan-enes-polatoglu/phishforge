import { useEffect, useState } from "react";
import { api } from "../api";

type Tab = "email" | "landing" | "profiles";

export default function Assets() {
  const [tab, setTab] = useState<Tab>("email");
  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold">Assets</h1>
      <div className="flex gap-2">
        {(["email", "landing", "profiles"] as Tab[]).map((t) => (
          <button key={t} className={tab === t ? "btn" : "btn-ghost"} onClick={() => setTab(t)}>
            {t === "email" ? "Email templates" : t === "landing" ? "Landing pages" : "Sending profiles"}
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
  return (
    <div>
      <div className="label">Live preview</div>
      <iframe title="preview" srcDoc={html} className="h-64 w-full rounded-md border border-slate-700 bg-white" />
    </div>
  );
}

function EmailTemplates() {
  const [list, setList] = useState<any[]>([]);
  const [f, setF] = useState({ name: "", subject: "", html: "<p>Hi {{.FirstName}},</p>\n<p><a href=\"{{.TrackURL}}\">Review the document</a></p>", text: "" });
  const [msg, setMsg] = useState("");
  const load = () => api("email-templates").then(setList);
  useEffect(() => { load(); }, []);
  async function save(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await api("email-templates", { method: "POST", body: f });
      setMsg("Saved.");
      load();
    } catch (e: any) { setMsg(e.message); }
  }
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <form onSubmit={save} className="card space-y-3">
        <h2 className="font-medium">New email template</h2>
        <p className="text-xs text-slate-400">Merge-tags: {"{{.FirstName}} {{.TrackURL}} {{.TrackPixel}} {{.ReportURL}}"}</p>
        <input className="input" placeholder="Name" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required />
        <input className="input" placeholder="Subject" value={f.subject} onChange={(e) => setF({ ...f, subject: e.target.value })} required />
        <textarea className="input h-40 font-mono text-xs" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
        <button className="btn">Save template</button>
        {msg && <div className="text-sm text-amber-200">{msg}</div>}
      </form>
      <div className="space-y-4">
        <Preview html={f.html} />
        <div className="card">
          <h3 className="mb-2 text-sm font-medium">Existing ({list.length})</h3>
          <ul className="text-sm">
            {list.map((t) => <li key={t.id} className="border-b border-slate-800/50 py-1">{t.name} — <span className="text-slate-500">{t.subject}</span></li>)}
          </ul>
        </div>
      </div>
    </div>
  );
}

function LandingPages() {
  const [list, setList] = useState<any[]>([]);
  const [f, setF] = useState({ name: "", html: "<h3>Sign in</h3>\n<form method=\"post\" action=\"{{.SubmitURL}}\">\n  <input name=\"username\" value=\"{{.Email}}\">\n  <input name=\"password\" type=\"password\">\n  <button>Sign in</button>\n</form>", capture_meta: false, redirect_url: "" });
  const [imp, setImp] = useState({ name: "", url: "" });
  const [msg, setMsg] = useState("");
  const load = () => api("landing-pages").then(setList);
  useEffect(() => { load(); }, []);
  async function save(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try { await api("landing-pages", { method: "POST", body: f }); setMsg("Saved."); load(); }
    catch (e: any) { setMsg(e.message); }
  }
  async function importUrl(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try { const r: any = await api("landing-pages/import", { method: "POST", body: imp }); setF({ ...f, name: r.name, html: r.html }); setMsg("Imported into editor."); load(); }
    catch (e: any) { setMsg(e.message); }
  }
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <div className="space-y-4">
        <form onSubmit={importUrl} className="card flex gap-2">
          <input className="input" placeholder="Clone from URL (authorized)" value={imp.url} onChange={(e) => setImp({ ...imp, url: e.target.value })} />
          <button className="btn">Import</button>
        </form>
        <form onSubmit={save} className="card space-y-3">
          <h2 className="font-medium">New landing page</h2>
          <p className="text-xs text-slate-400">Form must POST to {"{{.SubmitURL}}"}. Submitted values are never stored.</p>
          <input className="input" placeholder="Name" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required />
          <textarea className="input h-40 font-mono text-xs" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
          <input className="input" placeholder="Awareness redirect URL (after submit)" value={f.redirect_url} onChange={(e) => setF({ ...f, redirect_url: e.target.value })} />
          <label className="flex items-center gap-2 text-sm text-slate-300">
            <input type="checkbox" checked={f.capture_meta} onChange={(e) => setF({ ...f, capture_meta: e.target.checked })} />
            Capture which field names were filled (never the values)
          </label>
          <button className="btn">Save landing page</button>
          {msg && <div className="text-sm text-amber-200">{msg}</div>}
        </form>
      </div>
      <div className="space-y-4">
        <Preview html={f.html} />
        <div className="card">
          <h3 className="mb-2 text-sm font-medium">Existing ({list.length})</h3>
          <ul className="text-sm">{list.map((l) => <li key={l.id} className="border-b border-slate-800/50 py-1">{l.name}</li>)}</ul>
        </div>
      </div>
    </div>
  );
}

function SendingProfiles() {
  const [list, setList] = useState<any[]>([]);
  const [f, setF] = useState({ name: "", smtp_host: "", smtp_port: 587, username: "", password: "", from_address: "", from_name: "", use_tls: true });
  const [msg, setMsg] = useState("");
  const load = () => api("sending-profiles").then(setList);
  useEffect(() => { load(); }, []);
  async function save(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try { await api("sending-profiles", { method: "POST", body: f }); setMsg("Saved."); load(); }
    catch (e: any) { setMsg(e.message); }
  }
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <form onSubmit={save} className="card grid gap-3 sm:grid-cols-2">
        <h2 className="col-span-2 font-medium">New sending profile</h2>
        <input className="input" placeholder="Name" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required />
        <input className="input" placeholder="From address" value={f.from_address} onChange={(e) => setF({ ...f, from_address: e.target.value })} required />
        <input className="input" placeholder="SMTP host" value={f.smtp_host} onChange={(e) => setF({ ...f, smtp_host: e.target.value })} required />
        <input className="input" type="number" placeholder="Port" value={f.smtp_port} onChange={(e) => setF({ ...f, smtp_port: Number(e.target.value) })} />
        <input className="input" placeholder="Username" value={f.username} onChange={(e) => setF({ ...f, username: e.target.value })} />
        <input className="input" type="password" placeholder="Password" value={f.password} onChange={(e) => setF({ ...f, password: e.target.value })} />
        <input className="input" placeholder="From name" value={f.from_name} onChange={(e) => setF({ ...f, from_name: e.target.value })} />
        <label className="flex items-center gap-2 text-sm text-slate-300">
          <input type="checkbox" checked={f.use_tls} onChange={(e) => setF({ ...f, use_tls: e.target.checked })} /> STARTTLS
        </label>
        <div className="col-span-2"><button className="btn">Save profile</button></div>
        {msg && <div className="col-span-2 text-sm text-amber-200">{msg}</div>}
      </form>
      <div className="card">
        <h3 className="mb-2 text-sm font-medium">Existing ({list.length})</h3>
        <ul className="text-sm">{list.map((p) => <li key={p.id} className="border-b border-slate-800/50 py-1">{p.name} — <span className="text-slate-500">{p.from_address}</span></li>)}</ul>
      </div>
    </div>
  );
}

import { useEffect, useState } from "react";
import { api } from "../api";

type Tab = "email" | "landing" | "profiles";

export default function Assets() {
  const [tab, setTab] = useState<Tab>("email");
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Assets</h1>
      <div className="flex gap-2">
        {(["email", "landing", "profiles"] as Tab[]).map((t) => (
          <button key={t} className={tab === t ? "tab tab-active" : "tab"} onClick={() => setTab(t)}>
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
      <iframe title="preview" srcDoc={html} className="h-64 w-full rounded-lg border bg-white" style={{ borderColor: "var(--pf-border)" }} />
    </div>
  );
}

function EmailTemplates() {
  const [list, setList] = useState<any[]>([]);
  const [f, setF] = useState({ name: "", subject: "", html: '<p>Hi {{.FirstName}},</p>\n<p><a href="{{.TrackURL}}">Review the document</a></p>', text: "" });
  const [msg, setMsg] = useState("");
  const load = () => api("email-templates").then(setList);
  useEffect(() => { load(); }, []);
  async function save(e: React.FormEvent) {
    e.preventDefault(); setMsg("");
    try { await api("email-templates", { method: "POST", body: f }); setMsg("Saved."); load(); }
    catch (e: any) { setMsg(e.message); }
  }
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <form onSubmit={save} className="card space-y-3">
        <div className="section-title">New email template</div>
        <p className="text-xs muted">Merge-tags: {"{{.FirstName}} {{.TrackURL}} {{.TrackPixel}} {{.ReportURL}}"}</p>
        <input className="input" placeholder="Name" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required />
        <input className="input" placeholder="Subject" value={f.subject} onChange={(e) => setF({ ...f, subject: e.target.value })} required />
        <textarea className="input h-40 font-mono text-xs" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
        <button className="btn">Save template</button>
        {msg && <div className="text-sm" style={{ color: "#166534" }}>{msg}</div>}
      </form>
      <div className="space-y-4">
        <Preview html={f.html} />
        <div className="card">
          <div className="section-title mb-2">Existing ({list.length})</div>
          <ul className="text-sm">{list.map((t) => <li key={t.id} className="border-b py-1" style={{ borderColor: "#eef1f7" }}>{t.name} — <span className="muted">{t.subject}</span></li>)}</ul>
        </div>
      </div>
    </div>
  );
}

function LandingPages() {
  const [list, setList] = useState<any[]>([]);
  const [f, setF] = useState<any>({ name: "", html: '<h3>Sign in</h3>\n<form method="post" action="{{.SubmitURL}}">\n  <input name="username" value="{{.Email}}">\n  <input name="password" type="password">\n  <button>Sign in</button>\n</form>', capture_meta: false, capture_submitted_data: false, capture_passwords: false, redirect_url: "" });
  const [imp, setImp] = useState({ name: "", url: "" });
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);
  const load = () => api("landing-pages").then(setList);
  useEffect(() => { load(); }, []);
  async function save(e: React.FormEvent) {
    e.preventDefault(); setMsg("");
    try { await api("landing-pages", { method: "POST", body: f }); setMsg("Saved."); load(); }
    catch (e: any) { setMsg(e.message); }
  }
  async function importUrl(e: React.FormEvent) {
    e.preventDefault(); setMsg(""); setBusy(true);
    try {
      const r: any = await api("landing-pages/import", { method: "POST", body: imp });
      setF({ ...f, name: r.name, html: r.html });
      setMsg("Imported into the editor below. Review and Save to keep it.");
      load();
    } catch (e: any) {
      setMsg("Import failed: " + e.message);
    } finally { setBusy(false); }
  }
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <div className="space-y-4">
        <form onSubmit={importUrl} className="card space-y-2">
          <div className="section-title">Clone a page from URL</div>
          <div className="flex gap-2">
            <input className="input" placeholder="example.com/login" value={imp.url} onChange={(e) => setImp({ ...imp, url: e.target.value })} />
            <button className="btn" disabled={busy}>{busy ? "Fetching…" : "Import"}</button>
          </div>
          <p className="text-xs muted">Fetches the page HTML into the editor (a &lt;base&gt; tag is added so styles/images load). Some sites block automated fetches — if so, paste the HTML manually.</p>
        </form>
        <form onSubmit={save} className="card space-y-3">
          <div className="section-title">Landing page</div>
          <p className="text-xs muted">Form must POST to {"{{.SubmitURL}}"}.</p>
          <input className="input" placeholder="Name" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required />
          <textarea className="input h-40 font-mono text-xs" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
          <input className="input" placeholder="Awareness redirect URL after submit (optional — else auto-training)" value={f.redirect_url} onChange={(e) => setF({ ...f, redirect_url: e.target.value })} />

          <div className="rounded-lg border p-3 text-sm" style={{ borderColor: "var(--pf-border)", background: "#fafbff" }}>
            <div className="mb-2 font-semibold">Capture settings</div>
            <label className="checkbox-row"><input type="checkbox" checked={f.capture_meta} onChange={(e) => setF({ ...f, capture_meta: e.target.checked })} /> Capture which field names were filled</label>
            <label className="checkbox-row mt-1"><input type="checkbox" checked={f.capture_submitted_data} onChange={(e) => setF({ ...f, capture_submitted_data: e.target.checked })} /> Capture submitted form values</label>
            <label className="checkbox-row mt-1"><input type="checkbox" checked={f.capture_passwords} onChange={(e) => setF({ ...f, capture_passwords: e.target.checked })} /> Also capture password fields</label>
            {f.capture_passwords && (
              <p className="mt-2 rounded px-2 py-1 text-xs" style={{ background: "#fee2e2", color: "#991b1b" }}>
                ⚠ Capturing passwords stores sensitive data. Only enable with explicit client authorization; handle and purge per your engagement rules.
              </p>
            )}
          </div>

          <button className="btn">Save landing page</button>
          {msg && <div className="text-sm" style={{ color: "#166534" }}>{msg}</div>}
        </form>
      </div>
      <div className="space-y-4">
        <Preview html={f.html} />
        <div className="card">
          <div className="section-title mb-2">Existing ({list.length})</div>
          <ul className="text-sm">{list.map((l) => (
            <li key={l.id} className="flex items-center justify-between border-b py-1" style={{ borderColor: "#eef1f7" }}>
              <span>{l.name}</span>
              <span className="flex gap-1">
                {l.capture_submitted_data && <span className="badge badge-amber">captures data</span>}
                {l.capture_passwords && <span className="badge badge-red">captures pw</span>}
              </span>
            </li>
          ))}</ul>
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
    e.preventDefault(); setMsg("");
    try { await api("sending-profiles", { method: "POST", body: f }); setMsg("Saved."); load(); }
    catch (e: any) { setMsg(e.message); }
  }
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <form onSubmit={save} className="card grid gap-3 sm:grid-cols-2">
        <div className="col-span-2 section-title">New sending profile</div>
        <input className="input" placeholder="Name" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required />
        <input className="input" placeholder="From address" value={f.from_address} onChange={(e) => setF({ ...f, from_address: e.target.value })} required />
        <input className="input" placeholder="SMTP host" value={f.smtp_host} onChange={(e) => setF({ ...f, smtp_host: e.target.value })} required />
        <input className="input" type="number" placeholder="Port" value={f.smtp_port} onChange={(e) => setF({ ...f, smtp_port: +e.target.value })} />
        <input className="input" placeholder="Username" value={f.username} onChange={(e) => setF({ ...f, username: e.target.value })} />
        <input className="input" type="password" placeholder="Password" value={f.password} onChange={(e) => setF({ ...f, password: e.target.value })} />
        <input className="input" placeholder="From name" value={f.from_name} onChange={(e) => setF({ ...f, from_name: e.target.value })} />
        <label className="checkbox-row"><input type="checkbox" checked={f.use_tls} onChange={(e) => setF({ ...f, use_tls: e.target.checked })} /> STARTTLS</label>
        <div className="col-span-2"><button className="btn">Save profile</button></div>
        {msg && <div className="col-span-2 text-sm" style={{ color: "#166534" }}>{msg}</div>}
      </form>
      <div className="card">
        <div className="section-title mb-2">Existing ({list.length})</div>
        <ul className="text-sm">{list.map((p) => <li key={p.id} className="border-b py-1" style={{ borderColor: "#eef1f7" }}>{p.name} — <span className="muted">{p.from_address}</span></li>)}</ul>
      </div>
    </div>
  );
}

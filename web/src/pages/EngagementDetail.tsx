import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { api } from "../api";
import { StatusBadge } from "./Engagements";

export default function EngagementDetail() {
  const { id } = useParams();
  const [eng, setEng] = useState<any>(null);
  const [scope, setScope] = useState<any[]>([]);
  const [targets, setTargets] = useState<any[]>([]);
  const [campaigns, setCampaigns] = useState<any[]>([]);
  const [risk, setRisk] = useState<any[]>([]);
  const [assets, setAssets] = useState<{ templates: any[]; landing: any[]; profiles: any[] }>({
    templates: [], landing: [], profiles: [],
  });
  const [msg, setMsg] = useState("");

  async function loadAll() {
    const [e, s, t, c, rk, tpl, lp, sp] = await Promise.all([
      api(`engagements/${id}`), api(`engagements/${id}/scope`), api(`engagements/${id}/targets`),
      api(`engagements/${id}/campaigns`), api(`engagements/${id}/risk`),
      api("email-templates"), api("landing-pages"), api("sending-profiles"),
    ]);
    setEng(e); setScope(s); setTargets(t); setCampaigns(c); setRisk(rk);
    setAssets({ templates: tpl, landing: lp, profiles: sp });
  }
  useEffect(() => { loadAll().catch((e) => setMsg(e.message)); }, [id]);

  const [rule, setRule] = useState({ kind: "domain", pattern: "" });
  async function addRule(e: React.FormEvent) {
    e.preventDefault(); setMsg("");
    try { await api(`engagements/${id}/scope`, { method: "POST", body: rule }); setRule({ kind: "domain", pattern: "" }); loadAll(); }
    catch (e: any) { setMsg(e.message); }
  }
  async function setStatus(status: string) {
    setMsg("");
    try { await api(`engagements/${id}/status`, { method: "POST", body: { status } }); loadAll(); }
    catch (e: any) { setMsg(e.message); }
  }

  const [bulk, setBulk] = useState("");
  async function importTargets(e: React.FormEvent) {
    e.preventDefault(); setMsg("");
    const rows = bulk.split("\n").map((l) => l.trim()).filter(Boolean).map((l) => {
      const [email, first, last, tz] = l.split(",").map((x) => (x || "").trim());
      return { email, first_name: first || "", last_name: last || "", timezone: tz || "" };
    });
    try {
      const res: any = await api(`engagements/${id}/targets`, { method: "POST", body: { targets: rows } });
      setBulk("");
      if (res.rejected_out_of_scope?.length) setMsg(`Rejected (out of scope): ${res.rejected_out_of_scope.join(", ")}`);
      loadAll();
    } catch (e: any) { setMsg(e.message); }
  }

  const [camp, setCamp] = useState<any>({
    name: "", email_template_id: "", landing_page_id: "", sending_profile_id: "",
    rate_per_minute: 30, launch_at: "", send_window_start: 0, send_window_end: 24,
    business_days_only: false, jitter_seconds: 0, warmup_batch: 0, rewrite_links: true,
  });
  async function createCampaign(e: React.FormEvent) {
    e.preventDefault(); setMsg("");
    const body: any = { ...camp };
    body.launch_at = camp.launch_at ? new Date(camp.launch_at).toISOString() : null;
    try {
      const res: any = await api(`engagements/${id}/campaigns`, { method: "POST", body });
      setMsg(`Campaign created: ${res.targets_added} targets queued, ${res.skipped} skipped.`);
      loadAll();
    } catch (e: any) { setMsg(e.message); }
  }
  async function launch(cid: string) {
    setMsg("");
    try { const r: any = await api(`campaigns/${cid}/launch`, { method: "POST" }); setMsg(`Campaign ${r.status}.`); loadAll(); }
    catch (e: any) { setMsg(e.message); }
  }
  async function stopCampaign(cid: string) {
    setMsg("");
    try { await api(`campaigns/${cid}/stop`, { method: "POST" }); setMsg("Campaign stopped."); loadAll(); }
    catch (e: any) { setMsg(e.message); }
  }
  async function deleteCampaign(cid: string, name: string) {
    if (!confirm(`Delete campaign "${name}"? This removes its targets and events.`)) return;
    setMsg("");
    try { await api(`campaigns/${cid}`, { method: "DELETE" }); setMsg("Campaign deleted."); loadAll(); }
    catch (e: any) { setMsg(e.message); }
  }

  if (!eng) return <div className="muted">Loading…</div>;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-3">
        <Link to="/engagements" className="muted hover:underline">← Engagements</Link>
        <h1 className="text-2xl font-bold">{eng.client_name}</h1>
        <StatusBadge status={eng.status} />
        <div className="ml-auto flex gap-2">
          {eng.status !== "active" && <button className="btn" onClick={() => setStatus("active")}>Activate</button>}
          {eng.status === "active" && <button className="btn-ghost" onClick={() => setStatus("closed")}>Close</button>}
        </div>
      </div>
      <p className="text-sm muted">
        Authorization: <span style={{ color: "var(--pf-text)" }}>{eng.authz_ref}</span> · window{" "}
        {new Date(eng.starts_at).toLocaleDateString()} → {new Date(eng.ends_at).toLocaleDateString()}
      </p>
      {msg && <div className="rounded-lg px-3 py-2 text-sm" style={{ background: "#fef3c7", color: "#92400e" }}>{msg}</div>}

      <div className="grid gap-6 lg:grid-cols-2">
        <div className="card space-y-3">
          <div className="section-title">Scope (allowlist)</div>
          <p className="text-xs muted">Only targets matching a rule can be contacted. Activation requires at least one rule.</p>
          <form onSubmit={addRule} className="flex gap-2">
            <select className="input max-w-[120px]" value={rule.kind} onChange={(e) => setRule({ ...rule, kind: e.target.value })}>
              <option value="domain">domain</option>
              <option value="email">email</option>
            </select>
            <input className="input" placeholder={rule.kind === "domain" ? "acme.com" : "vip-*@acme.com"} value={rule.pattern} onChange={(e) => setRule({ ...rule, pattern: e.target.value })} />
            <button className="btn">Add</button>
          </form>
          <ul className="space-y-1 text-sm">
            {scope.map((r) => (
              <li key={r.id} className="flex items-center justify-between rounded-lg px-3 py-1.5" style={{ background: "#f8fafc" }}>
                <span><span className="badge badge-gray">{r.kind}</span> {r.pattern}</span>
              </li>
            ))}
            {scope.length === 0 && <li className="muted">No rules — engagement cannot be activated.</li>}
          </ul>
        </div>

        <div className="card space-y-3">
          <div className="section-title">Targets ({targets.length})</div>
          <form onSubmit={importTargets} className="space-y-2">
            <textarea className="input h-24 font-mono text-xs" placeholder="email,First,Last,Timezone (one per line)" value={bulk} onChange={(e) => setBulk(e.target.value)} />
            <button className="btn">Import (scope-checked)</button>
          </form>
          <div className="max-h-40 overflow-y-auto text-sm">
            {targets.map((t) => (
              <div key={t.id} className="flex justify-between border-b py-1" style={{ borderColor: "#eef1f7" }}>
                <span>{t.email}</span>
                <span className="muted">{t.timezone}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Campaign builder with advanced controls */}
      <div className="card space-y-4">
        <div className="section-title">New campaign</div>
        <form onSubmit={createCampaign} className="grid gap-3 sm:grid-cols-3">
          <Field label="Name"><input className="input" value={camp.name} onChange={(e) => setCamp({ ...camp, name: e.target.value })} required /></Field>
          <Select label="Email template" value={camp.email_template_id} onChange={(v) => setCamp({ ...camp, email_template_id: v })} options={assets.templates} />
          <Select label="Landing page" value={camp.landing_page_id} onChange={(v) => setCamp({ ...camp, landing_page_id: v })} options={assets.landing} />
          <Select label="Sending profile" value={camp.sending_profile_id} onChange={(v) => setCamp({ ...camp, sending_profile_id: v })} options={assets.profiles} />
          <Field label="Rate / minute"><input className="input" type="number" value={camp.rate_per_minute} onChange={(e) => setCamp({ ...camp, rate_per_minute: +e.target.value })} /></Field>
          <Field label="Schedule (optional)"><input className="input" type="datetime-local" value={camp.launch_at} onChange={(e) => setCamp({ ...camp, launch_at: e.target.value })} /></Field>
          <Field label="Send window start (h)"><input className="input" type="number" min={0} max={23} value={camp.send_window_start} onChange={(e) => setCamp({ ...camp, send_window_start: +e.target.value })} /></Field>
          <Field label="Send window end (h)"><input className="input" type="number" min={1} max={24} value={camp.send_window_end} onChange={(e) => setCamp({ ...camp, send_window_end: +e.target.value })} /></Field>
          <Field label="Warm-up per cycle (0=∞)"><input className="input" type="number" min={0} value={camp.warmup_batch} onChange={(e) => setCamp({ ...camp, warmup_batch: +e.target.value })} /></Field>
          <Field label="Jitter (seconds)"><input className="input" type="number" min={0} value={camp.jitter_seconds} onChange={(e) => setCamp({ ...camp, jitter_seconds: +e.target.value })} /></Field>
          <label className="checkbox-row mt-6"><input type="checkbox" checked={camp.business_days_only} onChange={(e) => setCamp({ ...camp, business_days_only: e.target.checked })} /> Business days only</label>
          <label className="checkbox-row mt-6"><input type="checkbox" checked={camp.rewrite_links} onChange={(e) => setCamp({ ...camp, rewrite_links: e.target.checked })} /> Auto-rewrite links for tracking</label>
          <div className="sm:col-span-3"><button className="btn">Create campaign</button> <span className="ml-2 text-xs muted">Send window &amp; timezone are evaluated per recipient. Add A/B variants from the report page.</span></div>
        </form>

        <table className="data">
          <thead><tr><th>Name</th><th>Status</th><th>Window</th><th></th></tr></thead>
          <tbody>
            {campaigns.map((c) => (
              <tr key={c.id}>
                <td className="font-semibold">{c.name}</td>
                <td><StatusBadge status={c.status} /></td>
                <td className="muted text-xs">{c.send_window_start}:00–{c.send_window_end}:00{c.business_days_only ? " · biz-days" : ""}</td>
                <td className="flex flex-wrap gap-2 py-2">
                  {["draft", "scheduled", "stopped"].includes(c.status) && <button className="btn btn-sm" onClick={() => launch(c.id)}>Launch</button>}
                  {["running", "scheduled"].includes(c.status) && <button className="btn-ghost btn-sm" onClick={() => stopCampaign(c.id)}>Stop</button>}
                  <Link className="btn-ghost btn-sm" to={`/campaigns/${c.id}`}>Report</Link>
                  <button className="btn-danger btn-sm" onClick={() => deleteCampaign(c.id, c.name)}>Delete</button>
                </td>
              </tr>
            ))}
            {campaigns.length === 0 && <tr><td colSpan={4} className="text-center muted">No campaigns yet.</td></tr>}
          </tbody>
        </table>
      </div>

      {/* Risk scoring */}
      <div className="card">
        <div className="section-title mb-3">User risk scores</div>
        <table className="data">
          <thead><tr><th>Target</th><th>Opens</th><th>Clicks</th><th>Submits</th><th>Reports</th><th>Score</th><th>Level</th></tr></thead>
          <tbody>
            {risk.map((r, i) => (
              <tr key={i}>
                <td>{r.email}</td><td>{r.opens}</td><td>{r.clicks}</td><td>{r.submits}</td><td>{r.reports}</td>
                <td className="font-semibold">{r.score}</td>
                <td><span className={`badge ${r.level === "high" ? "badge-red" : r.level === "medium" ? "badge-amber" : "badge-green"}`}>{r.level}</span></td>
              </tr>
            ))}
            {risk.length === 0 && <tr><td colSpan={7} className="text-center muted">No activity yet.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: any }) {
  return <div><label className="label">{label}</label>{children}</div>;
}
function Select({ label, value, onChange, options }: { label: string; value: string; onChange: (v: string) => void; options: any[] }) {
  return (
    <div>
      <label className="label">{label}</label>
      <select className="input" value={value} onChange={(e) => onChange(e.target.value)} required>
        <option value="">— select —</option>
        {options.map((o) => <option key={o.id} value={o.id}>{o.name}</option>)}
      </select>
    </div>
  );
}

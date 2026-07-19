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
  const [assets, setAssets] = useState<{ templates: any[]; landing: any[]; profiles: any[] }>({
    templates: [],
    landing: [],
    profiles: [],
  });
  const [msg, setMsg] = useState("");

  async function loadAll() {
    const [e, s, t, c, tpl, lp, sp] = await Promise.all([
      api(`engagements/${id}`),
      api(`engagements/${id}/scope`),
      api(`engagements/${id}/targets`),
      api(`engagements/${id}/campaigns`),
      api("email-templates"),
      api("landing-pages"),
      api("sending-profiles"),
    ]);
    setEng(e);
    setScope(s);
    setTargets(t);
    setCampaigns(c);
    setAssets({ templates: tpl, landing: lp, profiles: sp });
  }
  useEffect(() => {
    loadAll().catch((e) => setMsg(e.message));
  }, [id]);

  // ---- scope ----
  const [rule, setRule] = useState({ kind: "domain", pattern: "" });
  async function addRule(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await api(`engagements/${id}/scope`, { method: "POST", body: rule });
      setRule({ kind: "domain", pattern: "" });
      loadAll();
    } catch (e: any) {
      setMsg(e.message);
    }
  }

  // ---- status ----
  async function setStatus(status: string) {
    setMsg("");
    try {
      await api(`engagements/${id}/status`, { method: "POST", body: { status } });
      loadAll();
    } catch (e: any) {
      setMsg(e.message);
    }
  }

  // ---- targets ----
  const [bulk, setBulk] = useState("");
  async function importTargets(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    const rows = bulk
      .split("\n")
      .map((l) => l.trim())
      .filter(Boolean)
      .map((l) => {
        const [email, first, last] = l.split(",").map((x) => (x || "").trim());
        return { email, first_name: first || "", last_name: last || "" };
      });
    try {
      const res: any = await api(`engagements/${id}/targets`, { method: "POST", body: { targets: rows } });
      setBulk("");
      if (res.rejected_out_of_scope?.length) {
        setMsg(`Rejected (out of scope): ${res.rejected_out_of_scope.join(", ")}`);
      }
      loadAll();
    } catch (e: any) {
      setMsg(e.message);
    }
  }

  // ---- campaign ----
  const [camp, setCamp] = useState({ name: "", email_template_id: "", landing_page_id: "", sending_profile_id: "", rate_per_minute: 30 });
  async function createCampaign(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      const res: any = await api(`engagements/${id}/campaigns`, { method: "POST", body: camp });
      setMsg(`Campaign created: ${res.targets_added} targets queued, ${res.skipped} skipped.`);
      loadAll();
    } catch (e: any) {
      setMsg(e.message);
    }
  }
  async function launch(cid: string) {
    setMsg("");
    try {
      await api(`campaigns/${cid}/launch`, { method: "POST" });
      setMsg("Campaign scheduled — worker will send it.");
      loadAll();
    } catch (e: any) {
      setMsg(e.message);
    }
  }

  if (!eng) return <div className="text-slate-400">Loading…</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link to="/engagements" className="text-slate-400 hover:text-slate-200">
          ← Engagements
        </Link>
        <h1 className="text-xl font-semibold">{eng.client_name}</h1>
        <StatusBadge status={eng.status} />
        <div className="ml-auto flex gap-2">
          {eng.status !== "active" && (
            <button className="btn" onClick={() => setStatus("active")}>
              Activate
            </button>
          )}
          {eng.status === "active" && (
            <button className="btn-ghost" onClick={() => setStatus("closed")}>
              Close
            </button>
          )}
        </div>
      </div>
      <p className="text-sm text-slate-400">
        Authorization: <span className="text-slate-200">{eng.authz_ref}</span> · window{" "}
        {new Date(eng.starts_at).toLocaleDateString()} → {new Date(eng.ends_at).toLocaleDateString()}
      </p>
      {msg && <div className="rounded bg-slate-800 px-3 py-2 text-sm text-amber-200">{msg}</div>}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Scope */}
        <div className="card space-y-3">
          <h2 className="font-medium">Scope (allowlist)</h2>
          <p className="text-xs text-slate-400">
            Only targets matching a rule can be contacted. Activation requires at least one rule.
          </p>
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
              <li key={r.id} className="flex items-center justify-between rounded bg-slate-800/50 px-3 py-1.5">
                <span>
                  <span className="badge bg-slate-700 text-slate-200">{r.kind}</span> {r.pattern}
                </span>
              </li>
            ))}
            {scope.length === 0 && <li className="text-slate-500">No rules — engagement cannot be activated.</li>}
          </ul>
        </div>

        {/* Targets */}
        <div className="card space-y-3">
          <h2 className="font-medium">Targets ({targets.length})</h2>
          <form onSubmit={importTargets} className="space-y-2">
            <textarea className="input h-24 font-mono text-xs" placeholder="email,First,Last (one per line)" value={bulk} onChange={(e) => setBulk(e.target.value)} />
            <button className="btn">Import (scope-checked)</button>
          </form>
          <div className="max-h-40 overflow-y-auto text-sm">
            {targets.map((t) => (
              <div key={t.id} className="flex justify-between border-b border-slate-800/50 py-1">
                <span>{t.email}</span>
                <span className="text-slate-500">{[t.first_name, t.last_name].filter(Boolean).join(" ")}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Campaigns */}
      <div className="card space-y-4">
        <h2 className="font-medium">Campaigns</h2>
        <form onSubmit={createCampaign} className="grid gap-3 sm:grid-cols-2">
          <div>
            <label className="label">Name</label>
            <input className="input" value={camp.name} onChange={(e) => setCamp({ ...camp, name: e.target.value })} required />
          </div>
          <div>
            <label className="label">Rate / minute</label>
            <input className="input" type="number" value={camp.rate_per_minute} onChange={(e) => setCamp({ ...camp, rate_per_minute: Number(e.target.value) })} />
          </div>
          <Select label="Email template" value={camp.email_template_id} onChange={(v) => setCamp({ ...camp, email_template_id: v })} options={assets.templates} />
          <Select label="Landing page" value={camp.landing_page_id} onChange={(v) => setCamp({ ...camp, landing_page_id: v })} options={assets.landing} />
          <Select label="Sending profile" value={camp.sending_profile_id} onChange={(v) => setCamp({ ...camp, sending_profile_id: v })} options={assets.profiles} />
          <div className="flex items-end">
            <button className="btn">Create campaign</button>
          </div>
        </form>

        <table className="data">
          <thead>
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {campaigns.map((c) => (
              <tr key={c.id}>
                <td>{c.name}</td>
                <td>
                  <StatusBadge status={c.status} />
                </td>
                <td className="flex gap-2">
                  {["draft", "scheduled"].includes(c.status) && (
                    <button className="btn" onClick={() => launch(c.id)}>
                      Launch
                    </button>
                  )}
                  <Link className="btn-ghost" to={`/campaigns/${c.id}`}>
                    Report
                  </Link>
                </td>
              </tr>
            ))}
            {campaigns.length === 0 && (
              <tr>
                <td colSpan={3} className="text-center text-slate-500">
                  No campaigns yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function Select({ label, value, onChange, options }: { label: string; value: string; onChange: (v: string) => void; options: any[] }) {
  return (
    <div>
      <label className="label">{label}</label>
      <select className="input" value={value} onChange={(e) => onChange(e.target.value)} required>
        <option value="">— select —</option>
        {options.map((o) => (
          <option key={o.id} value={o.id}>
            {o.name}
          </option>
        ))}
      </select>
    </div>
  );
}

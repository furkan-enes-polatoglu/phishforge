import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { api, getAccess } from "../api";
import { StatusBadge } from "./Engagements";
import { useI18n } from "../i18n";

export default function EngagementDetail() {
  const { t } = useI18n();
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

  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  async function importFile(e: React.FormEvent) {
    e.preventDefault();
    if (!file) return;
    setMsg(""); setUploading(true);
    try {
      const fd = new FormData();
      fd.append("file", file);
      const res = await fetch(`/api/engagements/${id}/targets/import`, {
        method: "POST",
        headers: { Authorization: `Bearer ${getAccess()}` },
        body: fd,
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "import failed");
      let m = `${data.created?.length ?? 0} ${t("targets_added")}.`;
      if (data.rejected_out_of_scope?.length) m += ` ${data.rejected_out_of_scope.length} ${t("rejected_scope")}.`;
      if (data.parse_errors?.length) m += ` ${data.parse_errors.length} ${t("parse_errors")}.`;
      setMsg(m);
      setFile(null);
      loadAll();
    } catch (e: any) { setMsg(e.message); }
    finally { setUploading(false); }
  }
  function downloadTemplate() {
    const csv = "Ad Soyad,E-posta,Departman,Pozisyon,Saat Dilimi,VIP\nAyşe Yılmaz,ayse.yilmaz@ornek.com,Finans,Uzman,Europe/Istanbul,\nMehmet Demir,mehmet.demir@ornek.com,Yönetim,Genel Müdür,Europe/Istanbul,evet\n";
    const blob = new Blob(["﻿" + csv], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url; a.download = "phishforge-hedef-sablonu.csv"; a.click();
    URL.revokeObjectURL(url);
  }

  const [camp, setCamp] = useState<any>({
    name: "", email_template_id: "", landing_page_id: "", sending_profile_id: "",
    rate_per_minute: 30, launch_at: "", send_window_start: 0, send_window_end: 24,
    business_days_only: false, jitter_seconds: 0, warmup_batch: 0, rewrite_links: true,
    spoofed_from_name: "", spoofed_from_address: "", reply_to: "", landing_base_url: "",
  });
  // When a sending profile is picked, pre-fill the campaign's landing URL from
  // that profile's own domain — GoPhish-style "URL" field, except pre-filled
  // instead of retyped every launch. The operator can still edit it freely.
  function pickSendingProfile(id: string) {
    const profile = assets.profiles.find((p: any) => p.id === id);
    setCamp((c: any) => ({ ...c, sending_profile_id: id, landing_base_url: profile?.landing_base_url || "" }));
  }
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
        <Link to="/engagements" className="muted hover:underline">← {t("engagements")}</Link>
        <h1 className="text-2xl font-bold">{eng.client_name}</h1>
        <StatusBadge status={eng.status} />
        <div className="ml-auto flex gap-2">
          {eng.status !== "active" && <button className="btn" onClick={() => setStatus("active")}>{t("activate")}</button>}
          {eng.status === "active" && <button className="btn-ghost" onClick={() => setStatus("closed")}>{t("close_")}</button>}
        </div>
      </div>
      <p className="text-sm muted">
        {t("authorization")}: <span style={{ color: "var(--pf-text)" }}>{eng.authz_ref}</span> · {t("window").toLowerCase()}{" "}
        {new Date(eng.starts_at).toLocaleDateString()} → {new Date(eng.ends_at).toLocaleDateString()}
      </p>
      {msg && <div className="rounded-lg px-3 py-2 text-sm" style={{ background: "#fef3c7", color: "#92400e" }}>{msg}</div>}

      <div className="grid gap-6 lg:grid-cols-2">
        <div className="card space-y-3">
          <div className="section-title">{t("scope_allowlist")}</div>
          <p className="text-xs muted">{t("scope_help")}</p>
          <form onSubmit={addRule} className="flex gap-2">
            <select className="input max-w-[120px]" value={rule.kind} onChange={(e) => setRule({ ...rule, kind: e.target.value })}>
              <option value="domain">domain</option>
              <option value="email">email</option>
            </select>
            <input className="input" placeholder={rule.kind === "domain" ? "acme.com" : "vip-*@acme.com"} value={rule.pattern} onChange={(e) => setRule({ ...rule, pattern: e.target.value })} />
            <button className="btn">{t("add")}</button>
          </form>
          <ul className="space-y-1 text-sm">
            {scope.map((r) => (
              <li key={r.id} className="flex items-center justify-between rounded-lg px-3 py-1.5" style={{ background: "#f8fafc" }}>
                <span><span className="badge badge-gray">{r.kind}</span> {r.pattern}</span>
              </li>
            ))}
            {scope.length === 0 && <li className="muted">{t("no_rules")}</li>}
          </ul>
        </div>

        <div className="card space-y-4">
          <div className="section-title">{t("targets")} ({targets.length})</div>

          <div className="space-y-2 rounded-lg border p-3" style={{ borderColor: "var(--pf-border)", background: "#fafbff" }}>
            <div className="text-sm font-semibold">{t("import_from_file")}</div>
            <p className="text-xs muted">{t("import_file_help")}</p>
            <button type="button" className="btn-ghost btn-sm" onClick={downloadTemplate}>{t("download_template")}</button>
            <form onSubmit={importFile} className="flex flex-wrap gap-2">
              <input type="file" accept=".csv,.xlsx" onChange={(e) => setFile(e.target.files?.[0] || null)} className="input" style={{ maxWidth: 220 }} />
              <button className="btn btn-sm" disabled={!file || uploading}>{uploading ? t("fetching") : t("upload_and_import")}</button>
            </form>
          </div>

          <form onSubmit={importTargets} className="space-y-2">
            <textarea className="input h-20 font-mono text-xs" placeholder="email,Ad,Soyad,ZamanDilimi (satır başına bir)" value={bulk} onChange={(e) => setBulk(e.target.value)} />
            <button className="btn-ghost btn-sm">{t("import_scope_checked")}</button>
          </form>

          <div className="max-h-40 overflow-y-auto text-sm">
            {targets.map((tg) => (
              <div key={tg.id} className="flex items-center justify-between border-b py-1" style={{ borderColor: "#eef1f7" }}>
                <span>{tg.email} {tg.is_vip && <span className="badge badge-amber">VIP</span>}</span>
                <span className="muted">{tg.department}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Campaign builder with advanced controls */}
      <div className="card space-y-4">
        <div className="section-title">{t("new_campaign")}</div>
        <form onSubmit={createCampaign} className="grid gap-3 sm:grid-cols-3">
          <Field label={t("name")}><input className="input" value={camp.name} onChange={(e) => setCamp({ ...camp, name: e.target.value })} required /></Field>
          <Select label={t("email_template")} value={camp.email_template_id} onChange={(v) => setCamp({ ...camp, email_template_id: v })} options={assets.templates} />
          <Select label={t("landing_page")} value={camp.landing_page_id} onChange={(v) => setCamp({ ...camp, landing_page_id: v })} options={assets.landing} />
          <Select label={t("sending_profile")} value={camp.sending_profile_id} onChange={pickSendingProfile} options={assets.profiles} />
          <div className="sm:col-span-3">
            <label className="label">{t("campaign_url")}</label>
            <input className="input" placeholder="https://portal.musteri-domaini.com" value={camp.landing_base_url} onChange={(e) => setCamp({ ...camp, landing_base_url: e.target.value })} />
            <p className="mt-1 text-xs muted">{t("campaign_url_help")}</p>
          </div>
          <Field label={t("rate_per_min")}><input className="input" type="number" value={camp.rate_per_minute} onChange={(e) => setCamp({ ...camp, rate_per_minute: +e.target.value })} /></Field>
          <Field label={t("schedule_optional")}><input className="input" type="datetime-local" value={camp.launch_at} onChange={(e) => setCamp({ ...camp, launch_at: e.target.value })} /></Field>
          <Field label={t("window_start")}><input className="input" type="number" min={0} max={23} value={camp.send_window_start} onChange={(e) => setCamp({ ...camp, send_window_start: +e.target.value })} /></Field>
          <Field label={t("window_end")}><input className="input" type="number" min={1} max={24} value={camp.send_window_end} onChange={(e) => setCamp({ ...camp, send_window_end: +e.target.value })} /></Field>
          <Field label={t("warmup")}><input className="input" type="number" min={0} value={camp.warmup_batch} onChange={(e) => setCamp({ ...camp, warmup_batch: +e.target.value })} /></Field>
          <Field label={t("jitter")}><input className="input" type="number" min={0} value={camp.jitter_seconds} onChange={(e) => setCamp({ ...camp, jitter_seconds: +e.target.value })} /></Field>
          <label className="checkbox-row mt-6"><input type="checkbox" checked={camp.business_days_only} onChange={(e) => setCamp({ ...camp, business_days_only: e.target.checked })} /> {t("business_days")}</label>
          <label className="checkbox-row mt-6"><input type="checkbox" checked={camp.rewrite_links} onChange={(e) => setCamp({ ...camp, rewrite_links: e.target.checked })} /> {t("rewrite_links")}</label>

          <div className="sm:col-span-3 rounded-lg border p-3" style={{ borderColor: "var(--pf-border)", background: "#fafbff" }}>
            <div className="mb-1 font-semibold text-sm">{t("spoofed_from")}</div>
            <p className="mb-2 text-xs muted">{t("spoofed_from_help")}</p>
            <div className="grid gap-2 sm:grid-cols-3">
              <input className="input" placeholder={t("spoofed_from_name") + " (opsiyonel)"} value={camp.spoofed_from_name} onChange={(e) => setCamp({ ...camp, spoofed_from_name: e.target.value })} />
              <input className="input" placeholder={t("spoofed_from_address") + " (opsiyonel)"} value={camp.spoofed_from_address} onChange={(e) => setCamp({ ...camp, spoofed_from_address: e.target.value })} />
              <input className="input" placeholder={t("reply_to") + " (opsiyonel)"} value={camp.reply_to} onChange={(e) => setCamp({ ...camp, reply_to: e.target.value })} />
            </div>
            {(() => {
              const profile = assets.profiles.find((p: any) => p.id === camp.sending_profile_id);
              const spoofDomain = camp.spoofed_from_address.split("@")[1]?.toLowerCase();
              const profDomain = profile?.from_address?.split("@")[1]?.toLowerCase();
              if (spoofDomain && profDomain && spoofDomain !== profDomain) {
                return (
                  <p className="mt-2 rounded px-2 py-1 text-xs" style={{ background: "#fee2e2", color: "#991b1b" }}>
                    ⚠ {spoofDomain} ≠ {profDomain} — DMARC hizalaması hedefte başarısız olur. Teslimat sayfasındaki "Hedef Mail Ağ Geçidi Tespiti" ile bu gönderim altyapısını beyaz listeye aldırın.
                  </p>
                );
              }
              return null;
            })()}
          </div>

          <div className="sm:col-span-3"><button className="btn">{t("create")}</button></div>
        </form>

        <table className="data">
          <thead><tr><th>{t("name")}</th><th>{t("status")}</th><th>{t("window")}</th><th>{t("actions")}</th></tr></thead>
          <tbody>
            {campaigns.map((c) => (
              <tr key={c.id}>
                <td className="font-semibold">{c.name}</td>
                <td><StatusBadge status={c.status} /></td>
                <td className="muted text-xs">{c.send_window_start}:00–{c.send_window_end}:00{c.business_days_only ? " · biz-days" : ""}</td>
                <td className="flex flex-wrap gap-2 py-2">
                  {["draft", "scheduled", "stopped"].includes(c.status) && <button className="btn btn-sm" onClick={() => launch(c.id)}>{t("launch")}</button>}
                  {["running", "scheduled"].includes(c.status) && <button className="btn-ghost btn-sm" onClick={() => stopCampaign(c.id)}>{t("stop")}</button>}
                  <Link className="btn-ghost btn-sm" to={`/campaigns/${c.id}`}>{t("report")}</Link>
                  <button className="btn-danger btn-sm" onClick={() => deleteCampaign(c.id, c.name)}>{t("delete")}</button>
                </td>
              </tr>
            ))}
            {campaigns.length === 0 && <tr><td colSpan={4} className="text-center muted">{t("none_yet")}</td></tr>}
          </tbody>
        </table>
      </div>

      {/* Risk scoring */}
      <div className="card">
        <div className="section-title mb-3">{t("risk_scores")}</div>
        <table className="data">
          <thead><tr><th>{t("target")}</th><th>{t("department")}</th><th>{t("opens")}</th><th>{t("clicks")}</th><th>{t("submits")}</th><th>{t("reports")}</th><th>{t("score")}</th><th>{t("level")}</th></tr></thead>
          <tbody>
            {risk.map((r, i) => (
              <tr key={i}>
                <td>{r.email} {r.is_vip && <span className="badge badge-amber">VIP</span>}</td>
                <td className="muted">{r.department}</td>
                <td>{r.opens}</td><td>{r.clicks}</td><td>{r.submits}</td><td>{r.reports}</td>
                <td className="font-semibold">{r.score}</td>
                <td><span className={`badge ${r.level === "high" ? "badge-red" : r.level === "medium" ? "badge-amber" : "badge-green"}`}>{r.level}</span></td>
              </tr>
            ))}
            {risk.length === 0 && <tr><td colSpan={8} className="text-center muted">{t("none_yet")}</td></tr>}
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
  const { t } = useI18n();
  return (
    <div>
      <label className="label">{label}</label>
      <select className="input" value={value} onChange={(e) => onChange(e.target.value)} required>
        <option value="">{t("select")}</option>
        {options.map((o) => <option key={o.id} value={o.id}>{o.name}</option>)}
      </select>
    </div>
  );
}

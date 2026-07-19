import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { api } from "../api";
import { FunnelBars } from "../components/Funnel";
import { useI18n } from "../i18n";

export default function CampaignReport() {
  const { t } = useI18n();
  const { id } = useParams();
  const [report, setReport] = useState<any>(null);
  const [timeline, setTimeline] = useState<any[]>([]);
  const [variants, setVariants] = useState<any[]>([]);
  const [templates, setTemplates] = useState<any[]>([]);
  const [err, setErr] = useState("");
  const [nv, setNv] = useState({ name: "", email_template_id: "", weight: 1 });

  async function load() {
    try {
      const [r, t, v, tpl] = await Promise.all([
        api(`campaigns/${id}/report`), api(`campaigns/${id}/timeline`),
        api(`campaigns/${id}/variants`), api("email-templates"),
      ]);
      setReport(r); setTimeline(t); setVariants(v); setTemplates(tpl);
    } catch (e: any) { setErr(e.message); }
  }
  useEffect(() => {
    load();
    const iv = setInterval(load, 5000);
    return () => clearInterval(iv);
  }, [id]);

  async function addVariant(e: React.FormEvent) {
    e.preventDefault(); setErr("");
    try { await api(`campaigns/${id}/variants`, { method: "POST", body: nv }); setNv({ name: "", email_template_id: "", weight: 1 }); load(); }
    catch (e: any) { setErr(e.message); }
  }

  if (err) return <div style={{ color: "#b91c1c" }}>{err}</div>;
  if (!report) return <div className="muted">Loading…</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link to={`/engagements/${report.campaign.engagement_id}`} className="muted hover:underline">← {t("nav_engagements")}</Link>
        <h1 className="text-2xl font-bold">{report.campaign.name}</h1>
      </div>

      <div className="card">
        <div className="section-title mb-3">{t("funnel")}</div>
        <FunnelBars funnel={report.funnel} />
      </div>

      {/* A/B variants */}
      <div className="card space-y-3">
        <div className="section-title">{t("ab_variants")}</div>
        {report.variants?.length > 0 && (
          <table className="data">
            <thead><tr><th>{t("variant")}</th><th>{t("targets")}</th><th>{t("opens")}</th><th>{t("clicks")}</th><th>{t("submits")}</th><th>%</th></tr></thead>
            <tbody>
              {report.variants.map((v: any, i: number) => (
                <tr key={i}>
                  <td className="font-semibold">{v.variant}</td>
                  <td>{v.targets}</td><td>{v.opened}</td><td>{v.clicked}</td><td>{v.submitted}</td>
                  <td>{v.targets ? Math.round((v.clicked / v.targets) * 100) : 0}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        <form onSubmit={addVariant} className="flex flex-wrap items-end gap-2">
          <div><label className="label">{t("variant")}</label><input className="input" value={nv.name} onChange={(e) => setNv({ ...nv, name: e.target.value })} required /></div>
          <div><label className="label">{t("email_template")}</label>
            <select className="input" value={nv.email_template_id} onChange={(e) => setNv({ ...nv, email_template_id: e.target.value })} required>
              <option value="">{t("select")}</option>
              {templates.map((tp) => <option key={tp.id} value={tp.id}>{tp.name}</option>)}
            </select>
          </div>
          <div><label className="label">{t("weight")}</label><input className="input w-20" type="number" min={1} value={nv.weight} onChange={(e) => setNv({ ...nv, weight: +e.target.value })} /></div>
          <button className="btn">{t("add_variant")}</button>
        </form>
      </div>

      <div className="card">
        <div className="section-title mb-2">{t("timeline")}</div>
        <div className="overflow-x-auto">
          <table className="data">
            <thead><tr><th>{t("when")}</th><th>{t("target")}</th><th>{t("event")}</th><th>IP</th><th>{t("captured_data")}</th></tr></thead>
            <tbody>
              {timeline.map((ev, i) => (
                <tr key={i}>
                  <td className="muted">{new Date(ev.created_at).toLocaleString()}</td>
                  <td>{ev.email}</td>
                  <td><EventBadge type={ev.type} /></td>
                  <td className="muted">{ev.ip}</td>
                  <td className="font-mono text-xs">{renderCaptured(ev.meta)}</td>
                </tr>
              ))}
              {timeline.length === 0 && <tr><td colSpan={5} className="text-center muted">No events yet.</td></tr>}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function EventBadge({ type }: { type: string }) {
  const { t } = useI18n();
  const cls: Record<string, string> = {
    sent: "badge-gray", open: "badge-blue", click: "badge-amber", submit: "badge-red",
    report: "badge-green", scan: "badge-blue", attachment_open: "badge-red",
  };
  return <span className={`badge ${cls[type] || "badge-gray"}`}>{t(`event_${type}`)}</span>;
}

function renderCaptured(meta: any) {
  if (!meta) return "";
  if (meta.submitted && typeof meta.submitted === "object") {
    return Object.entries(meta.submitted).map(([k, v]) => (
      <div key={k}><span className="muted">{k}:</span> {String(v)}</div>
    ));
  }
  if (meta.fields_filled?.length) return <span className="muted">fields: {meta.fields_filled.join(", ")}</span>;
  return "";
}

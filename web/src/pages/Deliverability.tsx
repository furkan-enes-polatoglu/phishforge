import { useState } from "react";
import { api } from "../api";
import { useI18n } from "../i18n";

export default function Deliverability() {
  const { t } = useI18n();
  const [f, setF] = useState({ domain: "", dkim_selector: "", sender_ip: "", html: "" });
  const [res, setRes] = useState<any>(null);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function run(e: React.FormEvent) {
    e.preventDefault(); setErr(""); setBusy(true);
    try { setRes(await api("deliverability/check", { method: "POST", body: f })); }
    catch (e: any) { setErr(e.message); }
    finally { setBusy(false); }
  }

  const [seed, setSeed] = useState({ host: "", port: 993, username: "", password: "", use_tls: true, subject_marker: "" });
  const [seedRes, setSeedRes] = useState<any>(null);
  const [seedErr, setSeedErr] = useState("");
  const [seedBusy, setSeedBusy] = useState(false);
  async function runSeed(e: React.FormEvent) {
    e.preventDefault(); setSeedErr(""); setSeedBusy(true); setSeedRes(null);
    try { setSeedRes(await api("deliverability/seed-check", { method: "POST", body: seed })); }
    catch (e: any) { setSeedErr(e.message); }
    finally { setSeedBusy(false); }
  }

  const badge = (status: string) =>
    status === "ok" ? "badge-green" : status === "warn" ? "badge-amber" : "badge-red";
  const rec = (label: string, r: any) =>
    r && (
      <div className="flex items-center justify-between border-b py-2" style={{ borderColor: "#eef1f7" }}>
        <span>{label}</span>
        <span className={`badge ${badge(r.status)}`}>{r.status}</span>
      </div>
    );

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("deliverability")}</h1>
      <p className="text-sm muted">{t("deliverability_help")}</p>

      <form onSubmit={run} className="card grid gap-3 sm:grid-cols-2">
        <input className="input" placeholder={t("dkim_domain") + " (örn. mail.acme.com)"} value={f.domain} onChange={(e) => setF({ ...f, domain: e.target.value })} required />
        <input className="input" placeholder={t("dkim_selector")} value={f.dkim_selector} onChange={(e) => setF({ ...f, dkim_selector: e.target.value })} />
        <input className="input" placeholder="RBL için gönderen IPv4 (opsiyonel)" value={f.sender_ip} onChange={(e) => setF({ ...f, sender_ip: e.target.value })} />
        <textarea className="input h-24 font-mono text-xs sm:col-span-2" placeholder="Lint için e-posta HTML'i (opsiyonel)" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
        <div><button className="btn" disabled={busy}>{busy ? t("checking") : t("run_check")}</button></div>
      </form>

      {err && <div style={{ color: "#b91c1c" }}>{err}</div>}

      {res && (
        <div className="grid gap-6 lg:grid-cols-2">
          <div className="card">
            <div className="section-title mb-2">{t("authentication")} — {res.domain}</div>
            {rec("SPF", res.spf)}
            {rec("DMARC", res.dmarc)}
            {rec("DKIM", res.dkim)}
            {res.spam_score != null && (
              <div className="flex justify-between py-2"><span>SpamAssassin score</span><span className="badge badge-gray">{res.spam_score}</span></div>
            )}
            {res.rbl?.length > 0 && (
              <div className="mt-3">
                <div className="label">{t("blocklists")}</div>
                {res.rbl.map((b: any) => (
                  <div key={b.list} className="flex justify-between py-1 text-sm">
                    <span>{b.list}</span>
                    <span className={`badge ${b.listed ? "badge-red" : "badge-green"}`}>{b.listed ? "listed" : "clean"}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
          <div className="card space-y-3">
            <div>
              <div className="label">{t("advice")}</div>
              <ul className="list-disc space-y-1 pl-5 text-sm">{res.advice?.map((a: string, i: number) => <li key={i}>{a}</li>)}</ul>
            </div>
            {res.html_lint?.length > 0 && (
              <div>
                <div className="label">HTML lint</div>
                <ul className="list-disc space-y-1 pl-5 text-sm" style={{ color: "#92400e" }}>{res.html_lint.map((a: string, i: number) => <li key={i}>{a}</li>)}</ul>
              </div>
            )}
          </div>
        </div>
      )}

      <div className="card space-y-3">
        <div className="section-title">{t("seed_test")}</div>
        <p className="text-xs muted">{t("seed_test_help")}</p>
        <form onSubmit={runSeed} className="grid gap-3 sm:grid-cols-2">
          <input className="input" placeholder={t("imap_host") + " (örn. imap.gmail.com)"} value={seed.host} onChange={(e) => setSeed({ ...seed, host: e.target.value })} required />
          <input className="input" type="number" placeholder="Port (993)" value={seed.port} onChange={(e) => setSeed({ ...seed, port: +e.target.value })} />
          <input className="input" placeholder={t("username")} value={seed.username} onChange={(e) => setSeed({ ...seed, username: e.target.value })} required />
          <input className="input" type="password" placeholder={t("password")} value={seed.password} onChange={(e) => setSeed({ ...seed, password: e.target.value })} required />
          <input className="input sm:col-span-2" placeholder={t("subject_marker")} value={seed.subject_marker} onChange={(e) => setSeed({ ...seed, subject_marker: e.target.value })} required />
          <label className="checkbox-row"><input type="checkbox" checked={seed.use_tls} onChange={(e) => setSeed({ ...seed, use_tls: e.target.checked })} /> TLS</label>
          <div><button className="btn" disabled={seedBusy}>{seedBusy ? t("checking") : t("run_seed_check")}</button></div>
        </form>
        {seedErr && <div style={{ color: "#b91c1c" }}>{seedErr}</div>}
        {seedRes && (
          <div className="rounded-lg px-3 py-2 text-sm" style={{
            background: seedRes.found ? (seedRes.folder === "INBOX" ? "#dcfce7" : "#fee2e2") : "#f1f5f9",
            color: seedRes.found ? (seedRes.folder === "INBOX" ? "#166534" : "#991b1b") : "#475569",
          }}>
            {!seedRes.found ? t("seed_not_found") : seedRes.folder === "INBOX" ? `${t("seed_found_inbox")} (${seedRes.folder})` : `${t("seed_found_spam")} (${seedRes.folder})`}
          </div>
        )}
      </div>
    </div>
  );
}

import { useState } from "react";
import { api } from "../api";
import { useI18n } from "../i18n";

export default function Deliverability() {
  const { t } = useI18n();
  const [f, setF] = useState({ domain: "", dkim_selector: "", sender_ip: "", subject: "", html: "" });
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

  // ---- Target gateway detection (headline feature) ----
  const [gw, setGw] = useState({ target_domain: "", client_name: "", sending_domain: "", sending_ip: "", dkim_domain: "", dkim_selector: "" });
  const [gwRes, setGwRes] = useState<any>(null);
  const [gwErr, setGwErr] = useState("");
  const [gwBusy, setGwBusy] = useState(false);
  const [copied, setCopied] = useState(false);
  async function runGateway(e: React.FormEvent) {
    e.preventDefault(); setGwErr(""); setGwBusy(true); setGwRes(null); setCopied(false);
    try { setGwRes(await api("deliverability/gateway-check", { method: "POST", body: gw })); }
    catch (e: any) { setGwErr(e.message); }
    finally { setGwBusy(false); }
  }
  function copyEmail() {
    if (!gwRes?.cover_email) return;
    navigator.clipboard.writeText(gwRes.cover_email).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
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

      {/* Headline feature: target gateway detection + allowlist playbook */}
      <div className="card space-y-3" style={{ borderColor: "var(--pf-primary)", borderWidth: 2 }}>
        <div className="section-title">{t("gateway_detect")}</div>
        <p className="text-xs muted">{t("gateway_detect_help")}</p>
        <form onSubmit={runGateway} className="grid gap-3 sm:grid-cols-2">
          <input className="input" placeholder={t("target_domain") + " (örn. acme.com)"} value={gw.target_domain} onChange={(e) => setGw({ ...gw, target_domain: e.target.value })} required />
          <input className="input" placeholder={t("client_name")} value={gw.client_name} onChange={(e) => setGw({ ...gw, client_name: e.target.value })} />
          <input className="input" placeholder={t("from_address").replace("adresi", "alan adı") + " (sim.acme-test.com)"} value={gw.sending_domain} onChange={(e) => setGw({ ...gw, sending_domain: e.target.value })} />
          <input className="input" placeholder="Gönderen IP" value={gw.sending_ip} onChange={(e) => setGw({ ...gw, sending_ip: e.target.value })} />
          <input className="input" placeholder={t("dkim_domain")} value={gw.dkim_domain} onChange={(e) => setGw({ ...gw, dkim_domain: e.target.value })} />
          <input className="input" placeholder={t("dkim_selector")} value={gw.dkim_selector} onChange={(e) => setGw({ ...gw, dkim_selector: e.target.value })} />
          <div><button className="btn" disabled={gwBusy}>{gwBusy ? t("detecting") : t("detect_gateway")}</button></div>
        </form>
        {gwErr && <div style={{ color: "#b91c1c" }}>{gwErr}</div>}
        {gwRes && (
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="rounded-lg border p-3" style={{ borderColor: "var(--pf-border)" }}>
              <div className="mb-2 flex items-center gap-2">
                <span className="text-sm font-semibold">{t("detected_gateway")}:</span>
                {gwRes.provider ? (
                  <span className="badge badge-blue">{gwRes.provider.name}</span>
                ) : (
                  <span className="badge badge-gray">{t("gateway_unknown")}</span>
                )}
              </div>
              {gwRes.mx_hosts?.length > 0 && (
                <div className="mb-2 text-xs">
                  <div className="label">{t("mx_records")}</div>
                  {gwRes.mx_hosts.map((h: string, i: number) => <div key={i} className="font-mono">{h}</div>)}
                </div>
              )}
              {gwRes.provider?.steps && (
                <div className="text-xs">
                  <div className="label">{t("allowlist_steps")} — {gwRes.provider.feature_name}</div>
                  <ol className="list-decimal space-y-1 pl-4">
                    {gwRes.provider.steps.map((s: string, i: number) => <li key={i}>{s}</li>)}
                  </ol>
                </div>
              )}
            </div>
            <div className="rounded-lg border p-3" style={{ borderColor: "var(--pf-border)" }}>
              <div className="mb-2 flex items-center justify-between">
                <div className="label !mb-0">{t("cover_email")}</div>
                <button type="button" className="btn-ghost btn-sm" onClick={copyEmail}>{copied ? t("copied") : t("copy")}</button>
              </div>
              <textarea readOnly className="input h-64 font-mono text-xs" value={gwRes.cover_email} />
            </div>
          </div>
        )}
      </div>

      {/* Sender-side deliverability check */}
      <form onSubmit={run} className="card grid gap-3 sm:grid-cols-2">
        <div className="sm:col-span-2 section-title">Gönderen alan adı sağlığı</div>
        <input className="input" placeholder={t("dkim_domain") + " (örn. mail.acme.com)"} value={f.domain} onChange={(e) => setF({ ...f, domain: e.target.value })} required />
        <input className="input" placeholder={t("dkim_selector")} value={f.dkim_selector} onChange={(e) => setF({ ...f, dkim_selector: e.target.value })} />
        <input className="input" placeholder="RBL/PTR için gönderen IPv4 (opsiyonel)" value={f.sender_ip} onChange={(e) => setF({ ...f, sender_ip: e.target.value })} />
        <input className="input" placeholder={t("subject_line") + " (opsiyonel)"} value={f.subject} onChange={(e) => setF({ ...f, subject: e.target.value })} />
        <textarea className="input h-24 font-mono text-xs sm:col-span-2" placeholder="Lint için e-posta HTML'i (opsiyonel)" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
        <div><button className="btn" disabled={busy}>{busy ? t("checking") : t("run_check")}</button></div>
      </form>

      {err && <div style={{ color: "#b91c1c" }}>{err}</div>}

      {res && (
        <div className="space-y-4">
          <div className="card flex items-center gap-4">
            <div className="text-4xl font-bold" style={{ color: "var(--pf-primary)" }}>{res.score.score}<span className="text-lg muted">/100</span></div>
            <div>
              <span className={`badge ${res.score.grade === "A" || res.score.grade === "B" ? "badge-green" : res.score.grade === "C" ? "badge-amber" : "badge-red"}`} style={{ fontSize: 16, padding: "4px 12px" }}>{res.score.grade}</span>
              <div className="mt-1 text-xs muted">{t("delivery_score")}</div>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            <div className="card">
              <div className="section-title mb-2">{t("authentication")} — {res.domain}</div>
              {rec("SPF", res.spf)}
              {rec("DMARC", res.dmarc)}
              {rec("DKIM", res.dkim)}
              {res.ptr && rec(t("ptr_check"), res.ptr)}
              {res.mta_sts && rec(t("mta_sts"), res.mta_sts)}
              {res.dmarc_policy && (
                <div className="mt-3 text-xs">
                  <div className="label">{t("dmarc_policy")}</div>
                  <div className="font-mono">p={res.dmarc_policy.policy || "—"} sp={res.dmarc_policy.sub_policy || "—"} aspf={res.dmarc_policy.align_spf} adkim={res.dmarc_policy.align_dkim}</div>
                </div>
              )}
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
              {res.content && (
                <div>
                  <div className="label">{t("content_analysis")}</div>
                  <div className="text-sm">
                    {res.content.trigger_words_found?.length > 0 && <div>{t("trigger_words")}: <span className="font-mono">{res.content.trigger_words_found.join(", ")}</span></div>}
                    {res.content.shorteners_found?.length > 0 && <div>{t("shorteners")}: <span className="font-mono">{res.content.shorteners_found.join(", ")}</span></div>}
                    {res.content.all_caps_words > 0 && <div>{t("all_caps")}: {res.content.all_caps_words}</div>}
                    {res.content.image_only_warning && <div className="mt-1" style={{ color: "#991b1b" }}>{t("image_only_warning")}</div>}
                  </div>
                </div>
              )}
            </div>
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

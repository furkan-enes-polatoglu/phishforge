import { useState } from "react";
import { api } from "../api";

export default function Deliverability() {
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
      <h1 className="text-2xl font-bold">Deliverability</h1>
      <p className="text-sm muted">
        Legitimate pre-send email health checks — verify SPF/DKIM/DMARC, blocklists and markup so
        authorized test mail reaches the inbox. This is <b>not</b> a spam-filter evasion tool;
        coordinate an allowlist with the client's mail gateway.
      </p>

      <form onSubmit={run} className="card grid gap-3 sm:grid-cols-2">
        <input className="input" placeholder="Sender domain (e.g. mail.acme.com)" value={f.domain} onChange={(e) => setF({ ...f, domain: e.target.value })} required />
        <input className="input" placeholder="DKIM selector (optional)" value={f.dkim_selector} onChange={(e) => setF({ ...f, dkim_selector: e.target.value })} />
        <input className="input" placeholder="Sender IPv4 for RBL (optional)" value={f.sender_ip} onChange={(e) => setF({ ...f, sender_ip: e.target.value })} />
        <textarea className="input h-24 font-mono text-xs sm:col-span-2" placeholder="Paste email HTML for lint (optional)" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
        <div><button className="btn" disabled={busy}>{busy ? "Checking…" : "Run check"}</button></div>
      </form>

      {err && <div style={{ color: "#b91c1c" }}>{err}</div>}

      {res && (
        <div className="grid gap-6 lg:grid-cols-2">
          <div className="card">
            <div className="section-title mb-2">Authentication — {res.domain}</div>
            {rec("SPF", res.spf)}
            {rec("DMARC", res.dmarc)}
            {rec("DKIM", res.dkim)}
            {res.spam_score != null && (
              <div className="flex justify-between py-2"><span>SpamAssassin score</span><span className="badge badge-gray">{res.spam_score}</span></div>
            )}
            {res.rbl?.length > 0 && (
              <div className="mt-3">
                <div className="label">Blocklists</div>
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
              <div className="label">Advice</div>
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
    </div>
  );
}

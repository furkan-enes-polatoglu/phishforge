import { useState } from "react";
import { api } from "../api";

export default function Deliverability() {
  const [f, setF] = useState({ domain: "", dkim_selector: "", sender_ip: "", html: "" });
  const [res, setRes] = useState<any>(null);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function run(e: React.FormEvent) {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      setRes(await api("deliverability/check", { method: "POST", body: f }));
    } catch (e: any) {
      setErr(e.message);
    } finally {
      setBusy(false);
    }
  }

  const rec = (label: string, r: any) =>
    r && (
      <div className="flex items-center justify-between border-b border-slate-800/50 py-2">
        <span>{label}</span>
        <span className={`badge ${r.status === "ok" ? "bg-emerald-900 text-emerald-200" : r.status === "warn" ? "bg-amber-900 text-amber-200" : "bg-red-900 text-red-200"}`}>
          {r.status}
        </span>
      </div>
    );

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold">Deliverability</h1>
      <p className="text-sm text-slate-400">
        Legitimate pre-send email health checks — verify SPF/DKIM/DMARC, blocklists and markup so
        authorized test mail reaches the inbox. This is <b>not</b> a spam-filter evasion tool;
        coordinate an allowlist with the client's mail gateway.
      </p>

      <form onSubmit={run} className="card grid gap-3 sm:grid-cols-2">
        <input className="input" placeholder="Sender domain (e.g. mail.acme.com)" value={f.domain} onChange={(e) => setF({ ...f, domain: e.target.value })} required />
        <input className="input" placeholder="DKIM selector (optional)" value={f.dkim_selector} onChange={(e) => setF({ ...f, dkim_selector: e.target.value })} />
        <input className="input" placeholder="Sender IPv4 for RBL (optional)" value={f.sender_ip} onChange={(e) => setF({ ...f, sender_ip: e.target.value })} />
        <div className="sm:col-span-2">
          <textarea className="input h-24 font-mono text-xs" placeholder="Paste email HTML for lint (optional)" value={f.html} onChange={(e) => setF({ ...f, html: e.target.value })} />
        </div>
        <div><button className="btn" disabled={busy}>{busy ? "Checking…" : "Run check"}</button></div>
      </form>

      {err && <div className="text-red-300">{err}</div>}

      {res && (
        <div className="grid gap-6 lg:grid-cols-2">
          <div className="card">
            <h2 className="mb-2 font-medium">Authentication — {res.domain}</h2>
            {rec("SPF", res.spf)}
            {rec("DMARC", res.dmarc)}
            {rec("DKIM", res.dkim)}
            {res.spam_score != null && (
              <div className="flex justify-between py-2">
                <span>SpamAssassin score</span>
                <span className="badge bg-slate-700 text-slate-200">{res.spam_score}</span>
              </div>
            )}
            {res.rbl?.length > 0 && (
              <div className="mt-3">
                <div className="label">Blocklists</div>
                {res.rbl.map((b: any) => (
                  <div key={b.list} className="flex justify-between py-1 text-sm">
                    <span>{b.list}</span>
                    <span className={`badge ${b.listed ? "bg-red-900 text-red-200" : "bg-emerald-900 text-emerald-200"}`}>{b.listed ? "listed" : "clean"}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
          <div className="card space-y-3">
            <div>
              <div className="label">Advice</div>
              <ul className="list-disc space-y-1 pl-5 text-sm text-slate-300">
                {res.advice?.map((a: string, i: number) => <li key={i}>{a}</li>)}
              </ul>
            </div>
            {res.html_lint?.length > 0 && (
              <div>
                <div className="label">HTML lint</div>
                <ul className="list-disc space-y-1 pl-5 text-sm text-amber-200">
                  {res.html_lint.map((a: string, i: number) => <li key={i}>{a}</li>)}
                </ul>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

import { useState } from "react";
import { login } from "../api";
import { useI18n, Lang } from "../i18n";

export default function Login({ onLoggedIn }: { onLoggedIn: () => void }) {
  const { t, lang, setLang } = useI18n();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await login(email, password);
      onLoggedIn();
    } catch (e: any) {
      setErr(e.message || "login failed");
    } finally {
      setBusy(false);
    }
  }

  const langs: { code: Lang; label: string }[] = [
    { code: "tr", label: "🇹🇷 Türkçe" },
    { code: "en", label: "🇬🇧 English" },
  ];

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <form onSubmit={submit} className="card w-full max-w-sm space-y-5">
        <div className="flex justify-center gap-2">
          {langs.map((l) => (
            <button
              key={l.code}
              type="button"
              onClick={() => setLang(l.code)}
              className={lang === l.code ? "tab tab-active" : "tab"}
            >
              {l.label}
            </button>
          ))}
        </div>
        <div className="text-center">
          <div className="mb-2 text-5xl leading-none">🎣</div>
          <div className="text-xl font-bold">PhishForge</div>
          <p className="mt-1 text-xs muted">{t("login_subtitle")}</p>
        </div>
        {err && <div className="rounded-lg px-3 py-2 text-sm" style={{ background: "#fee2e2", color: "#991b1b" }}>{err}</div>}
        <div>
          <label className="label">{t("email")}</label>
          <input className="input" value={email} onChange={(e) => setEmail(e.target.value)} type="email" required />
        </div>
        <div>
          <label className="label">{t("password")}</label>
          <input className="input" value={password} onChange={(e) => setPassword(e.target.value)} type="password" required />
        </div>
        <button className="btn w-full" disabled={busy}>{busy ? t("signing_in") : t("sign_in")}</button>
      </form>
    </div>
  );
}

import { useEffect, useState } from "react";
import { api } from "../api";
import { useI18n } from "../i18n";

export default function Audit() {
  const { t } = useI18n();
  const [rows, setRows] = useState<any[]>([]);
  const [err, setErr] = useState("");
  useEffect(() => {
    api<any[]>("audit-log").then(setRows).catch((e) => setErr(e.message));
  }, []);
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">{t("audit_log")}</h1>
      <p className="text-sm muted">{t("audit_help")}</p>
      {err && <div style={{ color: "#b91c1c" }}>{err}</div>}
      <div className="card overflow-x-auto">
        <table className="data">
          <thead><tr><th>{t("when")}</th><th>{t("action")}</th><th>{t("entity")}</th><th>{t("detail")}</th></tr></thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.id}>
                <td className="muted">{new Date(r.created_at).toLocaleString()}</td>
                <td><span className="badge badge-gray">{r.action}</span></td>
                <td className="muted">{r.entity}</td>
                <td className="font-mono text-xs muted">{JSON.stringify(r.meta)}</td>
              </tr>
            ))}
            {rows.length === 0 && <tr><td colSpan={4} className="text-center muted">{t("none_yet")}</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}

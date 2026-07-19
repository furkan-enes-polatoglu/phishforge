import { useEffect, useState } from "react";
import { api } from "../api";
import { FunnelBars } from "../components/Funnel";
import { useI18n } from "../i18n";

interface Stats {
  engagements_total: number;
  engagements_active: number;
  role: string;
  funnel: Record<string, number>;
}

export default function Dashboard() {
  const { t } = useI18n();
  const [s, setS] = useState<Stats | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    api<Stats>("dashboard").then(setS).catch((e) => setErr(e.message));
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("dashboard")}</h1>
      {err && <div className="text-sm" style={{ color: "#b91c1c" }}>{err}</div>}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Stat label={t("stat_engagements")} value={s?.engagements_total ?? "—"} />
        <Stat label={t("stat_active")} value={s?.engagements_active ?? "—"} />
        <Stat label={t("stat_targets")} value={s?.funnel?.targets ?? "—"} />
        <Stat label={t("stat_role")} value={s?.role ?? "—"} />
      </div>

      <div className="card">
        <div className="section-title mb-3">{t("org_funnel")}</div>
        {s?.funnel ? <FunnelBars funnel={s.funnel} /> : <div className="muted text-sm">{t("loading")}</div>}
      </div>

      <div className="card text-sm">
        <p className="font-semibold">{t("authorized_only")}</p>
        <p className="mt-1 muted">{t("authorized_only_body")}</p>
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: any }) {
  return (
    <div className="stat">
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
    </div>
  );
}

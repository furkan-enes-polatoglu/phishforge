import { useEffect, useState } from "react";
import { Link, Navigate, Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { api, clearTokens, getAccess } from "./api";
import { useI18n } from "./i18n";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Engagements from "./pages/Engagements";
import EngagementDetail from "./pages/EngagementDetail";
import Assets from "./pages/Assets";
import CampaignReport from "./pages/CampaignReport";
import Deliverability from "./pages/Deliverability";
import Audit from "./pages/Audit";
import Training from "./pages/Training";
import Settings from "./pages/Settings";

interface Me {
  username: string;
  role: string;
}

function Shell({ me, onLogout }: { me: Me; onLogout: () => void }) {
  const loc = useLocation();
  const { t } = useI18n();
  const nav = [
    { to: "/", label: t("nav_dashboard"), icon: "📊" },
    { to: "/engagements", label: t("nav_engagements"), icon: "🎯" },
    { to: "/assets", label: t("nav_assets"), icon: "✉️" },
    { to: "/training", label: t("nav_training"), icon: "🎓" },
    { to: "/deliverability", label: t("nav_deliverability"), icon: "📬" },
    { to: "/settings", label: t("nav_settings"), icon: "⚙️" },
    { to: "/audit", label: t("nav_audit"), icon: "📝" },
  ];
  const active = (to: string) =>
    loc.pathname === to || (to !== "/" && loc.pathname.startsWith(to));
  return (
    <div className="flex min-h-screen">
      <aside className="sidebar">
        <div className="mb-6 flex items-center gap-2 px-2 text-lg font-bold text-white">
          <span className="grid h-9 w-9 place-items-center rounded-lg" style={{ background: "rgba(255,255,255,0.14)" }}>🎣</span>
          PhishForge
        </div>
        <nav className="flex flex-1 flex-col gap-1">
          {nav.map((n) => (
            <Link key={n.to} to={n.to} className={active(n.to) ? "side-link side-active" : "side-link"}>
              <span className="text-base">{n.icon}</span> {n.label}
            </Link>
          ))}
        </nav>
        <div className="mt-4 border-t pt-4" style={{ borderColor: "rgba(255,255,255,0.12)" }}>
          <div className="px-2 text-xs" style={{ color: "rgba(255,255,255,0.7)" }}>{me.username}</div>
          <div className="mt-1 flex items-center justify-between px-2">
            <span className="badge badge-blue">{me.role}</span>
            <button className="side-logout" onClick={onLogout}>{t("logout")}</button>
          </div>
        </div>
      </aside>
      <main className="flex-1 overflow-x-hidden px-6 py-6">
        <div className="mx-auto max-w-5xl">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/engagements" element={<Engagements />} />
            <Route path="/engagements/:id" element={<EngagementDetail />} />
            <Route path="/assets" element={<Assets />} />
            <Route path="/training" element={<Training />} />
            <Route path="/campaigns/:id" element={<CampaignReport />} />
            <Route path="/deliverability" element={<Deliverability />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/audit" element={<Audit />} />
            <Route path="*" element={<Navigate to="/" />} />
          </Routes>
        </div>
      </main>
    </div>
  );
}

export default function App() {
  const [me, setMe] = useState<Me | null>(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  async function loadMe() {
    if (!getAccess()) {
      setLoading(false);
      return;
    }
    try {
      const data = await api<Me>("auth/me");
      setMe(data);
    } catch {
      clearTokens();
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadMe();
  }, []);

  if (loading) return <div className="p-10 text-slate-400">Loading…</div>;

  if (!me) {
    return (
      <Routes>
        <Route
          path="*"
          element={<Login onLoggedIn={() => { setLoading(true); loadMe(); }} />}
        />
      </Routes>
    );
  }

  return (
    <Shell
      me={me}
      onLogout={() => {
        clearTokens();
        setMe(null);
        navigate("/");
      }}
    />
  );
}

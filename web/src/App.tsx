import { useEffect, useState } from "react";
import { Link, Navigate, Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { api, clearTokens, getAccess } from "./api";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Engagements from "./pages/Engagements";
import EngagementDetail from "./pages/EngagementDetail";
import Assets from "./pages/Assets";
import CampaignReport from "./pages/CampaignReport";
import Deliverability from "./pages/Deliverability";
import Audit from "./pages/Audit";

interface Me {
  email: string;
  role: string;
}

function Shell({ me, onLogout }: { me: Me; onLogout: () => void }) {
  const loc = useLocation();
  const nav = [
    { to: "/", label: "Dashboard" },
    { to: "/engagements", label: "Engagements" },
    { to: "/assets", label: "Assets" },
    { to: "/deliverability", label: "Deliverability" },
    { to: "/audit", label: "Audit" },
  ];
  return (
    <div className="min-h-screen">
      <header className="border-b border-slate-800 bg-slate-950/60">
        <div className="mx-auto flex max-w-6xl items-center gap-6 px-4 py-3">
          <span className="text-lg font-semibold">🎣 PhishForge</span>
          <nav className="flex gap-1">
            {nav.map((n) => (
              <Link
                key={n.to}
                to={n.to}
                className={`rounded-md px-3 py-1.5 text-sm ${
                  loc.pathname === n.to || (n.to !== "/" && loc.pathname.startsWith(n.to))
                    ? "bg-slate-800 text-white"
                    : "text-slate-300 hover:bg-slate-800/50"
                }`}
              >
                {n.label}
              </Link>
            ))}
          </nav>
          <div className="ml-auto flex items-center gap-3 text-sm text-slate-400">
            <span>
              {me.email} · <span className="badge bg-sky-900 text-sky-200">{me.role}</span>
            </span>
            <button className="btn-ghost" onClick={onLogout}>
              Log out
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/engagements" element={<Engagements />} />
          <Route path="/engagements/:id" element={<EngagementDetail />} />
          <Route path="/assets" element={<Assets />} />
          <Route path="/campaigns/:id" element={<CampaignReport />} />
          <Route path="/deliverability" element={<Deliverability />} />
          <Route path="/audit" element={<Audit />} />
          <Route path="*" element={<Navigate to="/" />} />
        </Routes>
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

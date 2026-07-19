// Minimal typed API client. Tokens are held in localStorage; on a 401 the access
// token is refreshed once using the refresh token.

const ACCESS = "pf_access";
const REFRESH = "pf_refresh";

export function getAccess() {
  return localStorage.getItem(ACCESS) || "";
}
export function setTokens(access: string, refresh?: string) {
  localStorage.setItem(ACCESS, access);
  if (refresh) localStorage.setItem(REFRESH, refresh);
}
export function clearTokens() {
  localStorage.removeItem(ACCESS);
  localStorage.removeItem(REFRESH);
}

async function refreshAccess(): Promise<boolean> {
  const refresh = localStorage.getItem(REFRESH);
  if (!refresh) return false;
  const res = await fetch("/api/auth/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refresh }),
  });
  if (!res.ok) return false;
  const data = await res.json();
  setTokens(data.access_token);
  return true;
}

export async function api<T = any>(
  path: string,
  opts: { method?: string; body?: any } = {}
): Promise<T> {
  const doFetch = () =>
    fetch(path.startsWith("/") ? path : `/api/${path}`, {
      method: opts.method || "GET",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${getAccess()}`,
      },
      body: opts.body ? JSON.stringify(opts.body) : undefined,
    });

  let res = await doFetch();
  if (res.status === 401 && (await refreshAccess())) {
    res = await doFetch();
  }
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    throw new Error(data?.error || `HTTP ${res.status}`);
  }
  return data as T;
}

export async function login(email: string, password: string) {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data?.error || "login failed");
  setTokens(data.access_token, data.refresh_token);
  return data;
}

# 🎣 PhishForge

An advanced, self-hostable **phishing-simulation & security-awareness** platform —
a modern, more capable take on GoPhish. Built for **authorized** red-team and
awareness engagements: measure user behavior, train people, and report results.

> ⚠️ **Authorized use only.** PhishForge is designed for engagements with **written
> client authorization**. Authorization guardrails are first-class features: every
> campaign runs inside an *engagement* (client + authorization reference + date
> window), targets must match an *allowlist*, submitted credential values are
> **never stored**, and all privileged actions are written to an append-only audit
> log. Do not use this software to phish anyone without explicit permission.

## Why PhishForge (vs. GoPhish)

See [`docs/gophish-analysis.md`](docs/gophish-analysis.md) for the full analysis.
Highlights:

| Area | GoPhish | PhishForge |
|------|---------|------------|
| Data store | SQLite monolith | PostgreSQL + Redis, API/worker split |
| Editors | HTML by hand | Code editor + live preview, merge-tags |
| Deliverability | none | SPF/DKIM/DMARC, RBL, SpamAssassin score, HTML lint |
| Analytics | basic counters | funnel + live timeline |
| Sending | simple mailer | rate-limited worker, scope re-check |
| Multi-tenant / RBAC | none | orgs + admin/operator/viewer |
| Authorization | operator's problem | engagement record + allowlist + audit log |
| Awareness | none | auto-redirect to training after submit |

Architecture details: [`docs/architecture.md`](docs/architecture.md).

## Stack

- **Backend:** Go 1.26 + Chi (single binary; `api` / `worker` / `migrate` modes)
- **Frontend:** React + TypeScript + Vite + Tailwind
- **Database:** PostgreSQL 16 · **Queue/cache:** Redis 7

## Quick start (Docker)

```bash
git clone <your-private-repo-url> phishforge && cd phishforge

cp .env.example .env
# Generate strong secrets and paste them into .env:
make seed-secrets          # prints JWT_SECRET and RID_SECRET
# Also set BOOTSTRAP_ADMIN_EMAIL / BOOTSTRAP_ADMIN_PASSWORD in .env

docker compose up -d --build
```

Then open:

- Admin UI + API: <http://localhost:8080>
- Phishing / tracking server: <http://localhost:8081>
- Health: <http://localhost:8080/healthz>

Log in with the bootstrap admin credentials from your `.env`. The first run
creates the org and admin automatically; migrations run on startup.

### Services

`docker compose` starts: `postgres`, `redis`, a one-shot `migrate`, the `api`
(serves the built SPA + admin API + phishing server), and the `worker`
(sends campaigns from the Redis queue).

## Typical workflow

1. **Create an engagement** — client name, **authorization reference** (e.g. signed
   SoW), and a start/end window.
2. **Define scope (allowlist)** — domains (`acme.com`) and/or email globs
   (`vip-*@acme.com`). At least one rule is required to activate.
3. **Activate** the engagement (only possible within scope + window).
4. **Import targets** — out-of-scope addresses are rejected automatically.
5. **Build assets** — email template (merge-tags: `{{.FirstName}}`, `{{.TrackURL}}`,
   `{{.TrackPixel}}`, `{{.ReportURL}}`) and a landing page (form posts to
   `{{.SubmitURL}}`; values are discarded, user is redirected to awareness training).
6. **Create a sending profile** — SMTP host/credentials, from address, STARTTLS.
7. **Run a deliverability check** — verify SPF/DKIM/DMARC, blocklists, HTML lint.
   Coordinate an allowlist with the client's mail gateway.
8. **Create & launch a campaign** — the worker sends at your configured rate and
   re-validates scope at send time.
9. **Watch the report** — sent → opened → clicked → submitted → reported funnel and
   a live event timeline.

## Deliverability, done legitimately

The deliverability module helps authorized test mail **reach the inbox** through
correct email infrastructure — SPF/DKIM/DMARC validation, blocklist checks, a
SpamAssassin score, HTML lint, and coordinated allowlisting with the client. It is
**not** a spam-filter evasion tool and contains no filter-deception techniques.

## Data minimization

When a target submits the landing-page form, PhishForge records only a `submit`
**event** — never the submitted values. Optionally (per landing page) it can record
the **set of field names** that were filled, still never any value. Passwords are
never stored, hashed, or logged.

## Local development

```bash
# Backend (needs Postgres + Redis + a .env)
make migrate && make run

# Frontend (proxies /api to :8080)
cd web && npm install && npm run dev   # http://localhost:5173

make test && make vet
```

## Production notes

- Put both ports behind TLS. Example with Caddy:

  ```
  admin.example.com { reverse_proxy 127.0.0.1:8080 }
  links.example.com { reverse_proxy 127.0.0.1:8081 }
  ```

  Set `ADMIN_BASE_URL`, `PHISH_BASE_URL`, and `CORS_ORIGINS` accordingly.
- Keep `JWT_SECRET` and `RID_SECRET` secret and stable (rotating `RID_SECRET`
  invalidates in-flight tracking links).
- Back up the `postgres` volume.

## License

MIT — see [`LICENSE`](LICENSE). Provided for authorized security testing and
awareness training. You are responsible for using it lawfully and with consent.

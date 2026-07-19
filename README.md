# 🎣 PhishForge

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-TS-61DAFB?logo=react)](https://react.dev)
[![Docker](https://img.shields.io/badge/Docker-compose-2496ED?logo=docker)](https://docs.docker.com/compose/)

An advanced, self-hostable, **open-source phishing-simulation & security-awareness**
platform — a modern, more capable take on GoPhish. Built for **authorized** red-team
and awareness engagements: measure user behavior, train people, and report results.

**One command to run it:**

```bash
git clone https://github.com/furkan-enes-polatoglu/phishforge.git
cd phishforge && ./scripts/quickstart.sh
```

That's it — the script generates strong secrets, prints your admin login, and starts
everything with Docker. Open <http://localhost:8080>. (Details below.)

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
| Awareness | none | training modules + auto-assign + completion tracking |
| Scheduling | basic | timezone-aware send windows, business-day gating, warm-up, jitter |
| A/B testing | none | multi-variant templates with per-variant funnel |
| Link tracking | manual | automatic link rewriting |
| Notifications | none | real-time Slack/Teams/webhook on open/click/submit/report |
| Automation | none | scoped API keys (`X-API-Key`) |
| Risk | none | per-user risk scoring across an engagement |
| Data capture | submitted data / passwords | same, opt-in **per landing page**, with password redaction control |

Architecture details: [`docs/architecture.md`](docs/architecture.md).

## Advanced features

- **Timezone-aware scheduling** — schedule a launch time, restrict sending to a
  daily window (evaluated in each recipient's timezone), skip weekends, ramp with
  a warm-up batch cap, and randomize timing with per-send jitter.
- **A/B testing** — attach multiple email-template variants (weighted); the worker
  splits targets across them and the report shows a per-variant funnel.
- **Automatic link rewriting** — every anchor in an email is rewritten to a signed
  tracked link, so clicks are recorded regardless of which link is followed.
- **Real-time notifications** — Slack/Teams incoming webhooks (auto-formatted) or
  signed JSON webhooks fire on open/click/submit/report.
- **Security-awareness training** — build training modules; targets who click or
  submit are auto-assigned and redirected, with completion tracked.
- **User risk scoring** — per-target scores aggregated across an engagement.
- **API keys** — automate everything via the `X-API-Key` header with a scoped role.

## Stack

- **Backend:** Go 1.26 + Chi (single binary; `api` / `worker` / `migrate` modes)
- **Frontend:** React + TypeScript + Vite + Tailwind
- **Database:** PostgreSQL 16 · **Queue/cache:** Redis 7

## Requirements

- **Docker** + the **Docker Compose v2** plugin. Nothing else — no Go or Node needed to run it.
  Install Docker: <https://docs.docker.com/get-docker/>

## Quick start (recommended)

```bash
git clone https://github.com/furkan-enes-polatoglu/phishforge.git
cd phishforge
./scripts/quickstart.sh
```

The script:
1. creates `.env` with strong random `JWT_SECRET` / `RID_SECRET`,
2. generates and **prints your admin email + password**,
3. builds the images and starts the whole stack.

Then open **<http://localhost:8080>** and log in with the printed credentials.

> Want to choose your own admin login? Set them before running:
> `ADMIN_EMAIL="you@example.com" ADMIN_PASS="a-strong-password" ./scripts/quickstart.sh`

## Manual start (if you prefer)

```bash
cp .env.example .env
make seed-secrets          # prints JWT_SECRET and RID_SECRET to paste into .env
# also set BOOTSTRAP_ADMIN_EMAIL / BOOTSTRAP_ADMIN_PASSWORD in .env
docker compose up -d --build
```

Endpoints:

- Admin UI + API: <http://localhost:8080>
- Phishing / tracking server: <http://localhost:8081>
- Health: <http://localhost:8080/healthz>

The first run creates the org and admin automatically; migrations run on startup.

## Manage it

```bash
docker compose logs -f api worker   # live logs
docker compose ps                   # status
docker compose down                 # stop (keeps data)
docker compose down -v              # stop and wipe all data
```

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

PhishForge maximizes inbox placement through **correct email infrastructure**, not
filter deception. It is **not** a spam-filter evasion tool.

- **DKIM signing** — generate a per-sending-profile RSA keypair in the UI; PhishForge
  publishes the DNS TXT record for you to add and signs every outbound message
  (RFC 6376). This is the single biggest legitimate deliverability win.
- **Well-formed messages** — multipart/alternative with a real text part, a valid
  `Date`, a unique `Message-ID`, and `List-Unsubscribe` headers.
- **Deliverability checker** — validate SPF/DKIM/DMARC, blocklists (RBL), a
  SpamAssassin score, and HTML lint before you send.
- **Reputation-friendly sending** — warm-up batching, per-send jitter, rate limits,
  and timezone-aware send windows.
- **Coordinate an allowlist** with the client's mail gateway — for an authorized
  test this is the most reliable path to the inbox.

## Manage everything (CRUD)

Engagements, email templates, landing pages, sending profiles, training modules,
campaigns, webhooks and API keys can all be **created, edited, duplicated and
deleted** from the UI, GoPhish-style.

## Language

The UI ships with a language selector on the login screen. **Turkish** is the
default; English is included as a base and more languages can be added in
`web/src/i18n.tsx`.

## Data capture (configurable per landing page)

Capture is **off by default** and controlled per landing page, so you record only
what the engagement authorizes:

- **default** — records only a `submit` event (no field data).
- **Capture field names** — records which field *names* were filled (no values).
- **Capture submitted data** — records the submitted field *values* (so you can
  show exactly what a target entered, GoPhish-style). Password-like fields are
  **redacted** unless the next option is also enabled.
- **Capture passwords** — also stores password-like values. This is sensitive:
  enable only with explicit client authorization, and handle/purge captured data
  per your engagement's rules. The UI shows a warning when this is on.

Captured values appear in the campaign report timeline under "Captured data".

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

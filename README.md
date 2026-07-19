# 🎣 PhishForge

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-TS-61DAFB?logo=react)](https://react.dev)
[![Docker](https://img.shields.io/badge/Docker-compose-2496ED?logo=docker)](https://docs.docker.com/compose/)

A self-hostable, **open-source phishing-simulation & security-awareness** platform
for **authorized** red-team and awareness engagements. Beyond the basics you'd
expect from any campaign tool, PhishForge focuses on the two things most
simulation tools get wrong: **actually landing in the inbox**, and giving
operators **red-team-grade simulation techniques**, not just an email blaster.

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
> **off by default**, and all privileged actions are written to an append-only audit
> log. Do not use this software to phish anyone without explicit permission.

## 🎯 Target mail gateway detection + allowlist playbook

The single highest-leverage action for guaranteed inbox delivery in an
authorized test is getting the sending infrastructure **explicitly allowlisted
in the exact product standing in front of the target mailbox**. Every gateway
vendor calls this something different and buries it in a different console —
PhishForge automates the reconnaissance and writes the request for you:

1. Enter the target company's domain — PhishForge resolves its MX records and
   fingerprints the gateway (Microsoft 365/EOP, Google Workspace, Proofpoint,
   Mimecast, Barracuda, or Cisco Secure Email/IronPort).
2. It returns the exact console path and steps to allowlist a sender in *that*
   product (e.g. Microsoft 365's "Advanced Delivery → Phishing simulation" — a
   feature Microsoft built specifically for third-party simulation tools).
3. It generates a ready-to-send, filled-in request email for the client's
   IT/security team, with a one-click copy button.

Validated against live MX records: Salesforce/Atlassian → Proofpoint,
Shopify/Stack Overflow → Google Workspace, GitHub → Microsoft 365.

## Deep deliverability engine

Not DNS-record-checking theater — a full pre-send diagnostic, aggregated into
one **Delivery Confidence Score (0-100, A-F)**:

- **DKIM signing** — generate a per-sending-profile RSA keypair in the UI;
  PhishForge publishes the DNS TXT record and signs every outbound message
  (RFC 6376).
- **PTR / Forward-Confirmed reverse DNS (FCrDNS)** — many corporate gateways
  silently drop mail from IPs with no PTR record or one that doesn't resolve
  back to the same IP. Commonly overlooked, checked automatically.
- **MTA-STS / TLS-RPT** checks, **DMARC alignment parsing** (`p=`/`sp=`/`aspf=`/`adkim=`/`rua=`)
  with specific alignment advice, **5 major blocklists queried in parallel**.
- **Content / spam-trigger analysis** — classic Bayesian-filter trigger
  phrases, link shorteners, ALL-CAPS shouting, image-only bodies.
- **Seed-list inbox placement test** — connects to a real mailbox over IMAP
  and reports whether a marked test send actually landed in the inbox or
  spam/junk — closing the loop instead of just checking DNS records.
- Realistic message shape: `Message-ID`, `Date`, `List-Unsubscribe`, and an
  optional `X-Mailer` header; rate limiting, warm-up batching, per-send
  jitter, and timezone-aware send windows to avoid burst-sending patterns.

This is **not** a spam-filter evasion tool — every technique here is legitimate
email infrastructure, the same things a real ESP does to protect its sender
reputation.

## Mailgun bounce/complaint feedback loop

A sending profile is just standard SMTP host/port/username/password — plug in
credentials from Mailgun (or any other server) and it works, no
provider-specific code involved. Raw SMTP is fire-and-forget, though: a
`250 OK` only means the relay *accepted* the message, not that it reached the
inbox. For Mailgun specifically, PhishForge closes that loop:

- When the sending profile's SMTP host is Mailgun's relay, every outbound
  message carries an `X-Mailgun-Variables` header with a correlation id — a
  header Mailgun recognizes on plain SMTP submissions (no API integration
  required) and echoes back on webhook events. It's omitted entirely for
  non-Mailgun profiles, where it would serve no purpose.
- A **signed webhook receiver** (`POST /webhooks/mailgun`, HMAC-verified
  against your Mailgun webhook signing key) ingests `delivered` / `failed`
  (bounce) / `complained` events in real time, matched back to the exact
  target via that correlation id — visible in the campaign report as
  **delivered / bounced / complained** counts, not just "sent."
- **Automatic reputation-safety pause** — once a campaign has at least 20
  sends with a result, a complaint rate above 0.3% or a bounce rate above 5%
  stops it immediately and logs why. A burnt sending domain silently damages
  every future engagement that reuses it; this catches it while it's
  happening instead of after a report is read.

Set `MAILGUN_WEBHOOK_SIGNING_KEY` (from Mailgun's dashboard: Sending →
Webhooks → Signing key) and point Mailgun's webhooks at
`https://<your-phish-domain>/webhooks/mailgun`.

## Pretext realism vs. authentication — the honest tradeoff

A campaign can show an **exact real address** in the visible "From:" field
(e.g. impersonating a real internal colleague or vendor), decoupled from the
sending profile's own technically-authenticated domain used for SPF/DKIM.
This is necessary for realistic pretexts, but it comes with a hard technical
fact: **spoofing a domain you don't control can never pass SPF/DKIM/DMARC
alignment** — only bypass filtering entirely. If the display address's domain
differs from the sending profile's domain, PhishForge shows a warning and
points you at the target gateway detection + allowlist playbook above — that
allowlist is what actually makes this land, not authentication tricks.

## Red-team simulation techniques

- **QR-code phishing (quishing)** — insert `<img src="{{.QRCodeURL}}">` in an
  email; scanning it is tracked as a distinct `scan` event before continuing
  into the normal click/landing flow.
- **Simulated attachment opens** — insert `<a href="{{.AttachmentURL}}">Invoice.pdf</a>`;
  "opening" it records an `attachment_open` event and routes to awareness
  training. No file is ever executed or delivered.
- **A/B testing** — attach multiple weighted email-template variants; the
  worker splits targets across them and the report shows a per-variant funnel.
- **Automatic link rewriting** — every anchor in an email is rewritten to a
  signed tracked link, so clicks are recorded regardless of which link is
  followed.
- **Timezone-aware scheduling** — a daily send window evaluated in each
  recipient's own timezone, business-day gating, warm-up ramp, and jitter.
- **Department & VIP tagging** — target metadata surfaces in per-user risk
  scoring so you can see which teams (or executives) are most exposed.
- **Security-awareness training loop** — targets who click or submit are
  auto-assigned a training module and redirected, with completion tracked —
  the awareness cycle closes itself.
- **Real-time notifications** — Slack/Teams or signed webhooks fire the
  instant a target opens, clicks, submits, or reports.
- **Scoped API keys** (`X-API-Key`) to drive engagements from your own tooling.
- **Bilingual bulk import** — upload a `.xlsx`/`.csv` target list with
  Turkish *or* English column headers in any order (`Ad Soyad`/`Full Name`,
  `Departman`/`Department`, `VIP`, …); out-of-scope rows are rejected
  automatically.

## Authorization guardrails (built in, not bolted on)

- Every campaign runs inside an **engagement**: client name, a written
  authorization reference, and a date window — nothing can send outside it.
- Targets must match an **allowlist** (domain or email pattern), enforced at
  import time *and* re-checked again at send time.
- Landing-page data capture is **off by default** and configurable per page:
  nothing captured → field names only → submitted values → passwords
  (explicitly opt-in, redacted unless enabled, UI warns when it's on).
- Every privileged action is written to an **append-only audit log**.
- Multi-tenant with role-based access (admin / operator / viewer).
- **Stopping a campaign immediately cuts off access to its links.** The
  tracking server checks the campaign's status (and its engagement's) on
  every request — an already-sent landing page/pixel/QR/report link stops
  resolving the moment you stop the campaign or the engagement closes/expires.
  A link from a *completed* (not stopped) campaign keeps working, so late
  clicks within an active engagement are still tracked correctly.

## Per-client landing domain

If you buy a fresh domain per engagement (website + SMTP on that same
domain — a common workflow for agencies running back-to-back client
simulations), set **Landing/tracking domain** on that client's sending
profile. Every tracking link (`{{.TrackURL}}`, `{{.QRCodeURL}}`,
`{{.AttachmentURL}}`, `{{.ReportURL}}`) in campaigns sent with that profile
uses that domain instead of the instance-wide default — no code change
needed. The tracking server itself is already Host-header-agnostic (it
resolves purely from the signed `rid` in the path), so pointing a new
domain's DNS A record at your server and adding a reverse-proxy site block
(Caddy's `on_demand_tls` handles this without editing config per domain) is
all that's required per new client domain.

## Also included

Turkish-first UI (with an English base and a login-screen language switch),
full CRUD (edit/duplicate/delete) on every asset, live previews with real
merge-tag substitution and a full-screen "open in new tab" view, a funnel +
timeline report per campaign, and campaign launch/stop/delete controls.

Curious how this compares feature-by-feature to GoPhish? See
[`docs/gophish-analysis.md`](docs/gophish-analysis.md). Architecture details:
[`docs/architecture.md`](docs/architecture.md).

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
2. generates and **prints your admin username + password**,
3. builds the images and starts the whole stack.

Then open **<http://localhost:8080>** and log in with the printed credentials.

> Want to choose your own admin login? Set them before running:
> `ADMIN_USER="admin" ADMIN_PASS="a-strong-password" ./scripts/quickstart.sh`

## Manual start (if you prefer)

```bash
cp .env.example .env
make seed-secrets          # prints JWT_SECRET and RID_SECRET to paste into .env
# also set BOOTSTRAP_ADMIN_USERNAME / BOOTSTRAP_ADMIN_PASSWORD in .env
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
4. **Import targets** — paste a list or upload an `.xlsx`/`.csv`; out-of-scope
   addresses are rejected automatically.
5. **Build assets** — email template (merge-tags: `{{.FirstName}}`, `{{.TrackURL}}`,
   `{{.QRCodeURL}}`, `{{.AttachmentURL}}`, `{{.ReportURL}}`) and a landing page
   (form posts to `{{.SubmitURL}}`; capture behavior is configured per page).
6. **Create a sending profile** — SMTP credentials (works with Mailgun's SMTP relay or any other server), DKIM key generation, optional `X-Mailer`.
7. **Run the deliverability check** — Delivery Confidence Score, and the
   target-gateway detection to get allowlisted before you send.
8. **Create & launch a campaign** — the worker sends at your configured rate
   and re-validates scope at send time.
9. **Watch the report** — sent → opened → clicked → submitted → reported
   funnel, per-variant A/B breakdown, and a live event timeline.

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

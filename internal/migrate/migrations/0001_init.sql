-- PhishForge initial schema
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE organizations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('admin','operator','viewer')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (email)
);
CREATE INDEX idx_users_org ON users(org_id);

CREATE TABLE engagements (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    client_name TEXT NOT NULL,
    authz_ref   TEXT NOT NULL,
    starts_at   TIMESTAMPTZ NOT NULL,
    ends_at     TIMESTAMPTZ NOT NULL,
    status      TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','active','closed')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_engagements_org ON engagements(org_id);

CREATE TABLE scope_rules (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    engagement_id UUID NOT NULL REFERENCES engagements(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL CHECK (kind IN ('domain','email')),
    pattern       TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_scope_engagement ON scope_rules(engagement_id);

CREATE TABLE sending_profiles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    smtp_host    TEXT NOT NULL,
    smtp_port    INT  NOT NULL DEFAULT 587,
    username     TEXT NOT NULL DEFAULT '',
    password     TEXT NOT NULL DEFAULT '',
    from_address TEXT NOT NULL,
    from_name    TEXT NOT NULL DEFAULT '',
    use_tls      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sending_profiles_org ON sending_profiles(org_id);

CREATE TABLE targets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    engagement_id UUID NOT NULL REFERENCES engagements(id) ON DELETE CASCADE,
    email         TEXT NOT NULL,
    first_name    TEXT NOT NULL DEFAULT '',
    last_name     TEXT NOT NULL DEFAULT '',
    position      TEXT NOT NULL DEFAULT '',
    timezone      TEXT NOT NULL DEFAULT 'UTC',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (engagement_id, email)
);
CREATE INDEX idx_targets_engagement ON targets(engagement_id);

CREATE TABLE email_templates (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    subject    TEXT NOT NULL,
    html       TEXT NOT NULL DEFAULT '',
    text       TEXT NOT NULL DEFAULT '',
    version    INT  NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_email_templates_org ON email_templates(org_id);

CREATE TABLE landing_pages (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    html         TEXT NOT NULL DEFAULT '',
    capture_meta BOOLEAN NOT NULL DEFAULT FALSE,
    redirect_url TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_landing_pages_org ON landing_pages(org_id);

CREATE TABLE campaigns (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    engagement_id      UUID NOT NULL REFERENCES engagements(id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    email_template_id  UUID NOT NULL REFERENCES email_templates(id),
    landing_page_id    UUID NOT NULL REFERENCES landing_pages(id),
    sending_profile_id UUID NOT NULL REFERENCES sending_profiles(id),
    status             TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','scheduled','running','completed')),
    launch_at          TIMESTAMPTZ,
    rate_per_minute    INT NOT NULL DEFAULT 30,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_campaigns_engagement ON campaigns(engagement_id);

CREATE TABLE campaign_targets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    target_id   UUID NOT NULL REFERENCES targets(id) ON DELETE CASCADE,
    rid         TEXT NOT NULL UNIQUE,
    status      TEXT NOT NULL DEFAULT 'pending',
    error       TEXT NOT NULL DEFAULT '',
    UNIQUE (campaign_id, target_id)
);
CREATE INDEX idx_campaign_targets_campaign ON campaign_targets(campaign_id);
CREATE INDEX idx_campaign_targets_rid ON campaign_targets(rid);

CREATE TABLE events (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_target_id UUID NOT NULL REFERENCES campaign_targets(id) ON DELETE CASCADE,
    type               TEXT NOT NULL CHECK (type IN ('sent','open','click','submit','report')),
    ip                 TEXT NOT NULL DEFAULT '',
    user_agent         TEXT NOT NULL DEFAULT '',
    meta               JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_events_ct ON events(campaign_target_id);
CREATE INDEX idx_events_type ON events(type);

CREATE TABLE deliverability_reports (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain      TEXT NOT NULL,
    spf         JSONB NOT NULL DEFAULT '{}'::jsonb,
    dkim        JSONB NOT NULL DEFAULT '{}'::jsonb,
    dmarc       JSONB NOT NULL DEFAULT '{}'::jsonb,
    spam_score  DOUBLE PRECISION NOT NULL DEFAULT 0,
    rbl         JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL,
    actor_id   UUID,
    action     TEXT NOT NULL,
    entity     TEXT NOT NULL,
    entity_id  TEXT NOT NULL DEFAULT '',
    meta       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_org ON audit_log(org_id);

CREATE TABLE webhooks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    url        TEXT NOT NULL,
    secret     TEXT NOT NULL DEFAULT '',
    events     TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- append-only guard: forbid UPDATE/DELETE on audit_log
CREATE OR REPLACE FUNCTION forbid_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_no_update BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION forbid_mutation();

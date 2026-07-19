-- PhishForge advanced features: scheduling, A/B, training, risk, API keys.

-- Scheduling & send-window controls on campaigns.
ALTER TABLE campaigns
    ADD COLUMN send_window_start  INT     NOT NULL DEFAULT 0,     -- local hour 0..23 (inclusive)
    ADD COLUMN send_window_end    INT     NOT NULL DEFAULT 24,    -- local hour 0..24 (exclusive)
    ADD COLUMN business_days_only BOOLEAN NOT NULL DEFAULT FALSE, -- skip Sat/Sun
    ADD COLUMN jitter_seconds     INT     NOT NULL DEFAULT 0,     -- random extra delay per send
    ADD COLUMN warmup_batch       INT     NOT NULL DEFAULT 0,     -- max sends per scheduler cycle (0 = no cap)
    ADD COLUMN rewrite_links      BOOLEAN NOT NULL DEFAULT TRUE;  -- auto-rewrite <a href> to tracked links

-- Credential/data capture on landing pages (GoPhish-parity, explicit opt-in).
-- capture_submitted_data: store submitted field values (password-like fields
--   redacted unless capture_passwords is also enabled).
-- capture_passwords: also store password-like field values (sensitive!).
ALTER TABLE landing_pages
    ADD COLUMN capture_submitted_data BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN capture_passwords      BOOLEAN NOT NULL DEFAULT FALSE;

-- A/B testing: a campaign may have multiple email-template variants.
CREATE TABLE campaign_variants (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id       UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    email_template_id UUID NOT NULL REFERENCES email_templates(id),
    weight            INT  NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_variants_campaign ON campaign_variants(campaign_id);

ALTER TABLE campaign_targets
    ADD COLUMN variant_id UUID REFERENCES campaign_variants(id);

-- Security-awareness training modules and per-target assignments.
CREATE TABLE training_modules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    html       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_training_modules_org ON training_modules(org_id);

CREATE TABLE training_assignments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id    UUID NOT NULL REFERENCES targets(id) ON DELETE CASCADE,
    module_id    UUID NOT NULL REFERENCES training_modules(id) ON DELETE CASCADE,
    campaign_id  UUID REFERENCES campaigns(id) ON DELETE SET NULL,
    status       TEXT NOT NULL DEFAULT 'assigned' CHECK (status IN ('assigned','completed')),
    token        TEXT NOT NULL UNIQUE,
    assigned_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);
CREATE INDEX idx_training_assign_target ON training_assignments(target_id);
CREATE INDEX idx_training_assign_token ON training_assignments(token);

-- API keys for automation (hashed at rest; only a prefix is stored in clear).
CREATE TABLE api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    prefix       TEXT NOT NULL,
    key_hash     TEXT NOT NULL,
    role         TEXT NOT NULL DEFAULT 'operator' CHECK (role IN ('admin','operator','viewer')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    revoked      BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX idx_api_keys_org ON api_keys(org_id);
CREATE INDEX idx_api_keys_prefix ON api_keys(prefix);

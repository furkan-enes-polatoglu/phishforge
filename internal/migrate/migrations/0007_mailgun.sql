-- Mailgun HTTP API support + delivery feedback loop (bounce/complaint webhooks)
-- + pretext realism (spoofed display-From / Reply-To decoupled from the
-- technically-authenticated sending domain).

ALTER TABLE sending_profiles
    ADD COLUMN provider         TEXT NOT NULL DEFAULT 'smtp' CHECK (provider IN ('smtp','mailgun_api')),
    ADD COLUMN mailgun_api_key  TEXT NOT NULL DEFAULT '',
    ADD COLUMN mailgun_domain   TEXT NOT NULL DEFAULT '';

ALTER TABLE campaigns
    ADD COLUMN spoofed_from_name    TEXT NOT NULL DEFAULT '',
    ADD COLUMN spoofed_from_address TEXT NOT NULL DEFAULT '',
    ADD COLUMN reply_to             TEXT NOT NULL DEFAULT '';

-- Delivery feedback events reported asynchronously by the ESP (Mailgun) via
-- webhook: delivered (confirmed inbox handoff), bounced (hard/soft failure),
-- complained (recipient marked as spam — the single worst reputation signal).
ALTER TABLE events DROP CONSTRAINT events_type_check;
ALTER TABLE events ADD CONSTRAINT events_type_check
    CHECK (type IN ('sent','open','click','submit','report','scan','attachment_open',
                     'delivered','bounced','complained'));

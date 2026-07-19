-- Department/VIP tagging on targets (for red-team reporting & bulk import).
ALTER TABLE targets
    ADD COLUMN department TEXT    NOT NULL DEFAULT '',
    ADD COLUMN is_vip     BOOLEAN NOT NULL DEFAULT FALSE;

-- Realistic mail-client header (deliverability).
ALTER TABLE sending_profiles
    ADD COLUMN x_mailer TEXT NOT NULL DEFAULT '';

-- New simulation event types: QR-code scan (quishing) and simulated attachment open.
ALTER TABLE events DROP CONSTRAINT events_type_check;
ALTER TABLE events ADD CONSTRAINT events_type_check
    CHECK (type IN ('sent','open','click','submit','report','scan','attachment_open'));

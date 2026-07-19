-- DKIM signing support on sending profiles (legitimate deliverability, not evasion).
ALTER TABLE sending_profiles
    ADD COLUMN dkim_domain      TEXT    NOT NULL DEFAULT '',
    ADD COLUMN dkim_selector    TEXT    NOT NULL DEFAULT '',
    ADD COLUMN dkim_private_key TEXT    NOT NULL DEFAULT '',
    ADD COLUMN sign_dkim        BOOLEAN NOT NULL DEFAULT FALSE;

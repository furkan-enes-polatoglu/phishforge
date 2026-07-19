-- Per-campaign landing/tracking domain override — like GoPhish's "URL" field on
-- the campaign launch screen, but pre-filled from the sending profile's own
-- domain (set once per client) rather than retyped every launch. Empty means
-- "use the sending profile's domain" (which itself falls back to the global
-- default).
ALTER TABLE campaigns
    ADD COLUMN landing_base_url TEXT NOT NULL DEFAULT '';

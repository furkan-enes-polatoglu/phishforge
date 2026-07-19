-- Per-client landing/tracking domain: operators who buy a fresh domain per
-- engagement (website + SMTP on that domain) can now point tracking links in
-- that client's emails at that same domain, instead of a single global one.
ALTER TABLE sending_profiles
    ADD COLUMN landing_base_url TEXT NOT NULL DEFAULT '';

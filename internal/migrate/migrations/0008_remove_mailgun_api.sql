-- Remove the Mailgun HTTP API sending path — unnecessary complexity for a
-- product that already speaks standard SMTP (which Mailgun's own SMTP relay
-- uses like any other server). The webhook receiver + bounce/complaint
-- feedback loop (events: delivered/bounced/complained) is unaffected: Mailgun
-- fires those webhooks at the account/domain level regardless of whether mail
-- was submitted via SMTP or the HTTP API.
ALTER TABLE sending_profiles
    DROP COLUMN provider,
    DROP COLUMN mailgun_api_key,
    DROP COLUMN mailgun_domain;

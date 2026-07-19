-- Switch PhishForge's own login accounts from email addresses to plain
-- usernames (e.g. "admin", "furkan"). This does NOT affect phishing targets,
-- sending-profile SMTP credentials, or email templates — those still use real
-- email addresses, only the operator/admin login identity changes.
ALTER TABLE users RENAME COLUMN email TO username;

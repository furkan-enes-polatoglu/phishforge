-- Allow campaigns to be stopped (halted mid-run).
ALTER TABLE campaigns DROP CONSTRAINT campaigns_status_check;
ALTER TABLE campaigns ADD CONSTRAINT campaigns_status_check
    CHECK (status IN ('draft','scheduled','running','completed','stopped'));

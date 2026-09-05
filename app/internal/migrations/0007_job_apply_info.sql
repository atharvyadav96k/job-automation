-- How to apply, when a source provides it (JSearch does; Remotive/Greenhouse/
-- Lever don't) — surfaced on the job detail page so the account owner can
-- apply manually. All nullable; never fabricated when a source lacks it.
ALTER TABLE jobs ADD COLUMN apply_url TEXT;
ALTER TABLE jobs ADD COLUMN apply_portal TEXT;
ALTER TABLE jobs ADD COLUMN contact_email TEXT;

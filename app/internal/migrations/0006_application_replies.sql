-- Reply tracking: the account owner reports replies manually (there's no
-- inbox integration), so these fields are only ever written by that manual
-- action — see the "I got a reply" flow in the review UI.
ALTER TABLE applications ADD COLUMN replied_at TIMESTAMPTZ;
ALTER TABLE applications ADD COLUMN reply_channel TEXT;
ALTER TABLE applications ADD COLUMN outcome TEXT NOT NULL DEFAULT 'none'; -- 'none', 'interview', 'rejected', 'offer', 'other'

-- M3 purse (D-042). Integer 냥 on the account sheet.

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS nyang INTEGER NOT NULL DEFAULT 0;

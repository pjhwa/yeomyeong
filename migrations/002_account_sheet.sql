-- Account sheet (issue #42, D-034). Four JSON columns keyed by account.

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS skills JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS stats JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS bag JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS equipment JSONB NOT NULL DEFAULT '{}'::jsonb;

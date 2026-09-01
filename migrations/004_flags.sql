-- Per-player story flags (first-10-minutes talk memory). Empty {} on old rows.

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS flags JSONB NOT NULL DEFAULT '{}'::jsonb;

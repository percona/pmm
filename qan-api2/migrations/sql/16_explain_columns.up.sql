ALTER TABLE metrics
  ADD COLUMN `explain_fingerprint` String,
  ADD COLUMN `placeholders_count` UInt32;

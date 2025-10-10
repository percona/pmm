ALTER TABLE metrics
  ADD COLUMN `top_queryid` LowCardinality(String),
  ADD COLUMN `application_name` LowCardinality(String),
  ADD COLUMN `planid` LowCardinality(String);

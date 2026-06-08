CREATE MATERIALIZED VIEW IF NOT EXISTS mv_dim_query TO dim_query AS
SELECT
  queryid,
  anyLast(fingerprint) AS fingerprint,
  anyLast(`tables`) AS `tables`,
  anyLast(explain_fingerprint) AS explain_fingerprint,
  anyLast(placeholders_count) AS placeholders_count,
  min(period_start) AS first_seen,
  max(period_start) AS last_seen
FROM metrics_raw
GROUP BY queryid;

SELECT pg_database.datname,tmp.mode,COALESCE(count,$1) as count
FROM ( VALUES ($2), ($3), ($4), ($5), ($6), ($7), ($8), ($9) ) AS tmp(mode)
CROSS JOIN pg_database
LEFT JOIN (SELECT database, lower(mode) AS mode,count(*) AS count FROM pg_locks WHERE database IS NOT NULL GROUP BY database, lower(mode) ) AS tmp2
ON tmp.mode=tmp2.mode and pg_database.oid = tmp2.database ORDER BY 1

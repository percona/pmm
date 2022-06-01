SELECT setup::date
FROM generate_series('2007-02-01', '2007-02-28', INTERVAL '1 day') AS setup

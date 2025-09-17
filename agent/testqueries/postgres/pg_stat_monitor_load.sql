DROP EXTENSION pg_stat_monitor;
Create EXTENSION pg_stat_monitor;

Set application_name = 'naeem' ; 
SELECT 1 AS num;
Set application_name = 'psql' ; 
SELECT 1 AS num;
SELECT query,application_name FROM pg_stat_monitor ORDER BY query, application_name COLLATE "C";

SELECT 1 AS num;
SELECT query,application_name FROM pg_stat_monitor ORDER BY query COLLATE "C";

SELECT 1 AS num;
SELECT query FROM pg_stat_monitor ORDER BY query COLLATE "C";
CREATE TABLE t1 (a INTEGER);
CREATE TABLE t2 (b INTEGER);
INSERT INTO t1 VALUES(1);
SELECT a FROM t1;
UPDATE t1 SET a = 2;
DELETE FROM t1;
SELECT b FROM t2 FOR UPDATE;
TRUNCATE t1;
DROP TABLE t1;
DROP TABLE t2;
SELECT query, cmd_type,  cmd_type_text FROM pg_stat_monitor ORDER BY query COLLATE "C";

CREATE TABLE t1 (a INTEGER);
CREATE TABLE t2 (b INTEGER);
CREATE TABLE t3 (c INTEGER);
CREATE TABLE t4 (d INTEGER);

Select * from pg_stat_monitor_settings; 
SELECT a,b,c,d FROM t1, t2, t3, t4 WHERE t1.a = t2.b AND t3.c = t4.d ORDER BY a;
SELECT a,b,c,d FROM t1, t2, t3, t4 WHERE t1.a = t2.b AND t3.c = t4.d ORDER BY a;
SELECT a,b,c,d FROM t1, t2, t3, t4 WHERE t1.a = t2.b AND t3.c = t4.d ORDER BY a;
SELECT a,b,c,d FROM t1, t2, t3, t4 WHERE t1.a = t2.b AND t3.c = t4.d ORDER BY a;
SELECT query,calls FROM pg_stat_monitor ORDER BY query COLLATE "C";

ALTER SYSTEM SET pg_stat_monitor.pgsm_track TO 'all';
SELECT pg_reload_conf();
Select * from pg_stat_monitor_settings; 
SELECT pg_sleep(2);

do $$
declare
   n integer:= 1;
begin
	loop
		PERFORM a,b,c,d FROM t1, t2, t3, t4 WHERE t1.a = t2.b AND t3.c = t4.d ORDER BY a;
		exit when n = 1000;
		n := n + 1;
	end loop;
end $$;
SELECT query,calls FROM pg_stat_monitor ORDER BY query COLLATE "C";

ALTER SYSTEM SET pg_stat_monitor.pgsm_track TO 'top';
SELECT pg_reload_conf();
SELECT pg_sleep(1);

DROP TABLE t1;
DROP TABLE t2;
DROP TABLE t3;
DROP TABLE t4;

Drop Table if exists Company;

CREATE TABLE Company(
   ID INT PRIMARY KEY     NOT NULL,
   NAME TEXT    NOT NULL
);

INSERT  INTO Company(ID, Name) VALUES (1, 'Percona'); 
INSERT  INTO Company(ID, Name) VALUES (1, 'Percona'); 

Drop Table if exists Company;
SELECT query, elevel, sqlcode, message FROM pg_stat_monitor ORDER BY query COLLATE "C",elevel;

SELECT 1/0;   -- divide by zero
SELECT * FROM unknown; -- unknown table
ELECET * FROM unknown; -- syntax error

do $$
BEGIN
RAISE WARNING 'warning message';
END $$;

SELECT query, elevel, sqlcode, message FROM pg_stat_monitor ORDER BY query COLLATE "C",elevel;

select pg_sleep(.5);
SELECT * FROM pg_stat_monitor_settings ORDER BY name COLLATE "C";

CREATE OR REPLACE FUNCTION generate_histogram()
    RETURNS TABLE (
    range TEXT, freq INT, bar TEXT
  )  AS $$
Declare
    bucket_id integer;
    query_id text;
BEGIN
    select bucket into bucket_id from pg_stat_monitor order by calls desc limit 1;
    select queryid into query_id from pg_stat_monitor order by calls desc limit 1;
    --RAISE INFO 'bucket_id %', bucket_id;
    --RAISE INFO 'query_id %', query_id;
    return query
    SELECT * FROM histogram(bucket_id, query_id) AS a(range TEXT, freq INT, bar TEXT);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION run_pg_sleep(INTEGER) RETURNS VOID AS $$
DECLARE
    loops ALIAS FOR $1;
BEGIN
    FOR i IN 1..loops LOOP
	--RAISE INFO 'Current timestamp: %', timeofday()::TIMESTAMP;
	RAISE INFO 'Sleep % seconds', i;
	PERFORM pg_sleep(i);
    END LOOP;
END;
$$ LANGUAGE 'plpgsql' STRICT;

Set pg_stat_monitor.pgsm_track='all';
select run_pg_sleep(5);

SELECT substr(query, 0,50) as query, calls, resp_calls FROM pg_stat_monitor ORDER BY query COLLATE "C";

select * from generate_histogram();

CREATE TABLE foo1(a int);
CREATE TABLE foo2(b int);
CREATE TABLE foo3(c int);
CREATE TABLE foo4(d int);

-- test the simple table names
SELECT * FROM foo1;
SELECT * FROM foo1, foo2;
SELECT * FROM foo1, foo2, foo3;
SELECT * FROM foo1, foo2, foo3, foo4;
SELECT query, relations from pg_stat_monitor ORDER BY query collate "C";

-- test the schema qualified table
CREATE schema sch1;
CREATE schema sch2;
CREATE schema sch3;
CREATE schema sch4;

CREATE TABLE sch1.foo1(a int);
CREATE TABLE sch2.foo2(b int);
CREATE TABLE sch3.foo3(c int);
CREATE TABLE sch4.foo4(d int);

SELECT * FROM sch1.foo1;
SELECT * FROM sch1.foo1, sch2.foo2;
SELECT * FROM sch1.foo1, sch2.foo2, sch3.foo3;
SELECT * FROM sch1.foo1, sch2.foo2, sch3.foo3, sch4.foo4;
SELECT query, relations from pg_stat_monitor ORDER BY query collate "C";

SELECT * FROM sch1.foo1, foo1;
SELECT * FROM sch1.foo1, sch2.foo2, foo1, foo2;
SELECT query, relations from pg_stat_monitor ORDER BY query;

-- test the view
CREATE VIEW v1 AS SELECT * from foo1;
CREATE VIEW v2 AS SELECT * from foo1,foo2;
CREATE VIEW v3 AS SELECT * from foo1,foo2,foo3;
CREATE VIEW v4 AS SELECT * from foo1,foo2,foo3,foo4;

SELECT * FROM v1;
SELECT * FROM v1,v2;
SELECT * FROM v1,v2,v3;
SELECT * FROM v1,v2,v3,v4;
SELECT query, relations from pg_stat_monitor ORDER BY query collate "C";

DROP VIEW v1;
DROP VIEW v2;
DROP VIEW v3;
DROP VIEW v4;

DROP TABLE foo1;
DROP TABLE foo2;
DROP TABLE foo3;
DROP TABLE foo4;

DROP TABLE sch1.foo1;
DROP TABLE sch2.foo2;
DROP TABLE sch3.foo3;
DROP TABLE sch4.foo4;

DROP SCHEMA sch1;
DROP SCHEMA sch2;
DROP SCHEMA sch3;
DROP SCHEMA sch4;

CREATE TABLE t1(a int);
CREATE TABLE t2(b int);
INSERT INTO t1 VALUES(generate_series(1,1000));
INSERT INTO t2 VALUES(generate_series(1,5000));

SELECT * FROM t1;
SELECT * FROM t2;

SELECT * FROM t1 LIMIT 10;
SELECt * FROM t2  WHERE b % 2 = 0;

SELECT query, rows_retrieved FROM pg_stat_monitor ORDER BY query COLLATE "C";

DROP TABLE t1;
DROP TABLE t2;

SELECT 1 AS num /* { "application", psql_app, "real_ip", 192.168.1.3) */;
SELECT query, comments FROM pg_stat_monitor ORDER BY query COLLATE "C";
ALTER SYSTEM SET pg_stat_monitor.pgsm_extract_comments TO 'yes';
SELECT pg_reload_conf();
select pg_sleep(1);
SELECT 1 AS num /* { "application", psql_app, "real_ip", 192.168.1.3) */;
SELECT query, comments FROM pg_stat_monitor ORDER BY query COLLATE "C";
ALTER SYSTEM SET pg_stat_monitor.pgsm_extract_comments TO 'no';
SELECT pg_reload_conf();

CREATE OR REPLACE FUNCTION add(int, int) RETURNS INTEGER AS
$$
BEGIN
	return (select $1 + $2);
END; $$ language plpgsql;

CREATE OR REPLACE function add2(int, int) RETURNS int as
$$
BEGIN
	return add($1,$2);
END;
$$ language plpgsql;

SELECT add2(1,2);
SELECT query, top_query FROM pg_stat_monitor ORDER BY query COLLATE "C";

ALTER SYSTEM SET pg_stat_monitor.pgsm_track TO 'all';
SELECT pg_reload_conf();
SELECT pg_sleep(1);

CREATE OR REPLACE FUNCTION add(int, int) RETURNS INTEGER AS
$$
BEGIN
	return (select $1 + $2);
END; $$ language plpgsql;

CREATE OR REPLACE function add2(int, int) RETURNS int as
$$
BEGIN
	return add($1,$2);
END;
$$ language plpgsql;

SELECT add2(1,2);
SELECT query, top_query FROM pg_stat_monitor ORDER BY query COLLATE "C";
ALTER SYSTEM SET pg_stat_monitor.pgsm_track TO 'top';
SELECT pg_reload_conf();
SELECT pg_sleep(1);

CREATE USER su WITH SUPERUSER;

SET ROLE su;

CREATE USER u1;
CREATE USER u2;

SET ROLE su;

SET ROLE u1;
CREATE TABLE t1 (a int);
SELECT * FROM t1;

SET ROLE u2;
CREATE TABLE t2 (a int);
SELECT * FROM t2;

SET ROLE su;
SELECT  bucket, userid, datname, client_ip, application_name,top_queryid, planid, queryid, calls, substr(query,0,50) as query FROM pg_stat_monitor ORDER BY query COLLATE "C";
DROP TABLE t1;
DROP TABLE t2;

DROP USER u1;
DROP USER u2;

SELECT pg_stat_monitor_version();

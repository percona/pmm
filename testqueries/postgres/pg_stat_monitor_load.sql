\c contrib_regression

-- application_name.sql
SELECT 1 AS num;
SELECT query,application_name FROM pg_stat_monitor ORDER BY query COLLATE "C";

-- basic.sql
SELECT 1 AS num;
SELECT query FROM pg_stat_monitor ORDER BY query COLLATE "C";

-- cmd_type.sql
CREATE TABLE t1 (a INTEGER);
CREATE TABLE t2 (b INTEGER);
INSERT INTO t1 VALUES(1);
SELECT a FROM t1;
UPDATE t1 SET a = 2;
DELETE FROM t1;
SELECT b FROM t2 FOR UPDATE;
TRUNCATE t1;
DROP TABLE t1;
SELECT query, cmd_type,  cmd_type_text FROM pg_stat_monitor ORDER BY query COLLATE "C";

-- counters.sql
CREATE TABLE t1 (a INTEGER);
CREATE TABLE t2 (b INTEGER);
CREATE TABLE t3 (c INTEGER);
CREATE TABLE t4 (d INTEGER);


SELECT a,b,c,d FROM t1, t2, t3, t4 WHERE t1.a = t2.b AND t3.c = t4.d ORDER BY a;
SELECT a,b,c,d FROM t1, t2, t3, t4 WHERE t1.a = t2.b AND t3.c = t4.d ORDER BY a;
SELECT a,b,c,d FROM t1, t2, t3, t4 WHERE t1.a = t2.b AND t3.c = t4.d ORDER BY a;
SELECT a,b,c,d FROM t1, t2, t3, t4 WHERE t1.a = t2.b AND t3.c = t4.d ORDER BY a;
SELECT query,calls FROM pg_stat_monitor ORDER BY query COLLATE "C";

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


DROP TABLE t1;
DROP TABLE t2;
DROP TABLE t3;
DROP TABLE t4;

-- database.sql
CREATE DATABASE db1;
CREATE DATABASE db2;

\c db1
CREATE TABLE t1 (a int);
CREATE TABLE t2 (b int);

\c db2
CREATE TABLE t3 (c int);
CREATE TABLE t4 (d int);

\c contrib_regression

\c db1
SELECT * FROM t1,t2 WHERE t1.a = t2.b;

\c db2
SELECT * FROM t3,t4 WHERE t3.c = t4.d;

\c contrib_regression
SELECT datname, query FROM pg_stat_monitor ORDER BY query COLLATE "C";


\c db1
DROP TABLE t1;
DROP TABLE t2;

\c db2
DROP TABLE t3;
DROP TABLE t4;

\c contrib_regression
DROP DATABASE db1;
DROP DATABASE db2;

-- error.sql
SELECT 1/0;   -- divide by zero
SELECT * FROM unknown; -- unknown table
SELECET * FROM unknown; -- syntax error

do $$
BEGIN
RAISE WARNING 'warning message';
END $$;

SELECT query, elevel, sqlcode, message FROM pg_stat_monitor ORDER BY query COLLATE "C";

-- guc.sql
select pg_sleep(.5);
SELECT * FROM pg_stat_monitor_settings ORDER BY name COLLATE "C";

-- histogram.sql
CREATE TABLE t1(a int);

INSERT INTO t1 VALUES(generate_series(1,10));
ANALYZE t1;
SELECT count(*) FROM t1;

INSERT INTO t1 VALUES(generate_series(1,10000));
ANALYZE t1;
SELECT count(*) FROM t1;;

INSERT INTO t1 VALUES(generate_series(1,1000000));
ANALYZE t1;
SELECT count(*) FROM t1;

INSERT INTO t1 VALUES(generate_series(1,10000000));
ANALYZE t1;
SELECT count(*) FROM t1;

SELECT query, calls, min_time, max_time, resp_calls FROM pg_stat_monitor ORDER BY query COLLATE "C";
SELECT * FROM histogram(0, 'F44CD1B4B33A47AF') AS a(range TEXT, freq INT, bar TEXT);

DROP TABLE t1;

-- relations.sql
CREATE TABLE foo1(a int);
CREATE TABLE foo2(b int);
CREATE TABLE foo3(c int);
CREATE TABLE foo4(d int);

-- test the simple table names

SELECT * FROM foo1;
SELECT * FROM foo1, foo2;
SELECT * FROM foo1, foo2, foo3;
SELECT * FROM foo1, foo2, foo3, foo4;
SELECT query, relations from pg_stat_monitor ORDER BY query;

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
SELECT query, relations from pg_stat_monitor ORDER BY query;

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
SELECT query, relations from pg_stat_monitor ORDER BY query;

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

-- rows.sql
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

-- state.sql
SELECT 1;
SELECT 1/0;   -- divide by zero
SELECT query, state_code, state FROM pg_stat_monitor ORDER BY query COLLATE "C";

-- tags.sql
SELECT 1 AS num /* { "application", psql_app, "real_ip", 192.168.1.3) */;
SELECT query, comments FROM pg_stat_monitor ORDER BY query COLLATE "C";

-- top_query.sql
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

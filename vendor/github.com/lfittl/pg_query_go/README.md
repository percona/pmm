# pg_query_go [![Build Status](https://travis-ci.org/lfittl/pg_query_go.svg)](https://travis-ci.org/lfittl/pg_query_go) [![GoDoc](https://godoc.org/github.com/lfittl/pg_query_go?status.svg)](https://godoc.org/github.com/lfittl/pg_query_go)

Go version of https://github.com/lfittl/pg_query

This Go library and its cgo extension use the actual PostgreSQL server source to parse SQL queries and return the internal PostgreSQL parse tree.

Note that the original Ruby version of this library is much more feature complete.

You can find further background to why a query's parse tree is useful here: https://pganalyze.com/blog/parse-postgresql-queries-in-ruby.html


## Installation

```
go get github.com/lfittl/pg_query_go
```

Due to compiling parts of PostgreSQL, the first time you build against this library it will take a bit longer.

Expect up to 3 minutes. You can use `go build -x` to see the progress.

## Usage with Go modules

When integrating this library using Go modules, and using a vendor/ directory,
you will need to explicitly copy over some of the C build files, since Go does
not copy files in subfolders without .go files whilst vendoring.

The best way to do so is to use [modvendor](https://github.com/goware/modvendor),
and vendor your modules like this:

```
go mod vendor
go get -u github.com/goware/modvendor
modvendor -copy="**/*.c **/*.h **/*.proto" -v
```

## Usage

### Parsing a query into JSON

Put the following in a new Go package, after having installed pg_query as above:

```go
package main

import (
  "fmt"
  "github.com/lfittl/pg_query_go"
)

func main() {
  tree, err := pg_query.ParseToJSON("SELECT 1")
  if err != nil {
    panic(err);
  }
  fmt.Printf("%s\n", tree)
}
```

Running will output the query's parse tree as JSON:

```json
$ go run main.go
[{"SELECT": {"distinctClause": null, "intoClause": null, "targetList": [{"RESTARGET": {"name": null, "indirection": null, "val": {"A_CONST": {"val": 1, "location": 7}}, "location": 7}}], "fromClause": null, "whereClause": null, "groupClause": null, "havingClause": null, "windowClause": null, "valuesLists": null, "sortClause": null, "limitOffset": null, "limitCount": null, "lockingClause": null, "withClause": null, "op": 0, "all": false, "larg": null, "rarg": null}}]
```

### Parsing a query into Go structs

When working with the query information inside Go its recommended you use the `Parse()` method which returns Go structs:

```go
package main

import (
  "fmt"
  "reflect"
  "github.com/lfittl/pg_query_go"
  nodes "github.com/lfittl/pg_query_go/nodes"
)

func main() {
  tree, err := pg_query.Parse("SELECT 1")
  if err != nil {
    panic(err);
  }

  fmt.Printf("%s\n", reflect.DeepEqual(tree, pg_query.ParsetreeList{
    Statements: []nodes.Node{
      nodes.SelectStmt{
        TargetList: []nodes.Node{
          nodes.ResTarget{
            Val: nodes.A_Const{
              Type: "integer",
              Val: nodes.Value{
                Type: nodes.T_Integer,
                Ival: 1,
              },
              Location: 7,
            },
            Location: 7,
          },
        },
      },
    },
  }));
}
```

You can find all the node struct types in the `nodes/` directory.

### Parsing a PL/pgSQL function into JSON (Experimental)

Put the following in a new Go package, after having installed pg_query as above:

```go
package main

import (
  "fmt"
  "github.com/lfittl/pg_query_go"
)

func main() {
  tree, err := pg_query.ParsePlPgSqlToJSON(
  `CREATE OR REPLACE FUNCTION cs_fmt_browser_version(v_name varchar, v_version varchar)
  			RETURNS varchar AS $$
  			BEGIN
  			    IF v_version IS NULL THEN
  			        RETURN v_name;
  			    END IF;
  			    RETURN v_name || '/' || v_version;
  			END;
  			$$ LANGUAGE plpgsql;`)
  if err != nil {
    panic(err);
  }
  fmt.Printf("%s\n", tree)
}
```

Running will output the functions's parse tree as JSON:

```json
$ go run main.go
[
{"PLpgSQL_function": {"datums": [{"PLpgSQL_var": {"refname": "found", "datatype": {"PLpgSQL_type": {"typname": "UNKNOWN"}}}}], "action": {"PLpgSQL_stmt_block": {"lineno": 2, "body": [{"PLpgSQL_stmt_if": {"lineno": 3, "cond": {"PLpgSQL_expr": {"query": "SELECT v_version IS NULL"}}, "then_body": [{"PLpgSQL_stmt_return": {"lineno": 4, "expr": {"PLpgSQL_expr": {"query": "SELECT v_name"}}}}]}}, {"PLpgSQL_stmt_return": {"lineno": 6, "expr": {"PLpgSQL_expr": {"query": "SELECT v_name || '/' || v_version"}}}}]}}}}
]
```

## Benchmarks

As it stands, parsing has considerable overhead for complex queries, due to the use of JSON to pass structs across the C <=> Go barrier.

```
BenchmarkParseSelect1-4               	  300000	     40723 ns/op	   14608 B/op	     226 allocs/op
BenchmarkParseSelect2-4               	  100000	    164339 ns/op	   49105 B/op	     742 allocs/op
BenchmarkParseCreateTable-4           	   30000	    504815 ns/op	  149826 B/op	    2123 allocs/op
BenchmarkParseSelect1Parallel-4       	 1000000	     12245 ns/op	   14608 B/op	     226 allocs/op
BenchmarkParseSelect2Parallel-4       	  300000	     46268 ns/op	   49105 B/op	     742 allocs/op
BenchmarkParseCreateTableParallel-4   	  100000	    157849 ns/op	  149827 B/op	    2123 allocs/op
```

A good portion of this is due to JSON parsing inside Go so we can work with Go structs - just the raw parser is 10x faster:

```
BenchmarkRawParseSelect1-4            	 5000000	      3012 ns/op	     192 B/op	       2 allocs/op
BenchmarkRawParseSelect2-4            	 2000000	      7755 ns/op	     672 B/op	       2 allocs/op
BenchmarkRawParseCreateTable-4        	 1000000	     20848 ns/op	    2080 B/op	       2 allocs/op
BenchmarkRawParseSelect1Parallel-4    	20000000	       801 ns/op	     192 B/op	       2 allocs/op
BenchmarkRawParseSelect2Parallel-4    	10000000	      2220 ns/op	     672 B/op	       2 allocs/op
BenchmarkRawParseCreateTableParallel-4	 2000000	      6153 ns/op	    2080 B/op	       2 allocs/op
```

Similarly, for query fingerprinting, you might want to use `pg_query.FastFingerprint` to let the C extension handle it:

```
BenchmarkFingerprintSelect1-4         	  300000	     42318 ns/op	   15564 B/op	     246 allocs/op
BenchmarkFingerprintSelect2-4         	  100000	    164205 ns/op	   53215 B/op	     834 allocs/op
BenchmarkFingerprintCreateTable-4     	   30000	    524524 ns/op	  162972 B/op	    2371 allocs/op
BenchmarkFastFingerprintSelect1-4     	 5000000	      3614 ns/op	      80 B/op	       2 allocs/op
BenchmarkFastFingerprintSelect2-4     	 2000000	      6748 ns/op	      80 B/op	       2 allocs/op
BenchmarkFastFingerprintCreateTable-4 	 1000000	     18361 ns/op	      80 B/op	       2 allocs/op
```

Normalization is already handled in the C extension, doesn't depend on JSON parsing at all, and is fast:

```
BenchmarkNormalizeSelect1-4           	10000000	      1859 ns/op	      24 B/op	       2 allocs/op
BenchmarkNormalizeSelect2-4           	 5000000	      3551 ns/op	      64 B/op	       2 allocs/op
BenchmarkNormalizeCreateTable-4       	 2000000	      6051 ns/op	     144 B/op	       2 allocs/op
```

See `benchmark_test.go` for the queries.

Benchmark numbers from running on a 3.2 GHz Intel Core i5 CPU, OSX 10.11.


## Authors

- [Lukas Fittl](mailto:lukas@fittl.com)


## License

Copyright (c) 2015, Lukas Fittl <lukas@fittl.com><br>
pg_query_go is licensed under the 3-clause BSD license, see LICENSE file for details.

This project includes code derived from the [PostgreSQL project](http://www.postgresql.org/),
see LICENSE.POSTGRESQL for details.

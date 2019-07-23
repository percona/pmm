package pg_query

import (
	"encoding/json"

	"github.com/lfittl/pg_query_go/parser"
)

// ParseToJSON - Parses the given SQL statement into an AST (JSON format)
func ParseToJSON(input string) (result string, err error) {
	return parser.ParseToJSON(input)
}

// Parse the given SQL statement into an AST (native Go structs)
func Parse(input string) (tree ParsetreeList, err error) {
	jsonTree, err := ParseToJSON(input)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(jsonTree), &tree)
	return
}

// ParsePlPgSqlToJSON - Parses the given PL/pgSQL function statement into an AST (JSON format)
func ParsePlPgSqlToJSON(input string) (result string, err error) {
	return parser.ParsePlPgSqlToJSON(input)
}

// Normalize the passed SQL statement to replace constant values with ? characters
func Normalize(input string) (result string, err error) {
	return parser.Normalize(input)
}

// FastFingerprint - Fingerprint the passed SQL statement using the C extension
func FastFingerprint(input string) (result string, err error) {
	return parser.FastFingerprint(input)
}

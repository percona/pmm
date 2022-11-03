package queryparser

import (
	"github.com/pkg/errors"
	"vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/sqlparser"
)

func MySQL(example string) (string, uint32, error) {
	normalizedQuery, _, err := sqlparser.Parse2(example)
	if err != nil {
		return "", 0, errors.Wrap(err, "cannot parse query")
	}

	bv := make(map[string]*query.BindVariable)
	err = sqlparser.Normalize(normalizedQuery, sqlparser.NewReservedVars("", sqlparser.GetBindvars(normalizedQuery)), bv)
	if err != nil {
		return "", 0, errors.Wrap(err, "cannot normalize query")
	}

	parsedQuery := sqlparser.NewParsedQuery(normalizedQuery)
	bindVars := sqlparser.GetBindvars(normalizedQuery)

	return parsedQuery.Query, uint32(len(bindVars)), nil
}

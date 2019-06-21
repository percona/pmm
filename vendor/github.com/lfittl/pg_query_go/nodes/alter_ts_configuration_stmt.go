// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type AlterTSConfigurationStmt struct {
	Kind    AlterTSConfigType `json:"kind"`    /* ALTER_TSCONFIG_ADD_MAPPING, etc */
	Cfgname List              `json:"cfgname"` /* qualified name (list of Value strings) */

	/*
	 * dicts will be non-NIL if ADD/ALTER MAPPING was specified. If dicts is
	 * NIL, but tokentype isn't, DROP MAPPING was specified.
	 */
	Tokentype List `json:"tokentype"`  /* list of Value strings */
	Dicts     List `json:"dicts"`      /* list of list of Value strings */
	Override  bool `json:"override"`   /* if true - remove old variant */
	Replace   bool `json:"replace"`    /* if true - replace dictionary by another */
	MissingOk bool `json:"missing_ok"` /* for DROP - skip error if missing? */
}

func (node AlterTSConfigurationStmt) MarshalJSON() ([]byte, error) {
	type AlterTSConfigurationStmtMarshalAlias AlterTSConfigurationStmt
	return json.Marshal(map[string]interface{}{
		"AlterTSConfigurationStmt": (*AlterTSConfigurationStmtMarshalAlias)(&node),
	})
}

func (node *AlterTSConfigurationStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["kind"] != nil {
		err = json.Unmarshal(fields["kind"], &node.Kind)
		if err != nil {
			return
		}
	}

	if fields["cfgname"] != nil {
		node.Cfgname.Items, err = UnmarshalNodeArrayJSON(fields["cfgname"])
		if err != nil {
			return
		}
	}

	if fields["tokentype"] != nil {
		node.Tokentype.Items, err = UnmarshalNodeArrayJSON(fields["tokentype"])
		if err != nil {
			return
		}
	}

	if fields["dicts"] != nil {
		node.Dicts.Items, err = UnmarshalNodeArrayJSON(fields["dicts"])
		if err != nil {
			return
		}
	}

	if fields["override"] != nil {
		err = json.Unmarshal(fields["override"], &node.Override)
		if err != nil {
			return
		}
	}

	if fields["replace"] != nil {
		err = json.Unmarshal(fields["replace"], &node.Replace)
		if err != nil {
			return
		}
	}

	if fields["missing_ok"] != nil {
		err = json.Unmarshal(fields["missing_ok"], &node.MissingOk)
		if err != nil {
			return
		}
	}

	return
}

// Auto-generated - DO NOT EDIT

package pg_query

import (
	"encoding/json"
	"fmt"
	"strings"
)

func UnmarshalNodeJSON(input json.RawMessage) (node Node, err error) {
	if input == nil || string(input) == "null" {
		return
	}

	if strings.HasPrefix(string(input), "[") {
		var list List
		list.Items, err = UnmarshalNodeArrayJSON(input)
		if err != nil {
			return
		}
		node = list
		return
	}

	var nodeMap map[string]json.RawMessage

	err = json.Unmarshal(input, &nodeMap)
	if err != nil {
		return
	}

	for nodeType, jsonText := range nodeMap {
		switch nodeType {

		case "Query":
			var outNode Query
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "TypeName":
			var outNode TypeName
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ColumnRef":
			var outNode ColumnRef
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ParamRef":
			var outNode ParamRef
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "A_Expr":
			var outNode A_Expr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "A_Const":
			var outNode A_Const
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "TypeCast":
			var outNode TypeCast
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CollateClause":
			var outNode CollateClause
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RoleSpec":
			var outNode RoleSpec
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "FuncCall":
			var outNode FuncCall
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "A_Star":
			var outNode A_Star
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "A_Indices":
			var outNode A_Indices
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "A_Indirection":
			var outNode A_Indirection
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "A_ArrayExpr":
			var outNode A_ArrayExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ResTarget":
			var outNode ResTarget
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "MultiAssignRef":
			var outNode MultiAssignRef
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "SortBy":
			var outNode SortBy
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "WindowDef":
			var outNode WindowDef
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RangeSubselect":
			var outNode RangeSubselect
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RangeFunction":
			var outNode RangeFunction
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RangeTableFunc":
			var outNode RangeTableFunc
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RangeTableFuncCol":
			var outNode RangeTableFuncCol
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RangeTableSample":
			var outNode RangeTableSample
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ColumnDef":
			var outNode ColumnDef
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "TableLikeClause":
			var outNode TableLikeClause
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "IndexElem":
			var outNode IndexElem
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DefElem":
			var outNode DefElem
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "LockingClause":
			var outNode LockingClause
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "XmlSerialize":
			var outNode XmlSerialize
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "PartitionElem":
			var outNode PartitionElem
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "PartitionSpec":
			var outNode PartitionSpec
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "PartitionBoundSpec":
			var outNode PartitionBoundSpec
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "PartitionRangeDatum":
			var outNode PartitionRangeDatum
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "PartitionCmd":
			var outNode PartitionCmd
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RangeTblEntry":
			var outNode RangeTblEntry
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RangeTblFunction":
			var outNode RangeTblFunction
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "TableSampleClause":
			var outNode TableSampleClause
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "WithCheckOption":
			var outNode WithCheckOption
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "SortGroupClause":
			var outNode SortGroupClause
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "GroupingSet":
			var outNode GroupingSet
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "WindowClause":
			var outNode WindowClause
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RowMarkClause":
			var outNode RowMarkClause
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "WithClause":
			var outNode WithClause
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "InferClause":
			var outNode InferClause
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "OnConflictClause":
			var outNode OnConflictClause
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CommonTableExpr":
			var outNode CommonTableExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "TriggerTransition":
			var outNode TriggerTransition
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RawStmt":
			var outNode RawStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "InsertStmt":
			var outNode InsertStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DeleteStmt":
			var outNode DeleteStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "UpdateStmt":
			var outNode UpdateStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "SelectStmt":
			var outNode SelectStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "SetOperationStmt":
			var outNode SetOperationStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateSchemaStmt":
			var outNode CreateSchemaStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterTableStmt":
			var outNode AlterTableStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ReplicaIdentityStmt":
			var outNode ReplicaIdentityStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterTableCmd":
			var outNode AlterTableCmd
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterCollationStmt":
			var outNode AlterCollationStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterDomainStmt":
			var outNode AlterDomainStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "GrantStmt":
			var outNode GrantStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ObjectWithArgs":
			var outNode ObjectWithArgs
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AccessPriv":
			var outNode AccessPriv
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "GrantRoleStmt":
			var outNode GrantRoleStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterDefaultPrivilegesStmt":
			var outNode AlterDefaultPrivilegesStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CopyStmt":
			var outNode CopyStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "VariableSetStmt":
			var outNode VariableSetStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "VariableShowStmt":
			var outNode VariableShowStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateStmt":
			var outNode CreateStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "Constraint":
			var outNode Constraint
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateTableSpaceStmt":
			var outNode CreateTableSpaceStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DropTableSpaceStmt":
			var outNode DropTableSpaceStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterTableSpaceOptionsStmt":
			var outNode AlterTableSpaceOptionsStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterTableMoveAllStmt":
			var outNode AlterTableMoveAllStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateExtensionStmt":
			var outNode CreateExtensionStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterExtensionStmt":
			var outNode AlterExtensionStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterExtensionContentsStmt":
			var outNode AlterExtensionContentsStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateFdwStmt":
			var outNode CreateFdwStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterFdwStmt":
			var outNode AlterFdwStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateForeignServerStmt":
			var outNode CreateForeignServerStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterForeignServerStmt":
			var outNode AlterForeignServerStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateForeignTableStmt":
			var outNode CreateForeignTableStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateUserMappingStmt":
			var outNode CreateUserMappingStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterUserMappingStmt":
			var outNode AlterUserMappingStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DropUserMappingStmt":
			var outNode DropUserMappingStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ImportForeignSchemaStmt":
			var outNode ImportForeignSchemaStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreatePolicyStmt":
			var outNode CreatePolicyStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterPolicyStmt":
			var outNode AlterPolicyStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateAmStmt":
			var outNode CreateAmStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateTrigStmt":
			var outNode CreateTrigStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateEventTrigStmt":
			var outNode CreateEventTrigStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterEventTrigStmt":
			var outNode AlterEventTrigStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreatePLangStmt":
			var outNode CreatePLangStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateRoleStmt":
			var outNode CreateRoleStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterRoleStmt":
			var outNode AlterRoleStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterRoleSetStmt":
			var outNode AlterRoleSetStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DropRoleStmt":
			var outNode DropRoleStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateSeqStmt":
			var outNode CreateSeqStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterSeqStmt":
			var outNode AlterSeqStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DefineStmt":
			var outNode DefineStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateDomainStmt":
			var outNode CreateDomainStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateOpClassStmt":
			var outNode CreateOpClassStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateOpClassItem":
			var outNode CreateOpClassItem
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateOpFamilyStmt":
			var outNode CreateOpFamilyStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterOpFamilyStmt":
			var outNode AlterOpFamilyStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DropStmt":
			var outNode DropStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "TruncateStmt":
			var outNode TruncateStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CommentStmt":
			var outNode CommentStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "SecLabelStmt":
			var outNode SecLabelStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DeclareCursorStmt":
			var outNode DeclareCursorStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ClosePortalStmt":
			var outNode ClosePortalStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "FetchStmt":
			var outNode FetchStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "IndexStmt":
			var outNode IndexStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateStatsStmt":
			var outNode CreateStatsStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateFunctionStmt":
			var outNode CreateFunctionStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "FunctionParameter":
			var outNode FunctionParameter
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterFunctionStmt":
			var outNode AlterFunctionStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DoStmt":
			var outNode DoStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "InlineCodeBlock":
			var outNode InlineCodeBlock
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RenameStmt":
			var outNode RenameStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterObjectDependsStmt":
			var outNode AlterObjectDependsStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterObjectSchemaStmt":
			var outNode AlterObjectSchemaStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterOwnerStmt":
			var outNode AlterOwnerStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterOperatorStmt":
			var outNode AlterOperatorStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RuleStmt":
			var outNode RuleStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "NotifyStmt":
			var outNode NotifyStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ListenStmt":
			var outNode ListenStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "UnlistenStmt":
			var outNode UnlistenStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "TransactionStmt":
			var outNode TransactionStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CompositeTypeStmt":
			var outNode CompositeTypeStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateEnumStmt":
			var outNode CreateEnumStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateRangeStmt":
			var outNode CreateRangeStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterEnumStmt":
			var outNode AlterEnumStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ViewStmt":
			var outNode ViewStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "LoadStmt":
			var outNode LoadStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreatedbStmt":
			var outNode CreatedbStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterDatabaseStmt":
			var outNode AlterDatabaseStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterDatabaseSetStmt":
			var outNode AlterDatabaseSetStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DropdbStmt":
			var outNode DropdbStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterSystemStmt":
			var outNode AlterSystemStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ClusterStmt":
			var outNode ClusterStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "VacuumStmt":
			var outNode VacuumStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ExplainStmt":
			var outNode ExplainStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateTableAsStmt":
			var outNode CreateTableAsStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RefreshMatViewStmt":
			var outNode RefreshMatViewStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CheckPointStmt":
			var outNode CheckPointStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DiscardStmt":
			var outNode DiscardStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "LockStmt":
			var outNode LockStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ConstraintsSetStmt":
			var outNode ConstraintsSetStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ReindexStmt":
			var outNode ReindexStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateConversionStmt":
			var outNode CreateConversionStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateCastStmt":
			var outNode CreateCastStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateTransformStmt":
			var outNode CreateTransformStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "PrepareStmt":
			var outNode PrepareStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ExecuteStmt":
			var outNode ExecuteStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DeallocateStmt":
			var outNode DeallocateStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DropOwnedStmt":
			var outNode DropOwnedStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ReassignOwnedStmt":
			var outNode ReassignOwnedStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterTSDictionaryStmt":
			var outNode AlterTSDictionaryStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterTSConfigurationStmt":
			var outNode AlterTSConfigurationStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreatePublicationStmt":
			var outNode CreatePublicationStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterPublicationStmt":
			var outNode AlterPublicationStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CreateSubscriptionStmt":
			var outNode CreateSubscriptionStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlterSubscriptionStmt":
			var outNode AlterSubscriptionStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "DropSubscriptionStmt":
			var outNode DropSubscriptionStmt
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "Alias":
			var outNode Alias
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RangeVar":
			var outNode RangeVar
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "TableFunc":
			var outNode TableFunc
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "IntoClause":
			var outNode IntoClause
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "Expr":
			var outNode Expr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "Var":
			var outNode Var
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "Const":
			var outNode Const
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "Param":
			var outNode Param
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "Aggref":
			var outNode Aggref
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "GroupingFunc":
			var outNode GroupingFunc
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "WindowFunc":
			var outNode WindowFunc
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ArrayRef":
			var outNode ArrayRef
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "FuncExpr":
			var outNode FuncExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "NamedArgExpr":
			var outNode NamedArgExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "OpExpr":
			var outNode OpExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ScalarArrayOpExpr":
			var outNode ScalarArrayOpExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "BoolExpr":
			var outNode BoolExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "SubLink":
			var outNode SubLink
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "SubPlan":
			var outNode SubPlan
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "AlternativeSubPlan":
			var outNode AlternativeSubPlan
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "FieldSelect":
			var outNode FieldSelect
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "FieldStore":
			var outNode FieldStore
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RelabelType":
			var outNode RelabelType
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CoerceViaIO":
			var outNode CoerceViaIO
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ArrayCoerceExpr":
			var outNode ArrayCoerceExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ConvertRowtypeExpr":
			var outNode ConvertRowtypeExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CollateExpr":
			var outNode CollateExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CaseExpr":
			var outNode CaseExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CaseWhen":
			var outNode CaseWhen
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CaseTestExpr":
			var outNode CaseTestExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ArrayExpr":
			var outNode ArrayExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RowExpr":
			var outNode RowExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RowCompareExpr":
			var outNode RowCompareExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CoalesceExpr":
			var outNode CoalesceExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "MinMaxExpr":
			var outNode MinMaxExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "SQLValueFunction":
			var outNode SQLValueFunction
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "XmlExpr":
			var outNode XmlExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "NullTest":
			var outNode NullTest
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "BooleanTest":
			var outNode BooleanTest
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CoerceToDomain":
			var outNode CoerceToDomain
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CoerceToDomainValue":
			var outNode CoerceToDomainValue
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "SetToDefault":
			var outNode SetToDefault
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "CurrentOfExpr":
			var outNode CurrentOfExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "NextValueExpr":
			var outNode NextValueExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "InferenceElem":
			var outNode InferenceElem
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "TargetEntry":
			var outNode TargetEntry
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "RangeTblRef":
			var outNode RangeTblRef
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "JoinExpr":
			var outNode JoinExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "FromExpr":
			var outNode FromExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "OnConflictExpr":
			var outNode OnConflictExpr
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ParamExternData":
			var outNode ParamExternData
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ParamListInfoData":
			var outNode ParamListInfoData
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "ParamExecData":
			var outNode ParamExecData
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "varatt_external":
			var outNode varatt_external
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "BlockIdData":
			var outNode BlockIdData
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "Integer":
			var outNode Integer
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "Float":
			var outNode Float
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "String":
			var outNode String
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "BitString":
			var outNode BitString
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "Null":
			var outNode Null
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		case "List":
			var outNode List
			err = json.Unmarshal(jsonText, &outNode)
			if err != nil {
				return
			}
			node = outNode
		default:
			err = fmt.Errorf("Could not unmarshal node of type %s and content %s", nodeType, jsonText)
			return
		}
	}

	return
}

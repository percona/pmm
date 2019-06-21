// Auto-generated from postgres/src/include/nodes/nodes.h - DO NOT EDIT

package pg_query

/*
 * CmdType -
 *	  enums for type of operation represented by a Query or PlannedStmt
 *
 * This is needed in both parsenodes.h and plannodes.h, so put it here...
 */
type CmdType uint

const (
	CMD_UNKNOWN CmdType = iota
	CMD_SELECT          /* select stmt */
	CMD_UPDATE          /* update stmt */
	CMD_INSERT          /* insert stmt */
	CMD_DELETE
	CMD_UTILITY /* cmds like create, destroy, copy, vacuum,
	 * etc. */

	CMD_NOTHING /* dummy command for instead nothing rules
	 * with qual */
)

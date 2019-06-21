// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/* ----------------------
 *		{Begin|Commit|Rollback} Transaction Statement
 * ----------------------
 */
type TransactionStmtKind uint

const (
	TRANS_STMT_BEGIN TransactionStmtKind = iota
	TRANS_STMT_START                     /* semantically identical to BEGIN */
	TRANS_STMT_COMMIT
	TRANS_STMT_ROLLBACK
	TRANS_STMT_SAVEPOINT
	TRANS_STMT_RELEASE
	TRANS_STMT_ROLLBACK_TO
	TRANS_STMT_PREPARE
	TRANS_STMT_COMMIT_PREPARED
	TRANS_STMT_ROLLBACK_PREPARED
)

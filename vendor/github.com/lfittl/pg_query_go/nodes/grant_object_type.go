// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

type GrantObjectType uint

const (
	ACL_OBJECT_COLUMN         GrantObjectType = iota /* column */
	ACL_OBJECT_RELATION                              /* table, view */
	ACL_OBJECT_SEQUENCE                              /* sequence */
	ACL_OBJECT_DATABASE                              /* database */
	ACL_OBJECT_DOMAIN                                /* domain */
	ACL_OBJECT_FDW                                   /* foreign-data wrapper */
	ACL_OBJECT_FOREIGN_SERVER                        /* foreign server */
	ACL_OBJECT_FUNCTION                              /* function */
	ACL_OBJECT_LANGUAGE                              /* procedural language */
	ACL_OBJECT_LARGEOBJECT                           /* largeobject */
	ACL_OBJECT_NAMESPACE                             /* namespace */
	ACL_OBJECT_TABLESPACE                            /* tablespace */
	ACL_OBJECT_TYPE                                  /* type */
)

// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

type AlterTableType uint

const (
	AT_AddColumn                 AlterTableType = iota /* add column */
	AT_AddColumnRecurse                                /* internal to commands/tablecmds.c */
	AT_AddColumnToView                                 /* implicitly via CREATE OR REPLACE VIEW */
	AT_ColumnDefault                                   /* alter column default */
	AT_DropNotNull                                     /* alter column drop not null */
	AT_SetNotNull                                      /* alter column set not null */
	AT_SetStatistics                                   /* alter column set statistics */
	AT_SetOptions                                      /* alter column set ( options ) */
	AT_ResetOptions                                    /* alter column reset ( options ) */
	AT_SetStorage                                      /* alter column set storage */
	AT_DropColumn                                      /* drop column */
	AT_DropColumnRecurse                               /* internal to commands/tablecmds.c */
	AT_AddIndex                                        /* add index */
	AT_ReAddIndex                                      /* internal to commands/tablecmds.c */
	AT_AddConstraint                                   /* add constraint */
	AT_AddConstraintRecurse                            /* internal to commands/tablecmds.c */
	AT_ReAddConstraint                                 /* internal to commands/tablecmds.c */
	AT_AlterConstraint                                 /* alter constraint */
	AT_ValidateConstraint                              /* validate constraint */
	AT_ValidateConstraintRecurse                       /* internal to commands/tablecmds.c */
	AT_ProcessedConstraint                             /* pre-processed add constraint (local in
	 * parser/parse_utilcmd.c) */

	AT_AddIndexConstraint        /* add constraint using existing index */
	AT_DropConstraint            /* drop constraint */
	AT_DropConstraintRecurse     /* internal to commands/tablecmds.c */
	AT_ReAddComment              /* internal to commands/tablecmds.c */
	AT_AlterColumnType           /* alter column type */
	AT_AlterColumnGenericOptions /* alter column OPTIONS (...) */
	AT_ChangeOwner               /* change owner */
	AT_ClusterOn                 /* CLUSTER ON */
	AT_DropCluster               /* SET WITHOUT CLUSTER */
	AT_SetLogged                 /* SET LOGGED */
	AT_SetUnLogged               /* SET UNLOGGED */
	AT_AddOids                   /* SET WITH OIDS */
	AT_AddOidsRecurse            /* internal to commands/tablecmds.c */
	AT_DropOids                  /* SET WITHOUT OIDS */
	AT_SetTableSpace             /* SET TABLESPACE */
	AT_SetRelOptions             /* SET (...) -- AM specific parameters */
	AT_ResetRelOptions           /* RESET (...) -- AM specific parameters */
	AT_ReplaceRelOptions         /* replace reloption list in its entirety */
	AT_EnableTrig                /* ENABLE TRIGGER name */
	AT_EnableAlwaysTrig          /* ENABLE ALWAYS TRIGGER name */
	AT_EnableReplicaTrig         /* ENABLE REPLICA TRIGGER name */
	AT_DisableTrig               /* DISABLE TRIGGER name */
	AT_EnableTrigAll             /* ENABLE TRIGGER ALL */
	AT_DisableTrigAll            /* DISABLE TRIGGER ALL */
	AT_EnableTrigUser            /* ENABLE TRIGGER USER */
	AT_DisableTrigUser           /* DISABLE TRIGGER USER */
	AT_EnableRule                /* ENABLE RULE name */
	AT_EnableAlwaysRule          /* ENABLE ALWAYS RULE name */
	AT_EnableReplicaRule         /* ENABLE REPLICA RULE name */
	AT_DisableRule               /* DISABLE RULE name */
	AT_AddInherit                /* INHERIT parent */
	AT_DropInherit               /* NO INHERIT parent */
	AT_AddOf                     /* OF <type_name> */
	AT_DropOf                    /* NOT OF */
	AT_ReplicaIdentity           /* REPLICA IDENTITY */
	AT_EnableRowSecurity         /* ENABLE ROW SECURITY */
	AT_DisableRowSecurity        /* DISABLE ROW SECURITY */
	AT_ForceRowSecurity          /* FORCE ROW SECURITY */
	AT_NoForceRowSecurity        /* NO FORCE ROW SECURITY */
	AT_GenericOptions            /* OPTIONS (...) */
	AT_AttachPartition           /* ATTACH PARTITION */
	AT_DetachPartition           /* DETACH PARTITION */
	AT_AddIdentity               /* ADD IDENTITY */
	AT_SetIdentity               /* SET identity column options */
	AT_DropIdentity              /* DROP IDENTITY */
)

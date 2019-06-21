// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/*-------------------------------------------------------------------------
 *
 * parsenodes.h
 *	  definitions for parse tree nodes
 *
 * Many of the node types used in parsetrees include a "location" field.
 * This is a byte (not character) offset in the original source text, to be
 * used for positioning an error cursor when there is an error related to
 * the node.  Access to the original source text is needed to make use of
 * the location.  At the topmost (statement) level, we also provide a
 * statement length, likewise measured in bytes, for convenience in
 * identifying statement boundaries in multi-statement source strings.
 *
 *
 * Portions Copyright (c) 1996-2017, PostgreSQL Global Development Group
 * Portions Copyright (c) 1994, Regents of the University of California
 *
 * src/include/nodes/parsenodes.h
 *
 *-------------------------------------------------------------------------
 */
type OverridingKind uint

const (
	OVERRIDING_NOT_SET OverridingKind = iota
	OVERRIDING_USER_VALUE
	OVERRIDING_SYSTEM_VALUE
)

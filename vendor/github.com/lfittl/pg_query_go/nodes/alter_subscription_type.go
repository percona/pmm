// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

type AlterSubscriptionType uint

const (
	ALTER_SUBSCRIPTION_OPTIONS AlterSubscriptionType = iota
	ALTER_SUBSCRIPTION_CONNECTION
	ALTER_SUBSCRIPTION_PUBLICATION
	ALTER_SUBSCRIPTION_REFRESH
	ALTER_SUBSCRIPTION_ENABLED
)

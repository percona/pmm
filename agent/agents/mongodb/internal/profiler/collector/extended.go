package collector

import "github.com/percona/percona-toolkit/src/go/mongolib/proto"

type ExtendedSystemProfile struct {
	proto.SystemProfile `bson:",inline"`
	PlanSummary         string `bson:"planSummary"`
}

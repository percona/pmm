package jobs

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/api/agentpb"
)

// MongoDBRestartJob implements Job for services that needs to be restarted after a successful restore.
// at the moment, this is only needed for restores from a physical backup,
// and it is only used to restart mongod and pbm-agents
type MongoDBRestartJob struct {
	id      string
	timeout time.Duration
	l       logrus.FieldLogger
	service agentpb.StartJobRequest_MongoDBRestartService_Service
}

func NewMongoDBRestartJob(id string, timeout time.Duration, service agentpb.StartJobRequest_MongoDBRestartService_Service) (*MongoDBRestartJob, error) {
	if service != agentpb.StartJobRequest_MongoDBRestartService_MONGOD && service != agentpb.StartJobRequest_MongoDBRestartService_PBM_AGENT {
		return nil, errors.Errorf("unsupported service '%s' specified for restart", service)
	}
	return &MongoDBRestartJob{
		id:      id,
		timeout: timeout,
		l:       logrus.WithFields(logrus.Fields{"id": id, "type": "mongodb_post_restore"}),
		service: service,
	}, nil
}

func (j *MongoDBRestartJob) ID() string {
	return j.id
}

func (j *MongoDBRestartJob) Type() JobType {
	return MongoDBPostRestore
}

func (j *MongoDBRestartJob) Timeout() time.Duration {
	return j.timeout
}

func (j *MongoDBRestartJob) Run(ctx context.Context, send Send) error {
	mongoServiceName := "mongod"
	pbmAgentServiceName := "pbm-agent"

	j.l.Info("restarting mongod after restore...")
	if err := startSystemctlService(ctx, mongoServiceName); err != nil {
		return errors.WithStack(err)
	}

	j.l.Info("restarting pbm-agent after restore...")
	if err := startSystemctlService(ctx, pbmAgentServiceName); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

j.l.Infof("%s successfully restarted", j.service)
send(&agentpb.JobResult{
JobId:     j.id,
Timestamp: timestamppb.Now(),
Result: &agentpb.JobResult_MongodbRestartService{
MongodbRestartService: &agentpb.JobResult_MongoDBRestartService{},
},
})
return nil
}

package jobs

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// MongoDBPostRestoreJob implements Job for actions that need to be take after restoring a MongoDB backup.
// at the moment, this is only needed for restores from a physical backup, and automate actions like:
// - restarting mongod and pbm-agents
type MongoDBPostRestoreJob struct {
	id      string
	timeout time.Duration
	l       logrus.FieldLogger
}

func NewMongoDBPostRestoreJob(id string, timeout time.Duration) (*MongoDBPostRestoreJob, error) {
	return &MongoDBPostRestoreJob{
		id:      id,
		timeout: timeout,
		l:       logrus.WithFields(logrus.Fields{"id": id, "type": "mongodb_post_restore"}),
	}, nil
}

func (j *MongoDBPostRestoreJob) ID() string {
	return j.id
}

func (j *MongoDBPostRestoreJob) Type() JobType {
	return MongoDBPostRestore
}

func (j *MongoDBPostRestoreJob) Timeout() time.Duration {
	return j.timeout
}

func (j *MongoDBPostRestoreJob) Run(ctx context.Context, send Send) error {
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

	j.l.Info("mongod and pbm-agent successfully restarted")
	return nil
}

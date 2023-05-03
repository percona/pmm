package highavailability

import (
	"github.com/hashicorp/memberlist"
	"github.com/sirupsen/logrus"
)

type eventDelegate struct {
	nodeCh chan *memberlist.Node
}

func (e *eventDelegate) NotifyJoin(node *memberlist.Node) {
	e.nodeCh <- node
}

func (e *eventDelegate) NotifyLeave(node *memberlist.Node) {
	e.nodeCh <- node
}

func (e *eventDelegate) NotifyUpdate(node *memberlist.Node) {
	logrus.Printf("NotifyUpdate: %v", node)
}

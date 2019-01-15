package receiver

import (
	"fmt"
	"io"

	collectorpb "github.com/Percona-Lab/qan-api/api/collector"
	"github.com/Percona-Lab/qan-api/models"
)

// Service implements gRPC service to communicate with agent.
type Service struct {
	qcm models.QueryClass
}

// NewService create new insstance of Service.
func NewService(qcm models.QueryClass) *Service {
	return &Service{qcm}
}

// DataInterchange implements rpc to exchange data between API and agent.
func (s *Service) DataInterchange(stream collectorpb.Agent_DataInterchangeServer) error {
	fmt.Println("Start...")
	for {
		agentMsg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("recved from agent: %+v", agentMsg)
		}
		err = s.qcm.Save(agentMsg)
		if err != nil {
			fmt.Printf("save error: %v \n", err)
			return fmt.Errorf("save error: %v", err)
		}
		savedAmount := len(agentMsg.QueryClass)
		fmt.Printf("Rcvd and saved %v QC\n", savedAmount)
		// look for msgs to be sent to client
		msg := collectorpb.ApiMessage{SavedAmount: uint32(savedAmount)}
		if err := stream.Send(&msg); err != nil {
			return err
		}
	}
}

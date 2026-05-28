package cronjob

import (
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/service"
)

type NodeConfigPushJob struct {
	service.NodeService
}

func NewNodeConfigPushJob() *NodeConfigPushJob {
	return &NodeConfigPushJob{}
}

func (s *NodeConfigPushJob) Run() {
	if err := s.NodeService.SyncAllRemoteNodeConfigs(); err != nil {
		logger.Warning("Node config push job failed: ", err)
	}
}

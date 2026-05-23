package cronjob

import (
	"encoding/json"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/service"
	"github.com/alireza0/s-ui/util"
)

type NodeSyncJob struct {
	service.NodeService
}

func NewNodeSyncJob() *NodeSyncJob {
	return &NodeSyncJob{}
}

func (s *NodeSyncJob) Run() {
	err := s.NodeService.SyncAllRemoteNodes()
	if err != nil {
		logger.Warning("Node sync job failed: ", err)
	}
	
	// Refresh public IP for local nodes with publicHostMode='public'
	err = s.refreshLocalNodePublicIP()
	if err != nil {
		logger.Warning("Local node public IP refresh failed: ", err)
	}
}

func (s *NodeSyncJob) refreshLocalNodePublicIP() error {
	db := database.GetDB()
	var localNodes []model.Node
	err := db.Model(model.Node{}).Where("type = ? and enabled = ? and public_host_mode = ?", "local", true, "public").Find(&localNodes).Error
	if err != nil {
		return err
	}
	
	if len(localNodes) == 0 {
		return nil
	}
	
	publicIp := util.GetPublicIP()
	if publicIp == "" {
		return nil
	}
	
	for i := range localNodes {
		node := &localNodes[i]
		// Update meta with public IP
		var meta map[string]interface{}
		if node.Meta != nil {
			if err := json.Unmarshal(node.Meta, &meta); err != nil {
				meta = make(map[string]interface{})
			}
		} else {
			meta = make(map[string]interface{})
		}
		meta["publicIp"] = publicIp
		
		metaBytes, err := json.Marshal(meta)
		if err != nil {
			continue
		}
		
		err = db.Model(&model.Node{}).Where("id = ?", node.Id).Update("meta", metaBytes).Error
		if err != nil {
			logger.Warning("Failed to update public IP for node ", node.Name, ": ", err)
		}
	}
	
	return nil
}
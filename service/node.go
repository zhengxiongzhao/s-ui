package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/util"
	"github.com/alireza0/s-ui/util/common"

	"gorm.io/gorm"
)

type NodeService struct {
	ConfigService
}

type agentResponse[T any] struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Obj     T      `json:"obj"`
}

type NodeInfoResponse struct {
	Success        bool     `json:"success"`
	Msg            string   `json:"msg"`
	NodeName       string   `json:"nodeName"`
	AgentVersion   string   `json:"agentVersion"`
	CoreRunning    bool     `json:"coreRunning"`
	CoreUptime     uint32   `json:"coreUptime"`
	Hostname       string   `json:"hostname"`
	Os             string   `json:"os"`
	Arch           string   `json:"arch"`
	Ips            []string `json:"ips"`
	PublicIp       string   `json:"publicIp"`
	LastConfigHash string   `json:"lastConfigHash"`
	LastDBHash     string   `json:"lastDBHash"`
	LastDBVersion  int      `json:"lastDBVersion"`
	LastApplyError string   `json:"lastApplyError"`
}

type NodeStatsResponse struct {
	Success bool        `json:"success"`
	Msg     string      `json:"msg"`
	Stats   []StatItem  `json:"stats"`
	Onlines OnlinesInfo `json:"onlines"`
}

type StatItem struct {
	Resource  string `json:"resource"`
	Tag       string `json:"tag"`
	Direction bool   `json:"direction"`
	Traffic   int64  `json:"traffic"`
}

type OnlinesInfo struct {
	Inbound  []string `json:"inbound,omitempty"`
	User     []string `json:"user,omitempty"`
	Outbound []string `json:"outbound,omitempty"`
}

type ApplyConfigRequest struct {
	NodeId  uint            `json:"node_id"`
	Version int             `json:"version"`
	Hash    string          `json:"hash"`
	Config  json.RawMessage `json:"config"`
}

type ApplyDatabaseRequest struct {
	NodeId   uint   `json:"node_id"`
	Version  int    `json:"version"`
	Hash     string `json:"hash"`
	Database []byte `json:"database"`
}

func NewNodeService() *NodeService {
	return &NodeService{}
}

func (s *NodeService) GetCurrentNode() (*model.Node, error) {
	db := database.GetDB()
	var nodes []model.Node
	if err := db.Find(&nodes).Error; err != nil {
		return nil, err
	}

	// 1. 优先使用本地记录的 Agent Node ID 精确匹配
	if config.IsAgent() {
		agentNodeID := config.GetAgentNodeID()
		if agentNodeID > 0 {
			for _, n := range nodes {
				if n.Id == agentNodeID {
					return &n, nil
				}
			}
		}
	}

	// 2. 备用方案（如 IP, Name, Token 精确匹配）
	// (a) 根据公网 IP 匹配
	publicIP := util.GetPublicIP()
	if publicIP != "" {
		for _, n := range nodes {
			if n.ApiHost == publicIP {
				return &n, nil
			}
		}
	}

	// (b) 根据 SUI_NODE_NAME 匹配
	if nodeName := config.GetNodeName(); nodeName != "" {
		for _, n := range nodes {
			if n.Name == nodeName {
				return &n, nil
			}
		}
	}

	// 3. 本地节点备选：如果只有一个节点，默认是自身
	if len(nodes) == 1 {
		return &nodes[0], nil
	}

	return nil, fmt.Errorf("current agent node not matched in database")
}

func (s *NodeService) GetAll() ([]model.Node, error) {
	db := database.GetDB()
	var nodes []model.Node
	err := db.Model(model.Node{}).Order("sort ASC, id ASC").Find(&nodes).Error
	return nodes, err
}

func (s *NodeService) Get(id uint) (*model.Node, error) {
	db := database.GetDB()
	var node model.Node
	err := db.Model(model.Node{}).Where("id = ?", id).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (s *NodeService) GetByName(name string) (*model.Node, error) {
	db := database.GetDB()
	var node model.Node
	err := db.Model(model.Node{}).Where("name = ?", name).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (s *NodeService) Save(tx *gorm.DB, act string, data json.RawMessage) error {
	var node model.Node
	err := json.Unmarshal(data, &node)
	if err != nil {
		return err
	}

	switch act {
	case "new":
		return tx.Create(&node).Error
	case "edit":
		return tx.Save(&node).Error
	case "del":
		return tx.Where("id = ?", node.Id).Delete(model.Node{}).Error
	default:
		return fmt.Errorf("unknown action: %s", act)
	}
}

func (s *NodeService) GenerateToken() string {
	token := common.Random(32)
	return token
}

func (s *NodeService) RotateToken(id uint) (string, error) {
	db := database.GetDB()
	token := s.GenerateToken()
	err := db.Model(model.Node{}).Where("id = ?", id).Update("token", token).Error
	return token, err
}

func (s *NodeService) ToggleEnabled(id uint) (bool, error) {
	db := database.GetDB()
	var node model.Node
	err := db.Where("id = ?", id).First(&node).Error
	if err != nil {
		return false, err
	}
	newEnabled := !node.Enabled
	err = db.Model(&node).Update("enabled", newEnabled).Error
	return newEnabled, err
}

// ToggleAutoSync 切换节点的自动同步状态
func (s *NodeService) ToggleAutoSync(id uint) (bool, error) {
	db := database.GetDB()
	var node model.Node
	err := db.Where("id = ?", id).First(&node).Error
	if err != nil {
		return false, err
	}
	newAutoSync := !node.AutoSync
	err = db.Model(&node).Update("auto_sync", newAutoSync).Error
	return newAutoSync, err
}

// UpdateNodeSorts 批量更新节点排序
func (s *NodeService) UpdateNodeSorts(sorts []NodeSortItem) error {
	db := database.GetDB()
	tx := db.Begin()
	defer func() {
		tx.Rollback()
	}()

	for _, item := range sorts {
		err := tx.Model(&model.Node{}).Where("id = ?", item.Id).Update("sort", item.Sort).Error
		if err != nil {
			return err
		}
	}
	return tx.Commit().Error
}

type NodeSortItem struct {
	Id   uint `json:"id"`
	Sort int  `json:"sort"`
}

func (s *NodeService) GetEnabledNodes() ([]model.Node, error) {
	db := database.GetDB()
	var nodes []model.Node
	err := db.Model(model.Node{}).Where("enabled = ?", true).Order("sort ASC, id ASC").Find(&nodes).Error
	return nodes, err
}

// CallAgent makes an HTTP request to the agent
func (s *NodeService) CallAgent(node *model.Node, method string, path string, body interface{}) ([]byte, error) {
	apiUrl := node.GetApiBaseUrl()
	if apiUrl == "" {
		return nil, fmt.Errorf("node %s has no API configured", node.Name)
	}
	if node.Token == "" {
		return nil, fmt.Errorf("node %s has no token configured", node.Name)
	}

	url := strings.TrimRight(apiUrl, "/") + path
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+node.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("agent returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetNodeInfo fetches info from the agent
func (s *NodeService) GetNodeInfo(node *model.Node) (*NodeInfoResponse, error) {
	respBody, err := s.CallAgent(node, "GET", "/info", nil)
	if err != nil {
		return nil, err
	}

	var wrapper agentResponse[NodeInfoResponse]
	err = json.Unmarshal(respBody, &wrapper)
	if err != nil {
		return nil, err
	}

	info := wrapper.Obj
	info.Success = wrapper.Success
	info.Msg = wrapper.Msg
	return &info, nil
}

// ApplyConfig sends config to the agent
func (s *NodeService) ApplyConfig(node *model.Node, config json.RawMessage) error {
	hash := s.computeHash(config)
	req := ApplyConfigRequest{
		NodeId:  node.Id,
		Version: int(time.Now().Unix()),
		Hash:    hash,
		Config:  config,
	}

	return s.callAgentAction(node, "POST", "/apply-config", req, "apply config")
}

// ApplyDatabase sends a database snapshot to the agent.
func (s *NodeService) ApplyDatabase(node *model.Node, snapshot []byte) error {
	version, err := s.SettingService.GetNodeConfigVersion()
	if err != nil {
		return err
	}
	return s.ApplyDatabaseWithVersion(node, snapshot, version)
}

func (s *NodeService) ApplyDatabaseWithVersion(node *model.Node, snapshot []byte, version int) error {
	hash := s.computeHash(snapshot)
	req := ApplyDatabaseRequest{
		NodeId:   node.Id,
		Version:  version,
		Hash:     hash,
		Database: snapshot,
	}

	return s.callAgentAction(node, "POST", "/apply-database", req, "apply database")
}

func (s *NodeService) callAgentAction(node *model.Node, method string, path string, body interface{}, action string) error {
	respBody, err := s.CallAgent(node, method, path, body)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return err
	}

	if success, ok := result["success"].(bool); !ok || !success {
		msg, _ := result["msg"].(string)
		return fmt.Errorf("%s failed: %s", action, msg)
	}

	return nil
}

// StartCore tells the agent to start core
func (s *NodeService) StartCore(node *model.Node) error {
	respBody, err := s.CallAgent(node, "POST", "/start-core", nil)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return err
	}

	if success, ok := result["success"].(bool); !ok || !success {
		msg, _ := result["msg"].(string)
		return fmt.Errorf("start core failed: %s", msg)
	}

	return nil
}

// StopCore tells the agent to stop core
func (s *NodeService) StopCore(node *model.Node) error {
	respBody, err := s.CallAgent(node, "POST", "/stop-core", nil)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return err
	}

	if success, ok := result["success"].(bool); !ok || !success {
		msg, _ := result["msg"].(string)
		return fmt.Errorf("stop core failed: %s", msg)
	}

	return nil
}

// RestartCore tells the agent to restart core
func (s *NodeService) RestartCore(node *model.Node) error {
	respBody, err := s.CallAgent(node, "POST", "/restart-core", nil)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return err
	}

	if success, ok := result["success"].(bool); !ok || !success {
		msg, _ := result["msg"].(string)
		return fmt.Errorf("restart core failed: %s", msg)
	}

	return nil
}

// GetNodeStats fetches stats from the agent
func (s *NodeService) GetNodeStats(node *model.Node) (*NodeStatsResponse, error) {
	respBody, err := s.CallAgent(node, "GET", "/stats", nil)
	if err != nil {
		return nil, err
	}

	var wrapper agentResponse[NodeStatsResponse]
	err = json.Unmarshal(respBody, &wrapper)
	if err != nil {
		return nil, err
	}

	stats := wrapper.Obj
	stats.Success = wrapper.Success
	stats.Msg = wrapper.Msg
	return &stats, nil
}

// UpdateNodeStatus updates node status in database
func (s *NodeService) UpdateNodeStatus(nodeId uint, status model.NodeStatus, lastError string, meta json.RawMessage) error {
	db := database.GetDB()
	updates := map[string]interface{}{
		"status":     status,
		"last_seen":  time.Now().Unix(),
		"last_error": lastError,
	}
	if meta != nil {
		updates["meta"] = meta
	}
	return db.Model(model.Node{}).Where("id = ?", nodeId).Updates(updates).Error
}

// SyncNodeInfo fetches info from agent and updates database
func (s *NodeService) SyncNodeInfo(node *model.Node) error {
	info, err := s.GetNodeInfo(node)
	if err != nil {
		s.UpdateNodeStatus(node.Id, model.NodeStatusOffline, err.Error(), nil)
		return err
	}

	status := model.NodeStatusOnline
	if info.LastApplyError != "" {
		status = model.NodeStatusError
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"hostname":       info.Hostname,
		"os":             info.Os,
		"arch":           info.Arch,
		"ips":            info.Ips,
		"publicIp":       info.PublicIp,
		"agentVersion":   info.AgentVersion,
		"coreRunning":    info.CoreRunning,
		"coreUptime":     info.CoreUptime,
		"lastConfigHash": info.LastConfigHash,
		"lastDBHash":     info.LastDBHash,
		"lastDBVersion":  info.LastDBVersion,
	})

	return s.UpdateNodeStatus(node.Id, status, info.LastApplyError, meta)
}

// SyncNodeStats fetches stats from agent and saves to database
func (s *NodeService) SyncNodeStats(node *model.Node) error {
	statsResp, err := s.GetNodeStats(node)
	if err != nil {
		return err
	}

	if !statsResp.Success {
		return fmt.Errorf("failed to get stats: %s", statsResp.Msg)
	}

	db := database.GetDB()
	tx := db.Begin()
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	// Save stats
	for _, stat := range statsResp.Stats {
		if stat.Resource == "user" {
			if stat.Direction {
				err = tx.Model(model.Client{}).Where("name = ?", stat.Tag).
					UpdateColumn("up", gorm.Expr("up + ?", stat.Traffic)).Error
			} else {
				err = tx.Model(model.Client{}).Where("name = ?", stat.Tag).
					UpdateColumn("down", gorm.Expr("down + ?", stat.Traffic)).Error
			}
			if err != nil {
				return err
			}
		}

		// Save to stats table
		statsRecord := model.Stats{
			DateTime:  time.Now().Unix(),
			Resource:  stat.Resource,
			Tag:       stat.Tag,
			Direction: stat.Direction,
			Traffic:   stat.Traffic,
			NodeId:    node.Id,
		}
		err = tx.Create(&statsRecord).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// GetNodeConfig generates config for a specific node
func (s *NodeService) GetNodeConfig(nodeId uint) ([]byte, error) {
	db := database.GetDB()

	// Get base config
	baseConfig, err := s.SettingService.GetConfig()
	if err != nil {
		return nil, err
	}

	singboxConfig := SingBoxConfig{}
	err = json.Unmarshal([]byte(baseConfig), &singboxConfig)
	if err != nil {
		return nil, err
	}

	// Get inbounds for this node
	var inbounds []*model.Inbound
	err = db.Model(model.Inbound{}).Preload("Tls").Where("node_id = ?", nodeId).Find(&inbounds).Error
	if err != nil {
		return nil, err
	}
	for _, inbound := range inbounds {
		inboundJson, err := inbound.MarshalJSON()
		if err != nil {
			return nil, err
		}
		inboundJson, err = s.InboundService.addUsers(db, inboundJson, inbound.Id, inbound.Type)
		if err != nil {
			return nil, err
		}
		singboxConfig.Inbounds = append(singboxConfig.Inbounds, inboundJson)
	}

	// Get outbounds for this node
	var outbounds []*model.Outbound
	err = db.Model(model.Outbound{}).Where("node_id = ?", nodeId).Find(&outbounds).Error
	if err != nil {
		return nil, err
	}
	for _, outbound := range outbounds {
		outboundJson, err := outbound.MarshalJSON()
		if err != nil {
			return nil, err
		}
		singboxConfig.Outbounds = append(singboxConfig.Outbounds, outboundJson)
	}

	// Get services for this node
	var services []*model.Service
	err = db.Model(model.Service{}).Preload("Tls").Where("node_id = ?", nodeId).Find(&services).Error
	if err != nil {
		return nil, err
	}
	for _, service := range services {
		serviceJson, err := service.MarshalJSON()
		if err != nil {
			return nil, err
		}
		singboxConfig.Services = append(singboxConfig.Services, serviceJson)
	}

	// Get endpoints for this node
	var endpoints []*model.Endpoint
	err = db.Model(model.Endpoint{}).Where("node_id = ?", nodeId).Find(&endpoints).Error
	if err != nil {
		return nil, err
	}
	for _, endpoint := range endpoints {
		endpointJson, err := endpoint.MarshalJSON()
		if err != nil {
			return nil, err
		}
		singboxConfig.Endpoints = append(singboxConfig.Endpoints, endpointJson)
	}

	rawConfig, err := json.MarshalIndent(singboxConfig, "", "  ")
	if err != nil {
		return nil, err
	}

	return rawConfig, nil
}

// SyncNodeConfig generates and sends config to agent
func (s *NodeService) SyncNodeConfig(node *model.Node) error {
	if node.Type == model.NodeTypeRemote {
		snapshot, err := database.ExportDB(true, true)
		if err != nil {
			return err
		}
		return s.ApplyDatabase(node, snapshot)
	}

	config, err := s.GetNodeConfig(node.Id)
	if err != nil {
		return err
	}

	return s.ApplyConfig(node, json.RawMessage(config))
}

func (s *NodeService) computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// compareVersions compares two version strings (e.g., "1.5.3")
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	v1Parts := parseVersion(v1)
	v2Parts := parseVersion(v2)

	maxLen := len(v1Parts)
	if len(v2Parts) > maxLen {
		maxLen = len(v2Parts)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int
		if i < len(v1Parts) {
			p1 = v1Parts[i]
		}
		if i < len(v2Parts) {
			p2 = v2Parts[i]
		}
		if p1 < p2 {
			return -1
		}
		if p1 > p2 {
			return 1
		}
	}
	return 0
}

// parseVersion parses a version string like "1.5.3" into []int{1, 5, 3}
func parseVersion(version string) []int {
	parts := strings.Split(version, ".")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		// Remove any non-numeric suffix (e.g., "3-beta" -> "3")
		numStr := strings.TrimFunc(part, func(r rune) bool {
			return r < '0' || r > '9'
		})
		if numStr == "" {
			continue
		}
		// Extract leading digits
		digits := ""
		for _, c := range numStr {
			if c >= '0' && c <= '9' {
				digits += string(c)
			} else {
				break
			}
		}
		if digits != "" {
			num, _ := strconv.Atoi(digits)
			result = append(result, num)
		}
	}
	return result
}

// getAgentVersion extracts agent version from node meta
func getAgentVersion(node *model.Node) string {
	if node.Meta == nil {
		return ""
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(node.Meta, &meta); err != nil {
		return ""
	}
	if version, ok := meta["agentVersion"].(string); ok {
		return version
	}
	return ""
}

// SyncAllRemoteNodes syncs info and stats for all remote nodes with auto_sync enabled
func (s *NodeService) SyncAllRemoteNodes() error {
	db := database.GetDB()
	var nodes []model.Node
	err := db.Model(model.Node{}).Where("type = ? AND enabled = ? AND auto_sync = ?", model.NodeTypeRemote, true, true).Find(&nodes).Error
	if err != nil {
		return err
	}

	panelVersion := config.GetVersion()

	for _, node := range nodes {
		// Sync info
		err := s.SyncNodeInfo(&node)
		if err != nil {
			logger.Warning("failed to sync node info for", node.Name, ":", err)
			continue
		}

		// Check agent version before syncing stats
		agentVersion := getAgentVersion(&node)
		if agentVersion != "" && compareVersions(agentVersion, panelVersion) < 0 {
			logger.Warning("skip sync stats for node", node.Name, ": agent version", agentVersion, "is lower than panel version", panelVersion)
			continue
		}

		// Sync stats
		err = s.SyncNodeStats(&node)
		if err != nil {
			logger.Warning("failed to sync node stats for", node.Name, ":", err)
		}
	}

	return nil
}

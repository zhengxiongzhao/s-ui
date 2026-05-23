package model

import (
	"encoding/json"
	"fmt"
)

type NodeType string

const (
	NodeTypeLocal  NodeType = "local"
	NodeTypeRemote NodeType = "remote"
)

type NodeStatus string

const (
	NodeStatusUnknown  NodeStatus = "unknown"
	NodeStatusOnline   NodeStatus = "online"
	NodeStatusOffline  NodeStatus = "offline"
	NodeStatusDisabled NodeStatus = "disabled"
	NodeStatusError    NodeStatus = "error"
)

type Node struct {
	Id            uint            `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Name          string          `json:"name" form:"name" gorm:"unique;not null"`
	Type          NodeType        `json:"type" form:"type" gorm:"not null;default:'local'"`
	Enabled       bool            `json:"enabled" form:"enabled" gorm:"default:true;not null"`
	// Agent API 连接配置（替代原 ApiBaseUrl）
	ApiHost       string          `json:"apiHost" form:"apiHost"`
	ApiPort       int             `json:"apiPort" form:"apiPort" gorm:"default:2097"`
	ApiScheme     string          `json:"apiScheme" form:"apiScheme" gorm:"default:'http'"`
	ApiBaseUrl    string          `json:"apiBaseUrl" form:"apiBaseUrl"` // 向后兼容，迁移后弃用
	Token         string          `json:"token" form:"token"`
	// 公网地址配置
	PublicHostMode  string          `json:"publicHostMode" form:"publicHostMode" gorm:"default:'agent'"`
	PublicHost      string          `json:"publicHost" form:"publicHost"`
	PublicPortMap   json.RawMessage `json:"publicPortMap" form:"publicPortMap"`
	// 状态
	Status        NodeStatus      `json:"status" form:"status" gorm:"default:'unknown'"`
	LastSeen      int64           `json:"lastSeen" form:"lastSeen"`
	LastError     string          `json:"lastError" form:"lastError"`
	Meta          json.RawMessage `json:"meta" form:"meta"`
}

// GetApiBaseUrl 拼接 Agent API 完整 URL
func (n *Node) GetApiBaseUrl() string {
	// 如果有旧的 ApiBaseUrl，优先使用（向后兼容）
	if n.ApiBaseUrl != "" {
		return n.ApiBaseUrl
	}
	scheme := n.ApiScheme
	if scheme == "" {
		scheme = "http"
	}
	port := n.ApiPort
	if port == 0 {
		port = 2097
	}
	return fmt.Sprintf("%s://%s:%d/agent/api", scheme, n.ApiHost, port)
}

// GetEffectiveHost 获取节点用于订阅的有效公网地址
func (n *Node) GetEffectiveHost() string {
	switch n.PublicHostMode {
	case "agent":
		return n.ApiHost
	case "custom":
		return n.PublicHost
	case "public":
		var meta map[string]interface{}
		json.Unmarshal(n.Meta, &meta)
		if ip, ok := meta["publicIp"].(string); ok && ip != "" {
			return ip
		}
		return ""
	case "local":
		var meta map[string]interface{}
		json.Unmarshal(n.Meta, &meta)
		if ips, ok := meta["ips"].([]interface{}); ok && len(ips) > 0 {
			if ip, ok := ips[0].(string); ok {
				return ip
			}
		}
		return ""
	default:
		// 默认回退到 ApiHost
		return n.ApiHost
	}
}

func (n *Node) MarshalFull() (*map[string]interface{}, error) {
	result := map[string]interface{}{
		"id":             n.Id,
		"name":           n.Name,
		"type":           n.Type,
		"enabled":        n.Enabled,
		"apiHost":        n.ApiHost,
		"apiPort":        n.ApiPort,
		"apiScheme":      n.ApiScheme,
		"token":          n.Token,
		"publicHostMode": n.PublicHostMode,
		"publicHost":     n.PublicHost,
		"publicPortMap":  n.PublicPortMap,
		"status":         n.Status,
		"lastSeen":       n.LastSeen,
		"lastError":      n.LastError,
		"meta":           n.Meta,
	}
	return &result, nil
}

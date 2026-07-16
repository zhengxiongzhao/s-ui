package sub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/service"
	"github.com/alireza0/s-ui/util"
)

type SubService struct {
	service.SettingService
	LinkService
}

func (s *SubService) GetSubs(subId string) (*string, []string, error) {
	var err error

	client, err := s.getClientBySubId(subId)
	if err != nil {
		return nil, nil, err
	}

	clientInfo := ""
	subShowInfo, _ := s.SettingService.GetSubShowInfo()
	if subShowInfo {
		clientInfo = s.getClientInfo(client)
	}

	clientLinks := json.RawMessage(client.Links)
	if config.IsAgent() {
		filteredLinks, err := s.filterLocalLinksToCurrentNode(&clientLinks)
		if err == nil {
			clientLinks = *filteredLinks
		}
	}
	sortedLinks := s.sortLocalLinksByNode(&clientLinks)
	linksArray := s.LinkService.GetLinks(sortedLinks, "all", clientInfo)
	result := strings.Join(linksArray, "\n")

	headers := s.getClientHeaders(client)

	subEncode, _ := s.SettingService.GetSubEncode()
	if subEncode {
		result = base64.StdEncoding.EncodeToString([]byte(result))
	}

	return &result, headers, nil
}

// sortLocalLinksByNode 将 local 类型的链接按其所属节点的 sort 字段重新排序，
// 非 local 链接（external、sub）保持原始顺序追加在末尾。
func (s *SubService) sortLocalLinksByNode(linkJson *json.RawMessage) *json.RawMessage {
	var links []Link
	if err := json.Unmarshal(*linkJson, &links); err != nil {
		return linkJson
	}

	// 检查是否存在 local 类型链接，不存在则直接返回
	hasLocal := false
	for _, l := range links {
		if l.Type == "local" {
			hasLocal = true
			break
		}
	}
	if !hasLocal {
		return linkJson
	}

	// 查询 inbound tag → node sort 映射
	type tagSort struct {
		Tag  string
		Sort int
	}
	db := database.GetDB()
	var tagSorts []tagSort
	err := db.Model(model.Inbound{}).
		Select("inbounds.tag as tag, COALESCE(nodes.sort, 999999) as sort").
		Joins("LEFT JOIN nodes ON nodes.id = inbounds.node_id").
		Scan(&tagSorts).Error
	if err != nil {
		return linkJson
	}
	sortMap := make(map[string]int, len(tagSorts))
	for _, ts := range tagSorts {
		sortMap[ts.Tag] = ts.Sort
	}

	// 分离 local 和非 local 链接
	var localLinks []Link
	var otherLinks []Link
	for _, l := range links {
		if l.Type == "local" {
			localLinks = append(localLinks, l)
		} else {
			otherLinks = append(otherLinks, l)
		}
	}

	// 稳定排序 local 链接
	sort.SliceStable(localLinks, func(i, j int) bool {
		si, ok := sortMap[localLinks[i].Remark]
		if !ok {
			si = 999999
		}
		sj, ok := sortMap[localLinks[j].Remark]
		if !ok {
			sj = 999999
		}
		return si < sj
	})

	// 重组
	result := append(localLinks, otherLinks...)
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return linkJson
	}
	resultRaw := json.RawMessage(resultBytes)
	return &resultRaw
}

func (s *SubService) filterLocalLinksToCurrentNode(linkJson *json.RawMessage) (*json.RawMessage, error) {
	nodeService := &service.NodeService{}
	currentNode, err := nodeService.GetCurrentNode()
	if err != nil || currentNode == nil {
		if config.IsAgent() {
			emptyBytes, _ := json.Marshal([]Link{})
			raw := json.RawMessage(emptyBytes)
			return &raw, nil
		}
		return linkJson, err
	}

	var links []Link
	if err := json.Unmarshal(*linkJson, &links); err != nil {
		return linkJson, err
	}

	db := database.GetDB()
	// 查询当前节点专属入站 + 全局入站(NodeId <= 1)
	var inbounds []model.Inbound
	if err := db.Where("node_id = ? OR node_id <= 1", currentNode.Id).Find(&inbounds).Error; err != nil {
		return linkJson, err
	}

	inboundMap := make(map[string]*model.Inbound)
	for i := range inbounds {
		inboundMap[inbounds[i].Tag] = &inbounds[i]
	}

	var portMap map[string]interface{}
	if currentNode.PublicPortMap != nil {
		_ = json.Unmarshal(currentNode.PublicPortMap, &portMap)
	}

	var filteredLinks []Link
	for _, l := range links {
		if l.Type == "local" {
			if inbound, ok := inboundMap[l.Remark]; ok {
				targetPort := 0
				// 全局入站(NodeId <= 1)不做端口映射重写
				if inbound.NodeId > 1 && portMap != nil {
					if mappedPort, ok := portMap[inbound.Tag]; ok {
						if portFloat, ok := mappedPort.(float64); ok {
							targetPort = int(portFloat)
						}
					}
				}
				// 全局入站(NodeId <= 1)不重写 host，保留原始链接地址
				if inbound.NodeId > 1 {
					effectiveHost := currentNode.GetEffectiveHost()
					if effectiveHost != "" {
						l.Uri = rewriteLinkUri(l.Uri, effectiveHost, targetPort)
					}
				}
				filteredLinks = append(filteredLinks, l)
			}
		} else {
			filteredLinks = append(filteredLinks, l)
		}
	}

	newBytes, err := json.Marshal(filteredLinks)
	if err != nil {
		return linkJson, err
	}
	raw := json.RawMessage(newBytes)
	return &raw, nil
}

// decodeBase64Safe decodes base64 string handling omitted padding
func decodeBase64Safe(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if l := len(s) % 4; l > 0 {
		s += strings.Repeat("=", 4-l)
	}
	return base64.URLEncoding.DecodeString(s)
}

func rewriteLinkUri(uri string, host string, targetPort int) string {
	if strings.HasPrefix(uri, "vmess://") {
		b64Data := strings.TrimPrefix(uri, "vmess://")
		// Try standard first, fallback to URLEncoding if needed
		decoded, err := base64.StdEncoding.DecodeString(b64Data)
		if err != nil {
			decoded, err = decodeBase64Safe(b64Data)
			if err != nil {
				return uri
			}
		}
		var config map[string]interface{}
		if err := json.Unmarshal(decoded, &config); err != nil {
			return uri
		}
		config["add"] = host
		if targetPort > 0 {
			config["port"] = targetPort
		}
		newJSON, err := json.Marshal(config)
		if err != nil {
			return uri
		}
		return "vmess://" + base64.StdEncoding.EncodeToString(newJSON)
	}

	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	originalHost := u.Host
	_, p, err := net.SplitHostPort(originalHost)
	if err != nil {
		p = ""
	}

	newPort := p
	if targetPort > 0 {
		newPort = strconv.Itoa(targetPort)
	}

	if newPort != "" {
		u.Host = net.JoinHostPort(host, newPort)
	} else {
		u.Host = host
	}

	return u.String()
}

func (j *SubService) getClientBySubId(subId string) (*model.Client, error) {
	db := database.GetDB()
	client := &model.Client{}
	err := db.Model(model.Client{}).Where("enable = true and name = ?", subId).First(client).Error
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (s *SubService) getClientHeaders(client *model.Client) []string {
	updateInterval, _ := s.SettingService.GetSubUpdates()
	return util.GetHeaders(client, updateInterval)
}

func (s *SubService) getClientInfo(c *model.Client) string {
	now := time.Now().Unix()

	var result []string
	if vol := c.Volume - (c.Up + c.Down); vol > 0 {
		result = append(result, fmt.Sprintf("%s%s", s.formatTraffic(vol), "📊"))
	}
	if c.Expiry > 0 {
		result = append(result, fmt.Sprintf("%d%s⏳", (c.Expiry-now)/86400, "Days"))
	}
	if len(result) > 0 {
		return " " + strings.Join(result, " ")
	} else {
		return " ♾"
	}
}

func (s *SubService) formatTraffic(trafficBytes int64) string {
	if trafficBytes < 1024 {
		return fmt.Sprintf("%.2fB", float64(trafficBytes)/float64(1))
	} else if trafficBytes < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB", float64(trafficBytes)/float64(1024))
	} else if trafficBytes < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB", float64(trafficBytes)/float64(1024*1024))
	} else if trafficBytes < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB", float64(trafficBytes)/float64(1024*1024*1024))
	} else if trafficBytes < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fTB", float64(trafficBytes)/float64(1024*1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2fEB", float64(trafficBytes)/float64(1024*1024*1024*1024*1024))
	}
}

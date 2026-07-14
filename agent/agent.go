package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/core"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/service"
	"github.com/alireza0/s-ui/sub"
	"github.com/alireza0/s-ui/util"
	"github.com/alireza0/s-ui/web"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Agent struct {
	core          *core.Core
	httpServer    *http.Server
	listener      net.Listener
	ctx           context.Context
	cancel        context.CancelFunc
	configCache   []byte
	configHash    string
	configVersion int
	dbHash        string
	dbVersion     int
	lastError     string
	mu            sync.RWMutex
	webServer     *web.Server
	subServer     *sub.Server
}

type AgentInfo struct {
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

type ApplyConfigRequest struct {
	Version int             `json:"version"`
	Hash    string          `json:"hash"`
	Config  json.RawMessage `json:"config"`
}

type ApplyDatabaseRequest struct {
	Version  int    `json:"version"`
	Hash     string `json:"hash"`
	Database []byte `json:"database"`
}

type StatsResponse struct {
	Stats   []StatItem `json:"stats"`
	Onlines Onlines    `json:"onlines"`
}

type StatItem struct {
	Resource  string `json:"resource"`
	Tag       string `json:"tag"`
	Direction bool   `json:"direction"`
	Traffic   int64  `json:"traffic"`
}

type Onlines struct {
	Inbound  []string `json:"inbound,omitempty"`
	User     []string `json:"user,omitempty"`
	Outbound []string `json:"outbound,omitempty"`
}

func NewAgent() *Agent {
	ctx, cancel := context.WithCancel(context.Background())
	return &Agent{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (a *Agent) Init() error {
	logger.Info("Agent initializing...")
	logger.Info("Node name:", config.GetNodeName())
	logger.Info("Agent API will listen on:", config.GetAgentListen(), ":", config.GetAgentPort())

	// Create cache directory
	cacheDir := config.GetAgentCacheDir()
	err := os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	// Load cached config if exists
	cachedConfigPath := filepath.Join(cacheDir, "config.json")
	if _, err := os.Stat(cachedConfigPath); err == nil {
		a.configCache, err = os.ReadFile(cachedConfigPath)
		if err != nil {
			logger.Warning("failed to read cached config:", err)
		} else {
			a.configHash = a.computeHash(a.configCache)
			logger.Info("Loaded cached config, hash:", a.configHash)
		}
	}

	// Initialize core
	a.core = core.NewCore()

	// Initialize web and sub servers
	a.webServer = web.NewServer()
	a.subServer = sub.NewServer()

	return nil
}

func (a *Agent) Start() error {
	// Start HTTP server for Agent API
	err := a.startHTTPServer()
	if err != nil {
		return err
	}

	// Start web server if enabled
	if config.GetEnableWeb() {
		err = a.webServer.Start()
		if err != nil {
			return err
		}
		logger.Info("Web server started")
	} else {
		logger.Info("Web server disabled by SUI_ENABLE_WEB=false")
	}

	// Start sub server if enabled
	if config.GetEnableSub() {
		err = a.subServer.Start()
		if err != nil {
			return err
		}
		logger.Info("Sub server started")
	} else {
		logger.Info("Sub server disabled by SUI_ENABLE_SUB=false")
	}

	// Try to start core with cached config
	if len(a.configCache) > 0 {
		err = a.startCore()
		if err != nil {
			logger.Warning("failed to start core with cached config:", err)
			a.lastError = err.Error()
		}
	}

	logger.Info("Agent started successfully")
	return nil
}

func (a *Agent) Stop() {
	logger.Info("Agent stopping...")

	if a.httpServer != nil {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
		a.httpServer.Shutdown(shutdownCtx)
		cancelShutdown()
	}

	// Stop sub server if enabled
	if config.GetEnableSub() && a.subServer != nil {
		err := a.subServer.Stop()
		if err != nil {
			logger.Warning("stop Sub Server err:", err)
		}
	}

	// Stop web server if enabled
	if config.GetEnableWeb() && a.webServer != nil {
		err := a.webServer.Stop()
		if err != nil {
			logger.Warning("stop Web Server err:", err)
		}
	}

	if a.core != nil {
		a.core.Stop()
	}

	a.cancel()
	logger.Info("Agent stopped")
}

func (a *Agent) startHTTPServer() error {
	if config.IsDebug() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.DefaultWriter = io.Discard
		gin.DefaultErrorWriter = io.Discard
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.Default()

	// Auth middleware
	engine.Use(a.authMiddleware())

	// Agent API routes
	g := engine.Group("/agent/api")
	g.GET("/info", a.handleInfo)
	g.POST("/apply-config", a.handleApplyConfig)
	g.POST("/apply-database", a.handleApplyDatabase)
	g.POST("/start-core", a.handleStartCore)
	g.POST("/stop-core", a.handleStopCore)
	g.POST("/restart-core", a.handleRestartCore)
	g.GET("/stats", a.handleStats)
	g.GET("/logs", a.handleLogs)

	listenAddr := net.JoinHostPort(config.GetAgentListen(), strconv.Itoa(config.GetAgentPort()))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", listenAddr, err)
	}

	a.listener = listener
	a.httpServer = &http.Server{
		Handler: engine,
	}

	logger.Info("Agent API server running on", listener.Addr())

	go func() {
		a.httpServer.Serve(listener)
	}()

	return nil
}

func (a *Agent) authMiddleware() gin.HandlerFunc {
	expectedToken := config.GetNodeToken()

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			jsonError(c, http.StatusUnauthorized, "missing Authorization header")
			c.Abort()
			return
		}

		// Check Bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			jsonError(c, http.StatusUnauthorized, "invalid Authorization format")
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != expectedToken {
			jsonError(c, http.StatusUnauthorized, "invalid token")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (a *Agent) handleInfo(c *gin.Context) {
	info := a.getInfo()
	jsonObj(c, info, nil)
}

func (a *Agent) handleApplyConfig(c *gin.Context) {
	var req ApplyConfigRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// Validate config JSON
	if len(req.Config) == 0 {
		jsonError(c, http.StatusBadRequest, "empty config")
		return
	}

	// Compute hash if not provided
	if req.Hash == "" {
		req.Hash = a.computeHash(req.Config)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Save config to cache
	cacheDir := config.GetAgentCacheDir()
	configPath := filepath.Join(cacheDir, "config.json")
	err = os.WriteFile(configPath, req.Config, 0644)
	if err != nil {
		a.lastError = err.Error()
		jsonError(c, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	a.configCache = req.Config
	a.configHash = req.Hash
	a.configVersion = req.Version

	// Restart core with new config
	err = a.restartCoreInternal()
	if err != nil {
		a.lastError = err.Error()
		jsonError(c, http.StatusInternalServerError, "failed to apply config: "+err.Error())
		return
	}

	a.lastError = ""
	logger.Info("Config applied successfully, version:", req.Version, "hash:", req.Hash)
	jsonMsg(c, "config applied", nil)
}

func (a *Agent) handleApplyDatabase(c *gin.Context) {
	var req ApplyDatabaseRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if len(req.Database) == 0 {
		jsonError(c, http.StatusBadRequest, "empty database")
		return
	}

	if req.Hash == "" {
		req.Hash = a.computeHash(req.Database)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	cacheDir := config.GetAgentCacheDir()
	dbPath := filepath.Join(cacheDir, "panel.db")
	if err = a.saveDatabaseSnapshot(dbPath, req.Database); err != nil {
		a.lastError = err.Error()
		jsonError(c, http.StatusInternalServerError, "failed to save database: "+err.Error())
		return
	}

	rawConfig, err := a.generateConfigFromDatabase(dbPath)
	if err != nil {
		a.lastError = err.Error()
		jsonError(c, http.StatusInternalServerError, "failed to generate config from database: "+err.Error())
		return
	}

	configPath := filepath.Join(cacheDir, "config.json")
	if err = os.WriteFile(configPath, rawConfig, 0644); err != nil {
		a.lastError = err.Error()
		jsonError(c, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	a.configCache = rawConfig
	a.configHash = a.computeHash(rawConfig)
	a.configVersion = req.Version
	a.dbHash = req.Hash
	a.dbVersion = req.Version

	if err = a.restartCoreInternal(); err != nil {
		a.lastError = err.Error()
		jsonError(c, http.StatusInternalServerError, "failed to apply database: "+err.Error())
		return
	}

	a.lastError = ""
	logger.Info("Database applied successfully, version:", req.Version, "hash:", req.Hash, "configHash:", a.configHash)
	jsonMsg(c, "database applied", nil)
}

func (a *Agent) saveDatabaseSnapshot(dbPath string, snapshot []byte) error {
	reader := bytes.NewReader(snapshot)
	valid, err := database.IsSQLiteDB(reader)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("invalid sqlite database")
	}
	if _, err = reader.Seek(0, 0); err != nil {
		return err
	}

	tempPath := dbPath + ".tmp"
	backupPath := dbPath + ".backup"
	if err = os.WriteFile(tempPath, snapshot, 0644); err != nil {
		return err
	}
	defer os.Remove(tempPath)

	testDB, err := gorm.Open(sqlite.Open(tempPath), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := testDB.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if _, err = os.Stat(dbPath); err == nil {
		_ = os.Remove(backupPath)
		if err = os.Rename(dbPath, backupPath); err != nil {
			return err
		}
	}

	if err = os.Rename(tempPath, dbPath); err != nil {
		if _, statErr := os.Stat(backupPath); statErr == nil {
			_ = os.Rename(backupPath, dbPath)
		}
		return err
	}
	_ = os.Remove(backupPath)
	return nil
}

func (a *Agent) generateConfigFromDatabase(dbPath string) ([]byte, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	configService := &service.ConfigService{}
	rawConfig, err := configService.GetConfigFromDB(db, "")
	if err != nil {
		return nil, err
	}
	return *rawConfig, nil
}

func (a *Agent) handleStartCore(c *gin.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	err := a.startCoreInternal()
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "failed to start core: "+err.Error())
		return
	}

	jsonMsg(c, "core started", nil)
}

func (a *Agent) handleStopCore(c *gin.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	err := a.stopCoreInternal()
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "failed to stop core: "+err.Error())
		return
	}

	jsonMsg(c, "core stopped", nil)
}

func (a *Agent) handleRestartCore(c *gin.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	err := a.restartCoreInternal()
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "failed to restart core: "+err.Error())
		return
	}

	jsonMsg(c, "core restarted", nil)
}

func (a *Agent) handleStats(c *gin.Context) {
	stats := a.getStats()
	jsonObj(c, stats, nil)
}

func (a *Agent) handleLogs(c *gin.Context) {
	count := c.Query("c")
	level := c.Query("l")

	cInt, err := strconv.Atoi(count)
	if err != nil {
		cInt = 20
	}

	logs := logger.GetLogs(cInt, level)
	jsonObj(c, logs, nil)
}

func (a *Agent) getInfo() AgentInfo {
	info := AgentInfo{
		NodeName:       config.GetNodeName(),
		AgentVersion:   config.GetVersion(),
		LastConfigHash: a.configHash,
		LastDBHash:     a.dbHash,
		LastDBVersion:  a.dbVersion,
		LastApplyError: a.lastError,
	}

	a.mu.RLock()
	info.CoreRunning = a.core != nil && a.core.IsRunning()
	if info.CoreRunning && a.core.GetInstance() != nil {
		info.CoreUptime = a.core.GetInstance().Uptime()
	}
	a.mu.RUnlock()

	// System info
	hostname, _ := os.Hostname()
	info.Hostname = hostname
	info.Os = runtime.GOOS
	info.Arch = runtime.GOARCH

	// Get IPs
	info.Ips = a.getIPs()

	// Get Public IP
	info.PublicIp = util.GetPublicIP()

	return info
}

func (a *Agent) getIPs() []string {
	ips := make([]string, 0)

	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		// Check if interface is up and not loopback
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ipStr := addr.String()
				if strings.Contains(ipStr, ".") {
					// IPv4
					ipStr = strings.Split(ipStr, "/")[0]
					ips = append(ips, ipStr)
				} else if strings.HasPrefix(ipStr, "fe80::") {
					// Skip link-local IPv6
					continue
				} else {
					// IPv6
					ipStr = strings.Split(ipStr, "/")[0]
					ips = append(ips, ipStr)
				}
			}
		}
	}

	return ips
}

func (a *Agent) getStats() StatsResponse {
	response := StatsResponse{
		Stats:   make([]StatItem, 0),
		Onlines: Onlines{},
	}

	a.mu.RLock()
	if a.core == nil || !a.core.IsRunning() {
		a.mu.RUnlock()
		return response
	}
	box := a.core.GetInstance()
	a.mu.RUnlock()

	if box == nil {
		return response
	}

	st := box.StatsTracker()
	if st == nil {
		return response
	}

	stats := st.GetStats()
	if stats == nil || len(*stats) == 0 {
		return response
	}

	for _, stat := range *stats {
		item := StatItem{
			Resource:  stat.Resource,
			Tag:       stat.Tag,
			Direction: stat.Direction,
			Traffic:   stat.Traffic,
		}
		response.Stats = append(response.Stats, item)

		if stat.Direction {
			switch stat.Resource {
			case "inbound":
				response.Onlines.Inbound = append(response.Onlines.Inbound, stat.Tag)
			case "outbound":
				response.Onlines.Outbound = append(response.Onlines.Outbound, stat.Tag)
			case "user":
				response.Onlines.User = append(response.Onlines.User, stat.Tag)
			}
		}
	}

	return response
}

func (a *Agent) startCore() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.startCoreInternal()
}

func (a *Agent) startCoreInternal() error {
	if len(a.configCache) == 0 {
		return fmt.Errorf("no cached config available")
	}

	if a.core == nil {
		a.core = core.NewCore()
	}

	if a.core.IsRunning() {
		return nil
	}

	return a.core.Start(a.configCache)
}

func (a *Agent) stopCore() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stopCoreInternal()
}

func (a *Agent) stopCoreInternal() error {
	if a.core == nil {
		return nil
	}
	return a.core.Stop()
}

func (a *Agent) restartCore() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.restartCoreInternal()
}

func (a *Agent) restartCoreInternal() error {
	// Stop first
	err := a.stopCoreInternal()
	if err != nil {
		logger.Warning("error stopping core:", err)
	}

	// Start with cached config
	return a.startCoreInternal()
}

func (a *Agent) computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Helper functions for JSON responses
func jsonMsg(c *gin.Context, msg string, err error) {
	response := gin.H{
		"success": err == nil,
		"msg":     msg,
	}
	if err != nil {
		response["msg"] = err.Error()
	}
	c.JSON(http.StatusOK, response)
}

func jsonError(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{
		"success": false,
		"msg":     msg,
	})
}

func jsonObj(c *gin.Context, obj interface{}, err error) {
	response := gin.H{
		"success": err == nil,
		"obj":     obj,
	}
	if err != nil {
		response["msg"] = err.Error()
	}
	c.JSON(http.StatusOK, response)
}

package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

//go:embed version
var version string

//go:embed name
var name string

type LogLevel string

type RunMode string

const (
	Debug  LogLevel = "debug"
	Info   LogLevel = "info"
	Warn   LogLevel = "warn"
	Error  LogLevel = "error"
	Silent LogLevel = "silent"
)

const (
	ModePanel RunMode = "panel"
	ModeAgent RunMode = "agent"
)

func GetVersion() string {
	return strings.TrimSpace(version)
}

func GetName() string {
	return strings.TrimSpace(name)
}

func GetLogLevel() LogLevel {
	if IsDebug() {
		return Debug
	}
	logLevel := os.Getenv("SUI_LOG_LEVEL")
	if logLevel == "" {
		return Info
	}
	return LogLevel(logLevel)
}

func IsDebug() bool {
	return os.Getenv("SUI_DEBUG") == "true"
}

// GetMode returns the run mode: panel (default) or agent
func GetMode() RunMode {
	mode := os.Getenv("SUI_MODE")
	if mode == "" {
		return ModePanel
	}
	return RunMode(strings.ToLower(mode))
}

// IsPanel returns true if running in panel mode
func IsPanel() bool {
	return GetMode() == ModePanel
}

// IsAgent returns true if running in agent mode
func IsAgent() bool {
	return GetMode() == ModeAgent
}

// GetNodeName returns the agent node name (only used in agent mode)
func GetNodeName() string {
	return os.Getenv("SUI_NODE_NAME")
}

// GetNodeToken returns the agent node token (only used in agent mode)
func GetNodeToken() string {
	return os.Getenv("SUI_NODE_TOKEN")
}

// GetAgentListen returns the agent API listen address
func GetAgentListen() string {
	listen := os.Getenv("SUI_AGENT_LISTEN")
	if listen == "" {
		return "0.0.0.0"
	}
	return listen
}

// GetAgentPort returns the agent API port
func GetAgentPort() int {
	port := os.Getenv("SUI_AGENT_PORT")
	if port == "" {
		return 2097
	}
	var portInt int
	fmt.Sscanf(port, "%d", &portInt)
	if portInt <= 0 || portInt > 65535 {
		return 2097
	}
	return portInt
}

// GetEnableSub returns whether to enable sub service
// In panel mode: default is true
// In agent mode: default is false
func GetEnableSub() bool {
	enable := os.Getenv("SUI_ENABLE_SUB")
	if enable == "" {
		// Agent mode defaults to false, panel mode defaults to true
		return IsPanel()
	}
	return enable != "false" && enable != "0"
}

// GetEnableWeb returns whether to enable web service
// In panel mode: default is true
// In agent mode: default is false
func GetEnableWeb() bool {
	enable := os.Getenv("SUI_ENABLE_WEB")
	if enable == "" {
		// Agent mode defaults to false, panel mode defaults to true
		return IsPanel()
	}
	return enable != "false" && enable != "0"
}

// GetAgentCacheDir returns the agent cache directory
func GetAgentCacheDir() string {
	cacheDir := os.Getenv("SUI_AGENT_CACHE_DIR")
	if cacheDir == "" {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			if runtime.GOOS == "windows" {
				return "C:\\Program Files\\s-ui\\agent"
			}
			return "/usr/local/s-ui/agent"
		}
		cacheDir = filepath.Join(dir, "agent")
	}
	return cacheDir
}

func GetDBFolderPath() string {
	dbFolderPath := os.Getenv("SUI_DB_FOLDER")
	if dbFolderPath == "" {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			// Cross-platform fallback path
			if runtime.GOOS == "windows" {
				return "C:\\Program Files\\s-ui\\db"
			}
			return "/usr/local/s-ui/db"
		}
		dbFolderPath = filepath.Join(dir, "db")
	}
	return dbFolderPath
}

func GetDBPath() string {
	return fmt.Sprintf("%s/%s.db", GetDBFolderPath(), GetName())
}

// GetAgentNodeID reads the node_id file in agent cache dir and returns it. Returns 0 if not exist or error.
func GetAgentNodeID() uint {
	nodeIdPath := filepath.Join(GetAgentCacheDir(), "node_id")
	data, err := os.ReadFile(nodeIdPath)
	if err != nil {
		return 0
	}
	idStr := strings.TrimSpace(string(data))
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// SaveAgentNodeID saves the given agent node ID to the node_id file in agent cache dir.
func SaveAgentNodeID(id uint) error {
	cacheDir := GetAgentCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}
	nodeIdPath := filepath.Join(cacheDir, "node_id")
	data := []byte(strconv.FormatUint(uint64(id), 10))
	return os.WriteFile(nodeIdPath, data, 0644)
}

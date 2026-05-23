package migration

import (
	"net/url"
	"strconv"

	"gorm.io/gorm"
)

func to1_4(db *gorm.DB) error {
	// Step 1: Add new columns
	columns := []struct {
		name     string
		dataType string
		def      string
	}{
		{"api_host", "TEXT", ""},
		{"api_port", "INTEGER", "2097"},
		{"api_scheme", "TEXT", "'http'"},
		{"public_host_mode", "TEXT", "'agent'"},
	}

	for _, col := range columns {
		if !db.Migrator().HasColumn("nodes", col.name) {
			sql := "ALTER TABLE nodes ADD COLUMN " + col.name + " " + col.dataType
			if col.def != "" {
				sql += " DEFAULT " + col.def
			}
			if err := db.Exec(sql).Error; err != nil {
				return err
			}
		}
	}

	// Step 2: Parse existing api_base_url into new fields
	type nodeRow struct {
		Id         uint
		Type       string
		ApiBaseUrl string
	}

	var nodes []nodeRow
	if err := db.Raw("SELECT id, type, api_base_url FROM nodes WHERE api_base_url IS NOT NULL AND api_base_url != ''").Find(&nodes).Error; err != nil {
		return err
	}

	for _, node := range nodes {
		if node.ApiBaseUrl == "" {
			continue
		}

		// Parse URL: e.g., http://1.2.3.4:2097/agent/api
		u, err := url.Parse(node.ApiBaseUrl)
		if err != nil {
			continue
		}

		scheme := u.Scheme
		if scheme == "" {
			scheme = "http"
		}

		host := u.Hostname()
		port := 2097
		if u.Port() != "" {
			if p, err := strconv.Atoi(u.Port()); err == nil {
				port = p
			}
		}

		// Set public_host_mode based on node type
		publicHostMode := "agent"
		if node.Type == "local" {
			publicHostMode = "local"
		}

		// Update the node with new fields
		if err := db.Exec(
			"UPDATE nodes SET api_host = ?, api_port = ?, api_scheme = ?, public_host_mode = ? WHERE id = ?",
			host, port, scheme, publicHostMode, node.Id,
		).Error; err != nil {
			return err
		}
	}

	// Step 3: Set default public_host_mode for nodes without api_base_url
	if err := db.Exec("UPDATE nodes SET public_host_mode = 'local' WHERE type = 'local' AND (public_host_mode IS NULL OR public_host_mode = '')").Error; err != nil {
		return err
	}
	if err := db.Exec("UPDATE nodes SET public_host_mode = 'agent' WHERE type = 'remote' AND (public_host_mode IS NULL OR public_host_mode = '')").Error; err != nil {
		return err
	}

	return nil
}

// removeEmptyPortMap removes empty public_port_map values for nodes
func removeEmptyPortMap(db *gorm.DB) error {
	return db.Exec("UPDATE nodes SET public_port_map = NULL WHERE public_port_map = ''").Error
}

// parseApiUrlForLocal handles nodes that have no api_base_url (local nodes)
func setLocalNodeDefaults(db *gorm.DB) error {
	// Local nodes without api_host should use localhost
	return db.Exec("UPDATE nodes SET api_host = '127.0.0.1', api_port = 2097, api_scheme = 'http' WHERE type = 'local' AND (api_host IS NULL OR api_host = '')").Error
}

func migrateNodePublicPortMap(db *gorm.DB) error {
	// Convert empty string public_port_map to NULL
	return db.Exec("UPDATE nodes SET public_port_map = NULL WHERE public_port_map = '' OR public_port_map = '{}'").Error
}

func cleanUpNodeFields(db *gorm.DB) error {
	// Ensure api_port has default value
	return db.Exec("UPDATE nodes SET api_port = 2097 WHERE api_port IS NULL OR api_port = 0").Error
}

func to1_4_full(db *gorm.DB) error {
	if err := to1_4(db); err != nil {
		return err
	}
	if err := setLocalNodeDefaults(db); err != nil {
		return err
	}
	if err := migrateNodePublicPortMap(db); err != nil {
		return err
	}
	if err := cleanUpNodeFields(db); err != nil {
		return err
	}
	if err := removeEmptyPortMap(db); err != nil {
		return err
	}
	return nil
}

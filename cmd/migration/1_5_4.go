package migration

import (
	"encoding/json"

	"gorm.io/gorm"
)

// to1_5_4 adds the nodes column to the clients table for node selection feature.
// Existing clients will have nodes initialized to empty array (use all enabled nodes).
func to1_5_4(tx *gorm.DB) error {
	// Check if nodes column already exists
	var count int64
	tx.Raw("SELECT COUNT(*) FROM pragma_table_info('clients') WHERE name = 'nodes'").Scan(&count)
	if count > 0 {
		return nil // Column already exists
	}

	// Add nodes column with default empty JSON array
	if err := tx.Exec("ALTER TABLE clients ADD COLUMN nodes TEXT DEFAULT '[]'").Error; err != nil {
		return err
	}

	// Initialize existing clients with empty nodes array
	emptyNodes, _ := json.Marshal([]uint{})
	if err := tx.Exec("UPDATE clients SET nodes = ? WHERE nodes IS NULL", string(emptyNodes)).Error; err != nil {
		return err
	}

	return nil
}

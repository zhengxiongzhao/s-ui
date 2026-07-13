package migration

import (
	"gorm.io/gorm"
)

// to1_5_5 adds sort and auto_sync columns to the nodes table
func to1_5_5(tx *gorm.DB) error {
	// Check if sort column already exists
	var sortCount int64
	tx.Raw("SELECT COUNT(*) FROM pragma_table_info('nodes') WHERE name = 'sort'").Scan(&sortCount)
	if sortCount == 0 {
		// Add sort column with default 0
		if err := tx.Exec("ALTER TABLE nodes ADD COLUMN sort INTEGER DEFAULT 0 NOT NULL").Error; err != nil {
			return err
		}
	}

	// Check if auto_sync column already exists
	var autoSyncCount int64
	tx.Raw("SELECT COUNT(*) FROM pragma_table_info('nodes') WHERE name = 'auto_sync'").Scan(&autoSyncCount)
	if autoSyncCount == 0 {
		// Add auto_sync column with default true (1)
		if err := tx.Exec("ALTER TABLE nodes ADD COLUMN auto_sync BOOLEAN DEFAULT 1 NOT NULL").Error; err != nil {
			return err
		}
	}

	return nil
}

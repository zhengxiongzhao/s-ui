package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONRawMessage is a custom type that wraps json.RawMessage and implements
// sql.Scanner and driver.Valuer interfaces to properly handle SQLite's
// string-based storage of JSON data.
type JSONRawMessage json.RawMessage

// Scan implements sql.Scanner interface for reading from database.
// SQLite returns TEXT columns as string, but json.RawMessage is []byte.
// This method handles the conversion from string to []byte.
func (j *JSONRawMessage) Scan(value interface{}) error {
	if value == nil {
		*j = JSONRawMessage("null")
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*j = append((*j)[0:0], v...)
	case string:
		*j = JSONRawMessage(v)
	default:
		return fmt.Errorf("JSONRawMessage.Scan: unsupported type %T", value)
	}
	return nil
}

// Value implements driver.Valuer interface for writing to database.
func (j JSONRawMessage) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

// MarshalJSON implements json.Marshaler interface.
func (j JSONRawMessage) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (j *JSONRawMessage) UnmarshalJSON(data []byte) error {
	if j == nil {
		return fmt.Errorf("JSONRawMessage.UnmarshalJSON: nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

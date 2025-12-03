// Package models - JSONB type for PostgreSQL
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONB is a custom type for PostgreSQL JSONB columns
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	if len(bytes) == 0 {
		*j = make(JSONB)
		return nil
	}

	result := make(JSONB)
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}
	*j = result
	return nil
}

// StringArray is a custom type for PostgreSQL TEXT[] columns
type StringArray []string

// Value implements the driver.Valuer interface
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	if len(bytes) == 0 {
		*s = make(StringArray, 0)
		return nil
	}

	// Try JSON array first
	var result []string
	if err := json.Unmarshal(bytes, &result); err != nil {
		// Try PostgreSQL array format: {val1,val2,val3}
		str := string(bytes)
		if len(str) > 2 && str[0] == '{' && str[len(str)-1] == '}' {
			// Remove braces and split
			str = str[1 : len(str)-1]
			if str == "" {
				*s = make(StringArray, 0)
				return nil
			}
			// Simple split (doesn't handle quoted strings with commas)
			result = splitPostgresArray(str)
		} else {
			return err
		}
	}
	*s = result
	return nil
}

func splitPostgresArray(s string) []string {
	var result []string
	var current string
	inQuotes := false

	for _, c := range s {
		switch c {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				result = append(result, current)
				current = ""
			} else {
				current += string(c)
			}
		default:
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

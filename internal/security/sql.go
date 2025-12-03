// Package security provides security-related utilities for Genesis
package security

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidIdentifierRegex matches valid PostgreSQL identifiers
// Only allows lowercase letters, digits, and underscores, starting with a letter or underscore
var ValidIdentifierRegex = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// ValidateIdentifier checks if a string is a valid SQL identifier
func ValidateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if len(name) > 63 {
		return fmt.Errorf("identifier too long (max 63 characters)")
	}
	if !ValidIdentifierRegex.MatchString(name) {
		return fmt.Errorf("invalid identifier: must contain only lowercase letters, numbers, and underscores, starting with a letter or underscore")
	}
	// Check for reserved words
	if isReservedWord(name) {
		return fmt.Errorf("'%s' is a reserved SQL keyword", name)
	}
	return nil
}

// QuoteIdentifier safely quotes a PostgreSQL identifier
// This should only be used AFTER validation
func QuoteIdentifier(name string) string {
	// Double any internal quotes for safety
	escaped := strings.ReplaceAll(name, `"`, `""`)
	return `"` + escaped + `"`
}

// SafeIdentifier validates and quotes an identifier for use in SQL
func SafeIdentifier(name string) (string, error) {
	if err := ValidateIdentifier(name); err != nil {
		return "", err
	}
	return QuoteIdentifier(name), nil
}

// EscapeLikePattern escapes special characters in LIKE patterns
func EscapeLikePattern(pattern string) string {
	// Escape the special characters used in SQL LIKE: %, _, and \
	pattern = strings.ReplaceAll(pattern, `\`, `\\`)
	pattern = strings.ReplaceAll(pattern, `%`, `\%`)
	pattern = strings.ReplaceAll(pattern, `_`, `\_`)
	return pattern
}

// SearchCondition builds a safe search condition
// Returns the condition string and the parameters
func SearchCondition(columnName, searchTerm string) (string, []interface{}) {
	escaped := EscapeLikePattern(searchTerm)
	// Use parameterized query with ESCAPE clause
	condition := fmt.Sprintf(`%s ILIKE $1 ESCAPE '\'`, QuoteIdentifier(columnName))
	param := "%" + escaped + "%"
	return condition, []interface{}{param}
}

// BuildMultiSearchCondition builds conditions for searching multiple columns
// Returns the condition string and the parameters
func BuildMultiSearchCondition(columns []string, searchTerm string, paramOffset int) (string, []interface{}) {
	if len(columns) == 0 || searchTerm == "" {
		return "", nil
	}

	escaped := EscapeLikePattern(searchTerm)
	param := "%" + escaped + "%"

	conditions := make([]string, 0, len(columns))
	for _, col := range columns {
		if err := ValidateIdentifier(col); err == nil {
			conditions = append(conditions, fmt.Sprintf(`%s ILIKE $%d ESCAPE '\'`, QuoteIdentifier(col), paramOffset))
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "(" + strings.Join(conditions, " OR ") + ")", []interface{}{param}
}

// isReservedWord checks if a word is a PostgreSQL reserved word
func isReservedWord(word string) bool {
	reserved := map[string]bool{
		"all": true, "analyse": true, "analyze": true, "and": true, "any": true,
		"array": true, "as": true, "asc": true, "asymmetric": true, "both": true,
		"case": true, "cast": true, "check": true, "collate": true, "column": true,
		"constraint": true, "create": true, "current_catalog": true, "current_date": true,
		"current_role": true, "current_time": true, "current_timestamp": true,
		"current_user": true, "default": true, "deferrable": true, "desc": true,
		"distinct": true, "do": true, "else": true, "end": true, "except": true,
		"false": true, "fetch": true, "for": true, "foreign": true, "from": true,
		"grant": true, "group": true, "having": true, "in": true, "initially": true,
		"intersect": true, "into": true, "lateral": true, "leading": true, "limit": true,
		"localtime": true, "localtimestamp": true, "not": true, "null": true, "offset": true,
		"on": true, "only": true, "or": true, "order": true, "placing": true,
		"primary": true, "references": true, "returning": true, "select": true,
		"session_user": true, "some": true, "symmetric": true, "table": true,
		"then": true, "to": true, "trailing": true, "true": true, "union": true,
		"unique": true, "user": true, "using": true, "variadic": true, "when": true,
		"where": true, "window": true, "with": true,
	}
	return reserved[strings.ToLower(word)]
}

// AllowedFilterOperators defines the allowed comparison operators for filters
var AllowedFilterOperators = map[string]string{
	"eq":  "=",
	"ne":  "!=",
	"gt":  ">",
	"gte": ">=",
	"lt":  "<",
	"lte": "<=",
	"in":  "IN",
	"nin": "NOT IN",
	"like": "ILIKE",
	"null": "IS NULL",
	"notnull": "IS NOT NULL",
}

// BuildFilterCondition builds a safe filter condition
// Returns condition and parameters
func BuildFilterCondition(column string, operator string, value interface{}, paramNum int) (string, interface{}, error) {
	if err := ValidateIdentifier(column); err != nil {
		return "", nil, err
	}

	quotedCol := QuoteIdentifier(column)

	op, exists := AllowedFilterOperators[operator]
	if !exists {
		op = "=" // default to equality
	}

	switch op {
	case "IS NULL":
		return fmt.Sprintf("%s IS NULL", quotedCol), nil, nil
	case "IS NOT NULL":
		return fmt.Sprintf("%s IS NOT NULL", quotedCol), nil, nil
	case "ILIKE":
		escaped := EscapeLikePattern(fmt.Sprintf("%v", value))
		return fmt.Sprintf(`%s ILIKE $%d ESCAPE '\'`, quotedCol, paramNum), "%" + escaped + "%", nil
	case "IN", "NOT IN":
		return fmt.Sprintf("%s %s ($%d)", quotedCol, op, paramNum), value, nil
	default:
		return fmt.Sprintf("%s %s $%d", quotedCol, op, paramNum), value, nil
	}
}

// Package conv provides internal value conversion helpers.
package conv

import "fmt"

// SprintValue formats a value as a string. If the value is already a string,
// it is returned without allocation.
func SprintValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// Package models defines data structures for Polymarket API responses.
package models

import "encoding/json"

// FlexString handles JSON fields that can be either string or number.
// The Polymarket API is inconsistent in whether it returns numeric fields
// as strings or numbers, so this type handles both cases gracefully.
type FlexString string

// UnmarshalJSON implements custom JSON unmarshaling for FlexString.
// It first tries to unmarshal as a string, then as a number.
// If both fail, it defaults to an empty string.
func (f *FlexString) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexString(s)
		return nil
	}
	// Try number (will be formatted as string)
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexString(n.String())
		return nil
	}
	// If null or empty, use empty string
	*f = ""
	return nil
}

// String returns the underlying string value.
func (f FlexString) String() string {
	return string(f)
}

// IsEmpty returns true if the value is empty.
func (f FlexString) IsEmpty() bool {
	return f == ""
}

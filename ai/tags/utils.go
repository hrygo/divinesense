package tags

import (
	"encoding/json"
)

// encodeJSON serializes a value to JSON bytes.
func encodeJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// decodeJSON deserializes JSON bytes into a value.
func decodeJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

package tags

import "encoding/json"

// encodeJSON encodes value to JSON bytes.
func encodeJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

// decodeJSON decodes JSON bytes to value.
func decodeJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

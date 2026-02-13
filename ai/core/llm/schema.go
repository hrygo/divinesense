package llm

import "encoding/json"

// JSONSchema implements json.Marshaler for OpenAI's JSON Schema format.
// The alias type prevents infinite recursion during marshaling.
// JSONSchema 实现 OpenAI JSON Schema 格式的 json.Marshaler。
// 别名类型防止序列化时的无限递归。
type JSONSchema struct {
	Properties           map[string]*JSONSchema `json:"properties,omitempty"`
	Type                 string                 `json:"type"`
	Description          string                 `json:"description,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Enum                 []string               `json:"enum,omitempty"`
	AdditionalProperties bool                   `json:"additionalProperties"`
}

// MarshalJSON implements json.Marshaler for JSONSchema.
// It uses type alias to prevent infinite recursion.
func (s *JSONSchema) MarshalJSON() ([]byte, error) {
	type alias JSONSchema
	return json.Marshal((*alias)(s))
}

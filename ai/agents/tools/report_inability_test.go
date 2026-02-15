package tools

import (
	"context"
	"testing"
)

func TestReportInabilityTool_Run(t *testing.T) {
	tool := NewReportInabilityTool()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "valid input with capability and reason",
			input:   `{"capability": "create_meeting", "reason": "I can only search memos, not create meetings"}`,
			want:    "INABILITY_REPORTED: create_meeting - I can only search memos, not create meetings",
			wantErr: false,
		},
		{
			name:    "valid input with suggested agent",
			input:   `{"capability": "schedule_event", "reason": "outside my domain", "suggested_agent": "schedule"}`,
			want:    "INABILITY_REPORTED: schedule_event - outside my domain (suggested_agent: schedule)",
			wantErr: false,
		},
		{
			name:    "missing capability",
			input:   `{"reason": "no reason"}`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing reason",
			input:   `{"capability": "test"}`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty capability",
			input:   `{"capability": "", "reason": "test"}`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `not json`,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := tool.Run(ctx, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReportInabilityTool_Name(t *testing.T) {
	tool := NewReportInabilityTool()
	if tool.Name() != "report_inability" {
		t.Errorf("got %q, want %q", tool.Name(), "report_inability")
	}
}

func TestReportInabilityTool_Description(t *testing.T) {
	tool := NewReportInabilityTool()
	desc := tool.Description()
	if desc == "" {
		t.Error("description should not be empty")
	}
	if len(desc) < 50 {
		t.Errorf("description too short: %q", desc)
	}
}

func TestReportInabilityTool_InputType(t *testing.T) {
	tool := NewReportInabilityTool()
	inputType := tool.InputType()

	if inputType == nil {
		t.Error("inputType should not be nil")
	}

	// Check that required fields are defined
	obj, ok := inputType["type"].(string)
	if !ok || obj != "object" {
		t.Errorf("expected type object, got %v", inputType["type"])
	}

	props, ok := inputType["properties"].(map[string]interface{})
	if !ok {
		t.Error("expected properties to be map")
		return
	}

	// Check required fields
	if _, ok := props["capability"]; !ok {
		t.Error("expected capability field")
	}
	if _, ok := props["reason"]; !ok {
		t.Error("expected reason field")
	}
	if _, ok := props["suggested_agent"]; !ok {
		t.Error("expected suggested_agent field")
	}
}

func TestReportInabilityInput_Error(t *testing.T) {
	input := ReportInabilityInput{
		Capability: "test_capability",
		Reason:     "test_reason",
	}

	expected := "cannot handle capability test_capability: test_reason"
	if input.Error() != expected {
		t.Errorf("got %q, want %q", input.Error(), expected)
	}
}

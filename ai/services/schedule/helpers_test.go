package schedule

import (
	"testing"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalReminders(t *testing.T) {
	tests := []struct {
		name      string
		reminders []*v1pb.Reminder
		wantEmpty bool
		wantErr   bool
	}{
		{
			name:      "empty reminders",
			reminders: []*v1pb.Reminder{},
			wantEmpty: true,
			wantErr:   false,
		},
		{
			name:      "nil reminders",
			reminders: nil,
			wantEmpty: true,
			wantErr:   false,
		},
		{
			name: "single reminder",
			reminders: []*v1pb.Reminder{
				{Type: "email", Value: 30, Unit: "minute"},
			},
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name: "multiple reminders",
			reminders: []*v1pb.Reminder{
				{Type: "email", Value: 30, Unit: "minute"},
				{Type: "notification", Value: 1, Unit: "hour"},
			},
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name: "reminder with zero value",
			reminders: []*v1pb.Reminder{
				{Type: "email", Value: 0, Unit: "minute"},
			},
			wantEmpty: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalReminders(tt.reminders)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.wantEmpty {
				assert.Empty(t, got)
			} else {
				assert.NotEmpty(t, got)
				// Verify it can be unmarshaled back
				unmarshaled, err := UnmarshalReminders(got)
				require.NoError(t, err)
				assert.Equal(t, len(tt.reminders), len(unmarshaled))
			}
		})
	}
}

func TestUnmarshalReminders(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantLen int
		wantNil bool
		wantErr bool
	}{
		{
			name:    "empty string",
			data:    "",
			wantNil: true,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "whitespace string",
			data:    "   ",
			wantNil: true,
			wantLen: 0,
			wantErr: true, // Invalid JSON
		},
		{
			name:    "valid empty array",
			data:    "[]",
			wantNil: false,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "valid single reminder",
			data:    `[{"type":"email","value":30,"unit":"minute"}]`,
			wantNil: false,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "valid multiple reminders",
			data:    `[{"type":"email","value":30,"unit":"minute"},{"type":"notification","value":1,"unit":"hour"}]`,
			wantNil: false,
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			data:    `{invalid}`,
			wantNil: true,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "malformed JSON array",
			data:    `[{"type":"email"}]`,
			wantNil: false,
			wantLen: 1,
			wantErr: false, // Missing fields are allowed in protobuf
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalReminders(tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.wantLen, len(got))
			}
		})
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	original := []*v1pb.Reminder{
		{Type: "email", Value: 30, Unit: "minute"},
		{Type: "notification", Value: 1, Unit: "hour"},
		{Type: "webhook", Value: 24, Unit: "day"},
	}

	// Marshal
	marshaled, err := MarshalReminders(original)
	require.NoError(t, err)
	require.NotEmpty(t, marshaled)

	// Unmarshal
	unmarshaled, err := UnmarshalReminders(marshaled)
	require.NoError(t, err)
	require.Len(t, unmarshaled, len(original))

	// Verify all fields
	for i := range original {
		assert.Equal(t, original[i].Type, unmarshaled[i].Type)
		assert.Equal(t, original[i].Value, unmarshaled[i].Value)
		assert.Equal(t, original[i].Unit, unmarshaled[i].Unit)
	}
}

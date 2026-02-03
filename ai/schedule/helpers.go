package schedule

import (
	"encoding/json"
	"fmt"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
)

// MarshalReminders converts protobuf reminders to JSON string.
func MarshalReminders(reminders []*v1pb.Reminder) (string, error) {
	if len(reminders) == 0 {
		return "", nil
	}
	data, err := json.Marshal(reminders)
	if err != nil {
		return "", fmt.Errorf("failed to marshal reminders: %w", err)
	}
	return string(data), nil
}

// UnmarshalReminders converts JSON to protobuf reminders.
func UnmarshalReminders(data string) ([]*v1pb.Reminder, error) {
	if data == "" {
		return nil, nil
	}
	var reminders []*v1pb.Reminder
	if err := json.Unmarshal([]byte(data), &reminders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reminders: %w", err)
	}
	return reminders, nil
}

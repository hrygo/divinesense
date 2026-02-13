package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/hrygo/divinesense/store"
	"github.com/lib/pq"
)

func (d *DB) CreateSchedule(ctx context.Context, create *store.Schedule) (*store.Schedule, error) {
	fields := []string{
		"uid", "creator_id", "title", "description", "location",
		"start_ts", "end_ts", "all_day", "timezone",
		"recurrence_rule", "recurrence_end_ts", "reminders", "payload",
	}
	placeholderValues := []any{
		create.UID, create.CreatorID, create.Title, create.Description, create.Location,
		create.StartTs, create.EndTs, create.AllDay, create.Timezone,
		create.RecurrenceRule, create.RecurrenceEndTs, create.Reminders, create.Payload,
	}

	// Add optional timestamps
	if create.CreatedTs != 0 {
		fields = append(fields, "created_ts")
		placeholderValues = append(placeholderValues, create.CreatedTs)
	}
	if create.UpdatedTs != 0 {
		fields = append(fields, "updated_ts")
		placeholderValues = append(placeholderValues, create.UpdatedTs)
	}

	stmt := `INSERT INTO schedule (` + strings.Join(fields, ", ") + `)
		VALUES (` + placeholders(len(placeholderValues)) + `)
		RETURNING id, created_ts, updated_ts, row_status`

	if err := d.db.QueryRowContext(ctx, stmt, placeholderValues...).Scan(
		&create.ID,
		&create.CreatedTs,
		&create.UpdatedTs,
		&create.RowStatus,
	); err != nil {
		// Check for EXCLUDE constraint violation (conflict detection)
		if isConflictConstraintError(err) {
			return nil, &ConflictConstraintError{
				Err:     err,
				Message: "schedule conflicts with an existing schedule",
			}
		}
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return create, nil
}

func (d *DB) ListSchedules(ctx context.Context, find *store.FindSchedule) ([]*store.Schedule, error) {
	where, args := []string{"1 = 1"}, []any{}

	if v := find.ID; v != nil {
		where, args = append(where, "schedule.id = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, "schedule.uid = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, "schedule.creator_id = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, "schedule.row_status = "+placeholder(len(args)+1)), append(args, *v)
	}

	// P1: 根据查询模式应用不同的时间范围过滤
	// QueryMode: 0 = AUTO, 1 = STANDARD (标准模式), 2 = STRICT (严格模式)
	queryMode := int32(0) // 默认标准模式
	if v := find.QueryMode; v != nil {
		queryMode = *v
	}

	if v := find.StartTs; v != nil {
		if queryMode == 2 {
			// 严格模式：日程必须完全在查询范围内
			// 日程的开始时间 >= 查询范围的开始时间
			where, args = append(where, "schedule.start_ts >= "+placeholder(len(args)+1)), append(args, *v)
		} else {
			// 标准模式（默认）：返回有任何部分在范围内的日程
			// 日程的结束时间 >= 查询范围的开始时间
			where, args = append(where, "(schedule.end_ts >= "+placeholder(len(args)+1)+" OR schedule.end_ts IS NULL)"), append(args, *v)
		}
	}
	if v := find.EndTs; v != nil {
		if queryMode == 2 {
			// 严格模式：日程必须完全在查询范围内
			// 日程的结束时间 <= 查询范围的结束时间
			where, args = append(where, "(schedule.end_ts <= "+placeholder(len(args)+1)+" OR schedule.end_ts IS NULL)"), append(args, *v)
		} else {
			// 标准模式（默认）：返回有任何部分在范围内的日程
			// 日程的开始时间 <= 查询范围的结束时间
			where, args = append(where, "schedule.start_ts <= "+placeholder(len(args)+1)), append(args, *v)
		}
	}

	// Ordering (always by start_ts ascending)
	orderBy := "ORDER BY schedule.start_ts ASC"

	query := `
		SELECT
			id, uid, creator_id, created_ts, updated_ts, row_status,
			title, description, location,
			start_ts, end_ts, all_day, timezone,
			recurrence_rule, recurrence_end_ts, reminders, payload
		FROM schedule
		WHERE ` + strings.Join(where, " AND ") + ` ` + orderBy

	if find.Limit != nil {
		limit := *find.Limit
		if limit < 0 {
			limit = 0
		}
		if limit > 10000 {
			limit = 10000
		}
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
		if find.Offset != nil {
			offset := *find.Offset
			if offset < 0 {
				offset = 0
			}
			query = fmt.Sprintf("%s OFFSET %d", query, offset)
		}
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	list := make([]*store.Schedule, 0)
	for rows.Next() {
		var schedule store.Schedule
		var description, location, reminders, payload string
		var recurrenceRule sql.NullString
		var endTs, recurrenceEndTs sql.NullInt64

		if err := rows.Scan(
			&schedule.ID,
			&schedule.UID,
			&schedule.CreatorID,
			&schedule.CreatedTs,
			&schedule.UpdatedTs,
			&schedule.RowStatus,
			&schedule.Title,
			&description,
			&location,
			&schedule.StartTs,
			&endTs,
			&schedule.AllDay,
			&schedule.Timezone,
			&recurrenceRule,
			&recurrenceEndTs,
			&reminders,
			&payload,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		schedule.Description = description
		schedule.Location = location
		if endTs.Valid {
			// Create a new int64 value to avoid pointer to local variable
			endTsVal := endTs.Int64
			schedule.EndTs = &endTsVal
		}
		if recurrenceRule.Valid {
			// Create a new string value to avoid pointer to local variable
			ruleVal := recurrenceRule.String
			schedule.RecurrenceRule = &ruleVal
		}
		if recurrenceEndTs.Valid {
			// Create a new int64 value to avoid pointer to local variable
			recEndTsVal := recurrenceEndTs.Int64
			schedule.RecurrenceEndTs = &recEndTsVal
		}
		// Always set reminders and payload (use default if empty)
		if reminders == "" || reminders == "[]" {
			reminders = "[]"
		}
		schedule.Reminders = &reminders
		if payload == "" || payload == "{}" {
			payload = "{}"
		}
		schedule.Payload = &payload

		list = append(list, &schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate schedules: %w", err)
	}

	return list, nil
}

func (d *DB) UpdateSchedule(ctx context.Context, update *store.UpdateSchedule) error {
	set, args := []string{}, []any{}

	if v := update.UID; v != nil {
		set, args = append(set, "uid = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.RowStatus; v != nil {
		set, args = append(set, "row_status = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.Title; v != nil {
		set, args = append(set, "title = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.Description; v != nil {
		set, args = append(set, "description = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.Location; v != nil {
		set, args = append(set, "location = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.StartTs; v != nil {
		set, args = append(set, "start_ts = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.EndTs; v != nil {
		set, args = append(set, "end_ts = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.AllDay; v != nil {
		set, args = append(set, "all_day = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.Timezone; v != nil {
		set, args = append(set, "timezone = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.RecurrenceRule; v != nil {
		set, args = append(set, "recurrence_rule = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.RecurrenceEndTs; v != nil {
		set, args = append(set, "recurrence_end_ts = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.Reminders; v != nil {
		set, args = append(set, "reminders = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.Payload; v != nil {
		set, args = append(set, "payload = "+placeholder(len(args)+1)), append(args, *v)
	}

	// If no fields to update, return early
	if len(set) == 0 {
		return nil
	}

	args = append(args, update.ID)

	stmt := `UPDATE schedule SET ` + strings.Join(set, ", ") + ` WHERE id = ` + placeholder(len(args)) + ` RETURNING id, updated_ts`
	var id int32
	var updatedTs int64
	err := d.db.QueryRowContext(ctx, stmt, args...).Scan(&id, &updatedTs)
	if err == sql.ErrNoRows {
		return fmt.Errorf("schedule not found")
	}
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	return nil
}

func (d *DB) DeleteSchedule(ctx context.Context, delete *store.DeleteSchedule) error {
	stmt := `DELETE FROM schedule WHERE id = ` + placeholder(1)
	result, err := d.db.ExecContext(ctx, stmt, delete.ID)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("schedule not found")
	}

	return nil
}

// ConflictConstraintError represents a schedule conflict constraint violation.
// This error is returned when the database EXCLUDE constraint detects an overlap.
type ConflictConstraintError struct {
	Err     error
	Message string
}

func (e *ConflictConstraintError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "schedule conflicts with an existing schedule"
}

// IsConflict implements the agent.ConflictError interface for DIP.
func (e *ConflictConstraintError) IsConflict() bool {
	return true
}

func (e *ConflictConstraintError) Unwrap() error {
	return e.Err
}

// IsConflictConstraintError checks if an error is from the EXCLUDE constraint violation.
// PostgreSQL returns error code 23514 for check_violation or exclusion_violation.
// The EXCLUDE constraint returns SQLState '23P01' (exclusion_violation).
func isConflictConstraintError(err error) bool {
	if err == nil {
		return false
	}

	// Check for PostgreSQL exclusion_violation error code
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// 23P01 is the SQLSTATE for exclusion_violation
		if pqErr.Code == "23P01" {
			return true
		}
		// Also check if the constraint name is in the error message
		if strings.Contains(pqErr.Message, "no_overlapping_schedules") {
			return true
		}
	}

	return false
}

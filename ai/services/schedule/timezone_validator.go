// Package schedule provides timezone-aware validation for schedule creation.
// This module handles Daylight Saving Time (DST) edge cases including:
//   - Invalid local times (spring forward gap)
//   - Ambiguous local times (fall back duplicate)
package schedule

import (
	"fmt"
	"time"

	"log/slog"
)

// ValidationResult contains the result of validating a local time.
type ValidationResult struct {
	ValidTime time.Time // The validated time (adjusted if necessary)
	Warnings  []string  // Any warnings about time adjustments
	IsValid   bool      // Whether the validation passed
}

// TimezoneValidator handles DST edge cases for a specific timezone.
type TimezoneValidator struct {
	location *time.Location
	timezone string
}

// NewTimezoneValidator creates a new validator for the given timezone.
func NewTimezoneValidator(tz string) *TimezoneValidator {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		slog.Warn("invalid timezone, using UTC", "timezone", tz, "error", err)
		loc = time.UTC
		tz = "UTC"
	}
	return &TimezoneValidator{
		location: loc,
		timezone: tz,
	}
}

// ValidateLocalTime validates and normalizes a local time, handling DST edge cases.
//
// Parameters:
//   - year, month, day, hour, min: The local time components to validate
//
// Returns:
//   - ValidationResult: Contains the validated time and any warnings
//
// DST Edge Cases Handled:
//
//  1. Spring Forward (Invalid Time): When clocks "spring forward", times in the gap don't exist.
//     Example: In America/New_York on 2024-03-10, time jumps from 1:59:59 AM to 3:00:00 AM.
//     Times like 2:30 AM don't exist - they're adjusted forward to 3:00 AM.
//
//  2. Fall Back (Ambiguous Time): When clocks "fall back", the same hour occurs twice.
//     Example: In America/New_York on 2024-11-03, 1:30 AM occurs twice (before and after the switch).
//     The first occurrence (Eastern Daylight Time) is used by default.
func (v *TimezoneValidator) ValidateLocalTime(year int, month time.Month, day, hour, min int) *ValidationResult {
	var warnings []string

	// Create the time in the configured location
	// time.Date handles invalid times by advancing the hour
	t := time.Date(year, month, day, hour, min, 0, 0, v.location)

	// Check for DST transition - invalid time (spring forward)
	// If the hour doesn't match what we requested, time.Date adjusted it
	if t.Hour() != hour {
		warning := fmt.Sprintf("specified time %04d-%02d-%02d %02d:%02d does not exist due to DST spring forward, adjusted to %02d:%02d in %s",
			year, month, day, hour, min, t.Hour(), t.Minute(), v.timezone)
		warnings = append(warnings, warning)
		slog.Debug("timezone_validator: invalid time adjusted",
			"requested_hour", hour,
			"adjusted_hour", t.Hour(),
			"timezone", v.timezone,
			"date", fmt.Sprintf("%04d-%02d-%02d", year, month, day))
	}

	// For ambiguous times (fall back), time.Date uses the first occurrence
	// We can detect this by checking if the zone name changes within the hour
	// and the time is in the "first" part of that hour
	if v.isAmbiguousTime(t) {
		warning := fmt.Sprintf("specified time %04d-%02d-%02d %02d:%02d is ambiguous due to DST fall back, using first occurrence in %s",
			year, month, day, hour, min, v.timezone)
		warnings = append(warnings, warning)
		slog.Debug("timezone_validator: ambiguous time detected",
			"hour", hour,
			"min", min,
			"timezone", v.timezone,
			"date", fmt.Sprintf("%04d-%02d-%02d", year, month, day))
	}

	return &ValidationResult{
		ValidTime: t,
		Warnings:  warnings,
		IsValid:   true,
	}
}

// ValidateTimestamp validates a Unix timestamp, converting it to the local timezone
// and checking for any issues.
func (v *TimezoneValidator) ValidateTimestamp(ts int64) *ValidationResult {
	t := time.Unix(ts, 0).In(v.location)

	// Get the local components and re-validate
	year, month, day := t.Date()
	hour, min, _ := t.Clock()

	return v.ValidateLocalTime(year, month, day, hour, min)
}

// ValidateTimeRange validates both start and end times for a schedule.
func (v *TimezoneValidator) ValidateTimeRange(startTs, endTs int64) *TimeRangeValidationResult {
	var warnings []string

	startResult := v.ValidateTimestamp(startTs)
	warnings = append(warnings, startResult.Warnings...)

	var endResult *ValidationResult
	if endTs > 0 {
		endResult = v.ValidateTimestamp(endTs)
		warnings = append(warnings, endResult.Warnings...)
	}

	return &TimeRangeValidationResult{
		StartValidTime: startResult.ValidTime,
		EndValidTime:   endResult.ValidTime,
		Warnings:       warnings,
		IsValid:        true,
	}
}

// TimeRangeValidationResult contains the result of validating a time range.
type TimeRangeValidationResult struct {
	StartValidTime time.Time
	EndValidTime   time.Time
	Warnings       []string
	IsValid        bool
}

// GetLocation returns the validator's timezone location.
func (v *TimezoneValidator) GetLocation() *time.Location {
	return v.location
}

// GetTimezone returns the validator's timezone name.
func (v *TimezoneValidator) GetTimezone() string {
	return v.timezone
}

// isAmbiguousTime checks if the given time falls into an ambiguous period
// during DST fall back. This happens when the same local time occurs twice.
func (v *TimezoneValidator) isAmbiguousTime(t time.Time) bool {
	// Get the timezone information
	_, offset := t.Zone()

	// Check one hour later
	oneHourLater := t.Add(time.Hour)
	_, offsetLater := oneHourLater.Zone()

	// If the offset is different, we're near a DST transition
	// The ambiguous period is when the offset is about to change
	if offset != offsetLater {
		// We're in the hour before DST ends (fall back)
		// Times in this hour are ambiguous
		return true
	}

	return false
}

// GetDSTTransitionInfo returns information about upcoming DST transitions
// for the given date range.
func (v *TimezoneValidator) GetDSTTransitionInfo(startTs, endTs int64) []*DSTTransition {
	startTime := time.Unix(startTs, 0).UTC()
	endTime := time.Unix(endTs, 0).UTC()

	var transitions []*DSTTransition

	// Check for transitions in the range by examining the zone offset
	current := startTime
	for current.Before(endTime) {
		tInLoc := current.In(v.location)
		zoneName, offset := tInLoc.Zone()

		// Check tomorrow at the same time
		tomorrow := current.Add(24 * time.Hour).In(v.location)
		_, offsetTomorrow := tomorrow.Zone()

		// If offset changed, we found a transition
		if offset != offsetTomorrow {
			// Determine transition type
			var transitionType DSTTransitionType
			if offsetTomorrow > offset {
				transitionType = DSTTransitionFallBack // Clocks go back, hour repeats
			} else {
				transitionType = DSTTransitionSpringForward // Clocks spring forward, hour skipped
			}

			zoneNameNew, _ := tomorrow.Zone()
			transitions = append(transitions, &DSTTransition{
				Time:        tInLoc,
				Type:        transitionType,
				FromOffset:  offset,
				ToOffset:    offsetTomorrow,
				ZoneName:    zoneName,
				ZoneNameNew: zoneNameNew,
			})
		}

		current = current.Add(24 * time.Hour)
	}

	return transitions
}

// DSTTransitionType represents the type of DST transition.
type DSTTransitionType int

const (
	DSTTransitionTypeUnknown   DSTTransitionType = iota
	DSTTransitionSpringForward                   // Clocks move forward (gap in time)
	DSTTransitionFallBack                        // Clocks move back (repeated hour)
)

// DSTTransition represents a DST transition event.
type DSTTransition struct {
	Time        time.Time
	ZoneName    string
	ZoneNameNew string
	Type        DSTTransitionType
	FromOffset  int
	ToOffset    int
}

// String returns a human-readable description of the transition.
func (t *DSTTransition) String() string {
	switch t.Type {
	case DSTTransitionSpringForward:
		return fmt.Sprintf("Spring forward at %s: offset changes from %d to %d seconds (%s to %s)",
			t.Time.Format("2006-01-02 15:04 MST"), t.FromOffset, t.ToOffset, t.ZoneName, t.ZoneNameNew)
	case DSTTransitionFallBack:
		return fmt.Sprintf("Fall back at %s: offset changes from %d to %d seconds (%s to %s)",
			t.Time.Format("2006-01-02 15:04 MST"), t.FromOffset, t.ToOffset, t.ZoneName, t.ZoneNameNew)
	default:
		return fmt.Sprintf("Unknown DST transition at %s", t.Time.Format("2006-01-02 15:04"))
	}
}

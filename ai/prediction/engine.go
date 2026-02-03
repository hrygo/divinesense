// Package prediction provides predictive interaction capabilities for AI agents.
package prediction

import (
	"context"
	"sort"
	"time"

	"github.com/hrygo/divinesense/ai/habit"
	"github.com/hrygo/divinesense/ai/memory"
)

// PredictionType defines the type of prediction.
type PredictionType string

const (
	PredictionTypeAction   PredictionType = "action"
	PredictionTypeQuery    PredictionType = "query"
	PredictionTypeReminder PredictionType = "reminder"
)

// ActionType defines specific actions that can be predicted.
type ActionType string

const (
	ActionViewWeekSchedule ActionType = "view_week_schedule"
	ActionViewTomorrow     ActionType = "view_tomorrow"
	ActionCreateSchedule   ActionType = "create_schedule"
	ActionSetReminder      ActionType = "set_reminder"
	ActionSearchRelated    ActionType = "search_related"
	ActionViewWeeklyReport ActionType = "view_weekly_report"
	ActionMonthlyReview    ActionType = "monthly_review"
	ActionQuickNote        ActionType = "quick_note"
)

// Prediction represents a predicted user action.
type Prediction struct {
	Payload    any            `json:"payload,omitempty"`
	Type       PredictionType `json:"type"`
	Label      string         `json:"label"`
	Action     ActionType     `json:"action"`
	Reason     string         `json:"reason,omitempty"`
	Confidence float64        `json:"confidence"`
}

// TriggerType defines what triggered the prediction.
type TriggerType string

const (
	TriggerTime    TriggerType = "time"
	TriggerContext TriggerType = "context"
	TriggerPattern TriggerType = "pattern"
)

// ContextEvent represents a recent user action for context-based prediction.
type ContextEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	TargetID  string    `json:"target_id"`
}

// Engine provides prediction capabilities based on user habits and context.
type Engine struct {
	habitAnalyzer  habit.HabitAnalyzer
	memoryService  memory.MemoryService
	maxPredictions int
}

// NewEngine creates a new prediction engine.
func NewEngine(habitAnalyzer habit.HabitAnalyzer, memSvc memory.MemoryService) *Engine {
	return &Engine{
		habitAnalyzer:  habitAnalyzer,
		memoryService:  memSvc,
		maxPredictions: 3,
	}
}

// Predict generates predictions for a user based on time, context, and patterns.
func (e *Engine) Predict(ctx context.Context, userID int32, recentEvents []ContextEvent) ([]Prediction, error) {
	var predictions []Prediction

	// 1. Time-based predictions
	timePredictions := e.predictByTime(ctx, userID)
	predictions = append(predictions, timePredictions...)

	// 2. Context-based predictions (based on recent events)
	if len(recentEvents) > 0 {
		contextPredictions := e.predictByContext(ctx, userID, recentEvents)
		predictions = append(predictions, contextPredictions...)
	}

	// 3. Pattern-based predictions (based on historical habits)
	patternPredictions, err := e.predictByPattern(ctx, userID)
	if err == nil {
		predictions = append(predictions, patternPredictions...)
	}

	// Deduplicate and sort by confidence
	predictions = e.deduplicateAndSort(predictions)

	// Return top N predictions
	if len(predictions) > e.maxPredictions {
		predictions = predictions[:e.maxPredictions]
	}

	return predictions, nil
}

// predictByTime generates predictions based on current time.
func (e *Engine) predictByTime(ctx context.Context, userID int32) []Prediction {
	now := time.Now()
	hour := now.Hour()
	weekday := now.Weekday()
	day := now.Day()

	var predictions []Prediction

	// Monday morning: view week schedule
	if weekday == time.Monday && hour >= 8 && hour <= 10 {
		predictions = append(predictions, Prediction{
			Type:       PredictionTypeAction,
			Label:      "查看本周日程",
			Confidence: 0.85,
			Action:     ActionViewWeekSchedule,
			Reason:     "周一早上是查看本周安排的好时机",
		})
	}

	// End of workday: view tomorrow
	if hour >= 17 && hour <= 18 {
		predictions = append(predictions, Prediction{
			Type:       PredictionTypeAction,
			Label:      "明天有什么安排？",
			Confidence: 0.75,
			Action:     ActionViewTomorrow,
			Reason:     "下班前预览明日日程",
		})
	}

	// Morning: quick note
	if hour >= 8 && hour <= 9 {
		predictions = append(predictions, Prediction{
			Type:       PredictionTypeAction,
			Label:      "记录今日想法",
			Confidence: 0.65,
			Action:     ActionQuickNote,
			Reason:     "早晨适合记录灵感",
		})
	}

	// Friday afternoon: weekly report
	if weekday == time.Friday && hour >= 15 && hour <= 17 {
		predictions = append(predictions, Prediction{
			Type:       PredictionTypeAction,
			Label:      "本周回顾",
			Confidence: 0.80,
			Action:     ActionViewWeeklyReport,
			Reason:     "周五下午适合回顾本周",
		})
	}

	// End of month: monthly review
	if day >= 28 {
		predictions = append(predictions, Prediction{
			Type:       PredictionTypeAction,
			Label:      "本月笔记回顾",
			Confidence: 0.70,
			Action:     ActionMonthlyReview,
			Reason:     "月底适合整理本月内容",
		})
	}

	return predictions
}

// predictByContext generates predictions based on recent user actions.
func (e *Engine) predictByContext(ctx context.Context, userID int32, events []ContextEvent) []Prediction {
	var predictions []Prediction

	// Only consider recent events (within last 5 minutes)
	recentCutoff := time.Now().Add(-5 * time.Minute)

	for _, event := range events {
		if event.Timestamp.Before(recentCutoff) {
			continue
		}

		switch event.Type {
		case "schedule_created":
			// After creating a schedule, suggest setting a reminder
			predictions = append(predictions, Prediction{
				Type:       PredictionTypeAction,
				Label:      "设置提醒",
				Confidence: 0.90,
				Action:     ActionSetReminder,
				Payload:    map[string]string{"schedule_id": event.TargetID},
				Reason:     "刚创建的日程可能需要提醒",
			})

		case "memo_viewed":
			// After viewing a memo, suggest searching related
			predictions = append(predictions, Prediction{
				Type:       PredictionTypeQuery,
				Label:      "搜索相关笔记",
				Confidence: 0.70,
				Action:     ActionSearchRelated,
				Payload:    map[string]string{"memo_id": event.TargetID},
				Reason:     "查看笔记后可能想找相关内容",
			})

		case "schedule_completed":
			// After completing a schedule, suggest creating next
			predictions = append(predictions, Prediction{
				Type:       PredictionTypeAction,
				Label:      "创建后续日程",
				Confidence: 0.65,
				Action:     ActionCreateSchedule,
				Reason:     "完成日程后可能有后续安排",
			})
		}
	}

	return predictions
}

// predictByPattern generates predictions based on historical patterns.
func (e *Engine) predictByPattern(ctx context.Context, userID int32) ([]Prediction, error) {
	var predictions []Prediction

	// Analyze user habits
	if e.habitAnalyzer == nil {
		return predictions, nil
	}

	habits, err := e.habitAnalyzer.Analyze(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	hour := now.Hour()
	weekday := now.Weekday()

	// Check if current time matches user's active hours
	if habits != nil && habits.Time != nil {
		for _, activeHour := range habits.Time.ActiveHours {
			if activeHour == hour {
				// User is typically active at this hour
				predictions = append(predictions, Prediction{
					Type:       PredictionTypeAction,
					Label:      "创建新日程",
					Confidence: 0.60,
					Action:     ActionCreateSchedule,
					Reason:     "您通常在这个时间创建日程",
				})
				break
			}
		}

		// Weekend preference (inverse of weekday pattern)
		isWeekend := weekday == time.Saturday || weekday == time.Sunday
		if !habits.Time.WeekdayPattern && isWeekend {
			predictions = append(predictions, Prediction{
				Type:       PredictionTypeAction,
				Label:      "周末安排",
				Confidence: 0.65,
				Action:     ActionCreateSchedule,
				Reason:     "您喜欢在周末安排活动",
			})
		}
	}

	return predictions, nil
}

// deduplicateAndSort removes duplicate actions and sorts by confidence.
func (e *Engine) deduplicateAndSort(predictions []Prediction) []Prediction {
	// Use action as key for deduplication
	seen := make(map[ActionType]bool)
	var unique []Prediction

	for _, p := range predictions {
		if !seen[p.Action] {
			seen[p.Action] = true
			unique = append(unique, p)
		}
	}

	// Sort by confidence descending
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Confidence > unique[j].Confidence
	})

	return unique
}

// SetMaxPredictions sets the maximum number of predictions to return.
func (e *Engine) SetMaxPredictions(n int) {
	if n > 0 && n <= 10 {
		e.maxPredictions = n
	}
}

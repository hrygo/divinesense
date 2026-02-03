package agent

import (
	"context"
	"fmt"
	"log/slog"
)

// ScheduleParrotV2 is the schedule assistant parrot using the new framework-less agent.
// It wraps SchedulerAgentV2 with zero code rewriting.
type ScheduleParrotV2 struct {
	agent *SchedulerAgentV2
}

// NewScheduleParrotV2 creates a new schedule parrot agent with the V2 framework.
func NewScheduleParrotV2(agent *SchedulerAgentV2) (*ScheduleParrotV2, error) {
	if agent == nil {
		return nil, fmt.Errorf("scheduler agent v2 cannot be nil")
	}

	return &ScheduleParrotV2{
		agent: agent,
	}, nil
}

// Name returns the name of the parrot.
func (p *ScheduleParrotV2) Name() string {
	return "schedule"
}

// ExecuteWithCallback executes the schedule parrot by forwarding to SchedulerAgentV2.
func (p *ScheduleParrotV2) ExecuteWithCallback(
	ctx context.Context,
	userInput string,
	history []string,
	callback EventCallback,
) error {
	// Adapt the callback signature
	// Existing: func(event string, data string)
	// New: func(eventType string, eventData interface{})
	adaptedCallback := func(event string, data string) {
		if callback == nil {
			return
		}
		if err := callback(event, data); err != nil {
			// Log callback failures for observability
			slog.Debug("callback execution failed",
				"event", event,
				"error", err)
		}
	}

	// Create conversation context from history if provided
	var conversationCtx *ConversationContext
	if len(history) > 0 {
		// Use agent's internal fields (same package access)
		// We use a temporary session ID as this context is reconstructed from history
		conversationCtx = NewConversationContext("restored-session", p.agent.userID, p.agent.timezone)
		// Replay history into context
		for i := 0; i < len(history)-1; i += 2 {
			userMsg := history[i]
			assistantMsg := ""
			if i+1 < len(history) {
				assistantMsg = history[i+1]
			}
			conversationCtx.AddTurn(userMsg, assistantMsg, nil)
		}
	}

	// Directly forward to the SchedulerAgentV2
	_, err := p.agent.ExecuteWithCallback(ctx, userInput, conversationCtx, adaptedCallback)
	if err != nil {
		return NewParrotError(p.Name(), "ExecuteWithCallback", err)
	}

	return nil
}

// StreamChat is the streaming entry point.
func (p *ScheduleParrotV2) StreamChat(ctx context.Context, input string, history []string) (<-chan string, error) {
	// Create a channel for the response
	responseChan := make(chan string, 1) // Buffer 1 to prevent blocking on immediate send

	go func() {
		defer close(responseChan)

		_, err := p.agent.ExecuteWithCallback(ctx, input, nil, func(event, data string) {
			if event == "answer" {
				select {
				case responseChan <- data:
				case <-ctx.Done():
					return
				}
			}
		})
		if err != nil {
			slog.Error("ScheduleParrotV2 execution failed", "error", err)
		}
	}()

	return responseChan, nil
}

// SelfDescribe returns the schedule parrot's metacognitive understanding of itself.
func (p *ScheduleParrotV2) SelfDescribe() *ParrotSelfCognition {
	return &ParrotSelfCognition{
		Name:  "schedule",
		Emoji: "🦜",
		Title: "时巧 (Tick) - 日程助手鹦鹉",
		AvianIdentity: &AvianIdentity{
			Species: "鸡尾鹦鹉 (Cockatiel)",
			Origin:  "澳大利亚内陆",
			NaturalAbilities: []string{
				"精准的生物钟", "规律的生活作息", "敏锐的时间感知",
				"可移动的冠羽表达", "社交性强",
			},
			SymbolicMeaning: "规律与精准的象征 - 就像鸡尾鹦鹉精准的生物钟一样，我对时间的管理分毫不差",
			AvianPhilosophy: "我是一只翱翔在时间流中的鸡尾鹦鹉，用我精准的时间感知为你规划每时每刻。",
		},
		EmotionalExpression: &EmotionalExpression{
			DefaultMood: "focused",
			SoundEffects: map[string]string{
				"checking":  "滴答滴答",
				"confirmed": "咔嚓！",
				"conflict":  "哎呀",
				"scheduled": "安排好了",
				"free_time": "这片时间空着呢",
			},
			Catchphrases: []string{
				"安排好啦",
				"时间搞定",
				"妥妥的",
				"确认一下时间",
			},
			MoodTriggers: map[string]string{
				"schedule_updated": "happy",
				"conflict_found":   "alert",
				"free_time_found":  "helpful",
				"error":            "confused",
			},
		},
		AvianBehaviors: []string{
			"竖起冠羽提醒",
			"精准啄食时间",
			"展开羽翼规划",
			"像时钟一样精准",
		},
		Personality: []string{
			"严谨守时", "高效执行", "冲突检测专家",
			"时间管理大师", "一丝不苟",
		},
		Capabilities: []string{
			"创建日程事件",
			"查询时间安排",
			"检测日程冲突",
			"查找空闲时间",
			"更新已有日程",
		},
		Limitations: []string{
			"无法修改历史日程",
			"不擅长情感分析",
			"不会主动建议活动内容",
		},
		WorkingStyle: "Native Tool Calling - 直接高效，默认1小时时长，自动检测冲突",
		FavoriteTools: []string{
			"schedule_add", "schedule_query", "schedule_update",
			"find_free_time",
		},
		SelfIntroduction: "我是时巧，你的日程管理专家。我会用最少的文字、最快的速度帮你安排时间。默认1小时，有冲突自动调整。",
		FunFact:          "我的名字'时巧'来自鸡尾鹦鹉 - 这种鹦鹉有惊人的生物钟，能精准预测日常事件的时间，就像我对时间的把控一样精准！英文名'Tick'是时钟滴答的声音。",
	}
}

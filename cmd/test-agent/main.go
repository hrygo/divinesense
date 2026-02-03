package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agent"
	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/server/service/schedule"
	"github.com/hrygo/divinesense/store"
	"github.com/hrygo/divinesense/store/db"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 1. 加载配置
	log.Println("加载配置...")
	profile := &profile.Profile{
		Mode:   "dev",
		Driver: "postgres",
		DSN:    "postgres://memos:memos@localhost:25432/memos?sslmode=disable",
		Data:   "./memos",
		Port:   28081,
	}
	profile.FromEnv()

	// 2. 初始化数据库驱动
	log.Println("初始化数据库...")
	dbDriver, err := db.NewDBDriver(profile)
	if err != nil {
		log.Fatalf("Failed to create db driver: %v", err)
	}

	// 3. 初始化 Store
	log.Println("初始化 Store...")
	storeInstance := store.New(dbDriver, profile)

	// 3. 初始化 LLM 服务
	log.Println("初始化 LLM 服务...")
	if !profile.IsAIEnabled() {
		log.Fatal("AI is not enabled. Please set MEMOS_AI_ENABLED=true in .env")
	}

	llmCfg := ai.LLMConfig{
		Provider: profile.AILLMProvider,
		APIKey:   profile.AIDeepSeekAPIKey,
		BaseURL:  profile.AIDeepSeekBaseURL,
		Model:    profile.AILLMModel,
	}

	llmService, err := ai.NewLLMService(&llmCfg)
	if err != nil {
		log.Fatalf("Failed to create LLM service: %v", err)
	}

	// 4. 创建 Schedule Service
	log.Println("创建 Schedule Service...")
	scheduleSvc := schedule.NewService(storeInstance)

	// 5. 创建 Scheduler Agent
	log.Println("创建 Scheduler Agent...")
	userID := int32(1) // 测试用户 ID
	userTimezone := "Asia/Shanghai"

	agent, err := agent.NewSchedulerAgentV2(llmService, scheduleSvc, userID, userTimezone)
	if err != nil {
		log.Fatalf("Failed to create scheduler agent: %v", err)
	}

	// 6. 运行测试
	ctx := context.Background()

	fmt.Println("\n========================================")
	fmt.Println("  日程智能体测试程序")
	fmt.Println("========================================")
	fmt.Println()

	// 测试用例
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "查询明天的日程",
			input:    "查看明天有什么安排",
			expected: "应该返回明天的日程列表或提示暂无日程",
		},
		{
			name:     "创建新日程",
			input:    "后天上午10点开个产品讨论会",
			expected: "应该成功创建日程并确认",
		},
		{
			name:     "查询本周日程",
			input:    "本周有哪些日程安排？",
			expected: "应该返回本周的日程列表",
		},
	}

	// 执行测试
	for i, test := range tests {
		fmt.Printf("\n[测试 %d/%d] %s\n", i+1, len(tests), test.name)
		fmt.Println("输入:", test.input)
		fmt.Println("预期:", test.expected)
		fmt.Println("执行中...")

		// 使用带回调的执行模式
		startTime := time.Now()
		response, err := agent.ExecuteWithCallback(ctx, test.input, nil, func(eventType, eventData string) {
			switch eventType {
			case "thinking":
				fmt.Println("  [思考中]", eventData)
			case "tool_use":
				fmt.Println("  [工具]", eventData)
			case "tool_result":
				fmt.Println("  [结果]", eventData)
			case "schedule_updated":
				fmt.Println("  [通知] 日程已更新")
			case "error":
				fmt.Println("  [错误]", eventData)
			}
		})
		duration := time.Since(startTime)

		if err != nil {
			log.Printf("测试失败: %v\n", err)
			continue
		}

		fmt.Println("响应:", response)
		fmt.Printf("耗时: %v\n", duration)
		fmt.Println("------------------------------------------------")
	}

	fmt.Println("\n========================================")
	fmt.Println("  所有测试完成！")
	fmt.Println("========================================")
}

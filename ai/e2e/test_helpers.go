//go:build e2e_manual
// +build e2e_manual

package e2e

import (
	"os"
	"testing"
)

// RequireManualE2E 运行期防护 - 确保 L2 测试不会被 CI 误执行
// 必须在每个 L2 测试函数的第一行调用
func RequireManualE2E(t *testing.T) {
	t.Helper()

	// 1. 显式跳过 Short 模式 (CI 通常运行 -short)
	if testing.Short() {
		t.Skip("Skipping L2 E2E test in short mode")
	}

	// 2. 检测 CI 环境变量
	if os.Getenv("CI") != "" {
		t.Fatal("CRITICAL: Manual E2E test running in CI environment! Aborting.")
	}

	// 3. 必须显式设置开启开关
	if os.Getenv("ENABLE_MANUAL_E2E") != "true" {
		t.Skip("Skipping L2 E2E test: ENABLE_MANUAL_E2E not set to 'true'")
	}
}

// SkipInCI 在 CI 环境中跳过测试
func SkipInCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI environment")
	}
}

package geek

import "log/slog"

// Shutdown terminates all active CLI sessions managed by GeekParrot and EvolutionParrot.
// This should be called during graceful server shutdown to ensure all Claude Code CLI
// child processes are properly terminated.
// Shutdown 终止 GeekParrot 和 EvolutionParrot 管理的所有活动 CLI 会话。
// 应在优雅服务器关闭期间调用，确保所有 Claude Code CLI 子进程被正确终止。
func Shutdown() {
	slog.Info("Geek package: shutting down all session managers")
	// Shutdown shared session manager for GeekParrot
	shutdownSharedSessionManager()
	// Shutdown all evolution session managers
	shutdownEvolution()
}

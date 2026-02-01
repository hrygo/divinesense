// Package main provides the deployment wizard CLI entry point
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hrygo/divinesense/deploy/interactive"
)

func main() {
	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Println("⚠️  警告: 非 root 用户运行，某些操作可能需要 sudo")
		fmt.Println("建议: sudo " + filepath.Base(os.Args[0]))
		fmt.Println()
	}

	// Run the wizard
	if err := interactive.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ 部署失败: %v\n", err)
		os.Exit(1)
	}
}

# SQLite-Vec 实验性功能归档

本目录包含 DivineSense 在探索 SQLite AI 支持过程中的实验性文档和方案。

## 最终方案

我们采用了 **sqlite-vec 官方 releases + go generate** 的方案：

- **文档**: `docs/research/SQLITE_VEC_OFFICIAL_RELEASES.md`
- **实现**: `store/db/sqlite/sqlite_vec_internal.go`
- **下载**: `store/db/sqlite/download_sqlite_vec.sh`

## 归档内容

### 代码审查报告
- `CODE_REVIEW_SQLITE_AI.md` - 代码审查发现的 P0/P1 问题

### 修复报告
- `FIX_REPORT_SQLITE_AI.md` - 详细修复记录

### 多平台编译研究
- `MULTIPLATFORM_SQLITE_VEC.md` - 多平台编译分析

### 静态链接实现
- `STATIC_LINKING_IMPLEMENTATION.md` - 静态链接方案实验

## 历史背景

1. **初始方案**: 尝试自行编译静态库（已废弃）
2. **中间方案**: init() 编译时下载（已废弃）
3. **最终方案**: 使用官方 releases + go generate（✅ 采用）

## 为什么这些方案被废弃？

1. **自行编译**: 需要维护多平台编译脚本和静态库，成本高
2. **init() 下载**: CGO 编译阶段问题，init() 执行时机晚于 cgo 指令处理
3. **官方 releases**: 简单可靠，由 sqlite-vec 官方维护

---

**归档时间**: 2026-02-06
**最终方案**: docs/research/SQLITE_VEC_OFFICIAL_RELEASES.md

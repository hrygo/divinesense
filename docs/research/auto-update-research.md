# 应用自动更新功能调研报告

> **Issue**: [#30](https://github.com/hrygo/divinesense/issues/30)
> **调研时间**: 2026-02-01
> **状态**: 已完成，待开发

---

## 需求概述

为二进制部署模式的应用添加自动检测新版本并一键升级的能力。

**核心需求**：
- 部署模式：二进制文件
- 更新策略：通知 + 自动执行（可选）
- 检查频率：每天一次后台检查
- 回滚机制：备份旧版本，支持一键回滚
- 通知渠道：前端 UI 提示

---

## 技术可行性

### 评级
✅ **高** - 现有代码基础良好，成熟库支持

### 现有基础（可复用）

| 文件 | 现有能力 | 复用方式 |
|:-----|:---------|:---------|
| `internal/version/version.go` | 版本管理 | 扩展版本比较逻辑 |
| `deploy/aliyun/deploy-binary.sh` | `get_latest_version()` | 迁移到 Go |
| `deploy/aliyun/deploy-binary.sh` | `backup()` / `backup_auto()` | 通过系统命令调用 |
| `web/src/pages/Setting.tsx` | 版本显示 | 扩展为完整更新模块 |

### 推荐技术方案

使用 `creativeprojects/go-selfupdate` 库处理下载和替换：
- GitHub: [creativeprojects/go-selfupdate](https://github.com/creativeprojects/go-selfupdate)
- 支持 GitHub / GitLab / Gitea
- 跨平台支持（Linux, macOS, Windows, ARM）
- 内置校验和验证

---

## 用户价值

| 维度 | 说明 |
|:-----|:-----|
| **解决问题** | 手动更新需 SSH 登录、容易错过重要版本 |
| **目标用户** | 二进制部署用户（生产环境管理员） |
| **使用频率** | 中（更新时使用，后台检查每天一次） |

---

## 竞品分析

| 产品 | 实现方式 | 备注 |
|:-----|:---------|:-----|
| **Memos** | ❌ 无自动更新 | 需手动 Docker pull |
| **Home Assistant** | ✅ UI 更新按钮 | 前端触发，后台下载 |
| **VS Code** | ✅ 自动检测 + 通知 | 下载后提示重启 |
| **Go CLI 工具** | `go-selfupdate` 库 | 行业标准方案 |

---

## 复杂度评估

| 指标 | 评估 |
|:-----|:-----|
| **工作量** | 1-1.5 人周（后端 2 天 + 前端 2 天 + 测试 1 天） |
| **风险等级** | 中等（权限、进程重启、回滚） |
| **依赖项** | 无前置依赖，可独立开发 |

---

## 风险与缓解

| 风险 | 影响 | 概率 | 措施 |
|:-----|:-----|:-----|:-----|
| 升级失败服务不可用 | 高 | 低 | 自动回滚到备份版本 |
| 权限不足无法写 `/opt/divinesense/bin` | 中 | 中 | 检测权限并提示；支持 sudo 模式 |
| 下载网络超时 | 低 | 中 | 超时重试（3 次）；支持代理配置 |
| SHA256 校验失败 | 中 | 低 | 中止升级，保留备份；告警日志 |
| systemd 重启失败 | 高 | 低 | 检测服务状态；失败时手动提示 |
| 并发升级 | 中 | 低 | 分布式锁；同一时间只允许一个升级进程 |

---

## 技术方案

### 后端架构

```
server/service/update/
├── service.go           # UpdateService 核心逻辑
├── checker.go           # 版本检测（GitHub API）
├── downloader.go        # 二进制下载 + 校验
├── backup.go            # 备份管理（保留 3 个版本）
├── rollback.go          # 回滚逻辑
└── progress.go          # 进度流式推送
```

### 核心流程

```go
// 1. 每日检测（后台 runner）
func (s *UpdateService) StartDailyChecker() {
    ticker := time.NewTicker(24 * time.Hour)
    go func() {
        for range ticker.C {
            release, err := s.CheckForUpdate()
            if err == nil && release != nil {
                s.cache.Set("latest_release", release, 24*time.Hour)
            }
        }
    }()
}

// 2. 升级流程
func (s *UpdateService) PerformUpdate(ctx context.Context) error {
    // 2.1 备份当前版本
    backup, err := s.CreateBackup()

    // 2.2 下载新版本
    binaryPath, err := s.DownloadRelease(release)

    // 2.3 验证 SHA256
    if err := s.VerifyChecksum(binaryPath, release.Asset.SHA256); err != nil {
        return err
    }

    // 2.4 停止服务
    s.StopService()

    // 2.5 替换二进制
    if err := s.ReplaceBinary(binaryPath); err != nil {
        s.Rollback(backup) // 失败自动回滚
        return err
    }

    // 2.6 启动服务
    s.StartService()

    // 2.7 清理旧备份（保留 3 个）
    s.CleanupOldBackups(3)

    return nil
}
```

### API 设计

```protobuf
service UpdateService {
  rpc CheckForUpdate(google.protobuf.Empty) returns (UpdateStatus);
  rpc PerformUpdate(google.protobuf.Empty) returns (stream UpdateProgress);
  rpc Rollback(RollbackRequest) returns (google.protobuf.Empty);
  rpc GetBackups(google.protobuf.Empty) returns (GetBackupsResponse);
}

message UpdateStatus {
  string current_version = 1;
  string latest_version = 2;
  bool has_update = 3;
  int64 last_checked = 4;
  Release release = 5;
}

message UpdateProgress {
  string step = 1;      // backup, download, verify, stop, install, start
  int32 percentage = 2; // 0-100
  string message = 3;
  bool error = 4;
}
```

### 前端组件

```
web/src/components/
├── UpdateBanner.tsx        # 顶部横幅通知
├── UpdateProgressModal.tsx # 升级进度弹窗
└── Settings/
    └── UpdateSection.tsx   # 设置页更新区块

web/src/hooks/
└── useUpdate.ts            # 更新状态管理
```

---

## 参考资源

**技术实现**：
- [creativeprojects/go-selfupdate](https://github.com/creativeprojects/go-selfupdate)
- [rhysd/go-github-selfupdate](https://github.com/rhysd/go-github-selfupdate)
- [Comprehensive Guide to Golang Self-updating binary](https://lokal.so/blog/comprehensive-guide-on-golang-go-self-upgrading-binary/)
- [Golang程序自动更新的终极解决方案](https://blog.csdn.net/gitblog_01161/article/details/155252628)

**UI 设计**：
- [Best Practices for Notifications UI Design - Setproduct](https://www.setproduct.com/blog/notifications-ui-design)
- [Indicators, Validations, and Notifications - Nielsen Norman Group](https://www.nngroup.com/articles/indicators-validations-notifications/)
- [Notification design dos and don'ts - Webflow](https://www.webflow.com/blog/notification-ux)
- [Website Notification Banner - UserGuiding](https://userguiding.com/blog/website-notification-banner)

---

## 元认知自检（5 维度评估）

| 维度 | 评分 | 说明 |
|:-----|:-----|:-----|
| **技术可行性** | 5/5 | 现有代码基础良好，成熟库支持 |
| **用户价值** | 4/5 | 解决真实痛点，但用户群体有限 |
| **实现复杂度** | 4/5 | 中等复杂度，权限处理需谨慎 |
| **调研完整性** | 5/5 | 已覆盖技术、竞品、UI 设计 |
| **方案清晰度** | 5/5 | 架构清晰，可执行 |

**综合评分**: 4.6/5 ✅

---

## 更新记录

| 日期 | 变更 |
|:-----|:-----|
| 2026-02-01 | 初始调研完成，创建 Issue #30 |

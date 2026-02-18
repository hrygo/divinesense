# /explore 页面重构调研

> **Issue**: #262 | **日期**: 2026-02-18 | **状态**: ✅ Issue Created

---

## 背景与问题

`/explore` 页面定位为「公开笔记」—— 无需登录、只读、可评论。但当前实现存在「伪功能」：

| 功能 | 状态 | 问题 |
|:-----|:-----|:-----|
| 分类入口（发现用户、热门） | ❌ 伪 | 路由不存在、参数未实现 |
| 统计卡片 | ❌ 伪 | 硬编码 `{0} {0}` |
| 禅意设计 | ✅ | 与「公开广场」定位略有偏差 |

## 产品定位澄清

通过用户访谈确定：

1. **评论权限**：需登录（与现有 API 一致）
2. **公开定义**：默认私有，用户显式选择公开
3. **MVP 功能**：
   - ✅ 作者信息展示
   - ✅ 标签/分类筛选
4. **设计风格**：保留禅意风格，去除伪功能

## 技术分析

### 已有能力（存真）

| 功能 | 现状 | 复用方式 |
|:-----|:-----|:---------|
| MemoBlockV3 | `showCreator={true}` | 直接复用 |
| MemoExplorer 侧边栏 | context="explore" | 已接入 |
| Visibility 过滤 | 访客只看 PUBLIC | 逻辑正确 |
| 评论 API | CreateMemoComment | 需登录 |

### 需删除（去伪）

1. **QuickFilterCard 组件**
   - `/explore/users` → 404
   - `?trending` → 未实现

2. **ZenStatCard 组件**
   - `totalMemos={0}` 硬编码
   - `totalUsers={0}` 硬编码

3. **i18n key**
   - `explore.categories.*`
   - `explore.stats.*`

## 方案设计

### 功能边界

**保留**：
- 禅意设计风格（breathe 动画、背景光晕）
- MemoListV3 笔记列表
- MemoExplorer 侧边栏（标签筛选）
- 评论功能入口

**删除**：
- QuickFilterCard 分类入口
- ZenStatCard 统计卡片
- 相关 i18n key

### 代码改动

```
web/src/components/Memo/ExploreHeroSection.tsx
├── 删除 QuickFilterCard 组件及引用
├── 删除 ZenStatCard 组件及引用
├── 保留禅意背景动画
└── 简化为：标题 + 副标题 + 禅意背景

web/src/locales/zh-Hans.json
web/src/locales/en.json
└── 删除 explore.categories.* 和 explore.stats.*
```

## 复杂度

- **工作量**: 0.5 人日
- **风险**: 低（仅 UI 简化）
- **后端**: 无需修改

## 参考

- Issue: https://github.com/hrygo/divinesense/issues/262
- 相关组件：`MemoBlockV3.tsx`, `MemoListV3.tsx`, `MemoExplorer.tsx`

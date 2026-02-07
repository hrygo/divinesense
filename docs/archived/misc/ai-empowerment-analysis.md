# Memos AI 赋能分析报告（笔记 + 附件模块）

> 分析范围：笔记管理、附件管理
> 生成日期：2026-01-26

## 一、笔记管理模块现状

### 1.1 核心组件架构

```
MemoEditor/                     # 编辑器
├── Editor/
│   ├── index.tsx              # 核心 textarea
│   ├── TagSuggestions.tsx     # 标签自动补全
│   └── SlashCommands.tsx      # 斜杠命令
├── Toolbar/
│   ├── InsertMenu.tsx         # 插入菜单（文件/链接/位置）
│   └── VisibilitySelector.tsx # 可见性选择
├── components/
│   ├── AttachmentList.tsx     # 附件预览
│   ├── RelationList.tsx       # 关联笔记
│   └── LocationDisplay.tsx    # 位置显示
└── hooks/
    ├── useAutoSave.ts         # 自动保存
    └── useFocusMode.ts        # 专注模式
```

### 1.2 现有功能清单

| 功能 | 实现位置 | AI 相关 |
|-----|---------|--------|
| Markdown 编辑 | `Editor/index.tsx` | 否 |
| 标签自动补全 | `TagSuggestions.tsx` | 否（基于历史标签） |
| 斜杠命令 | `SlashCommands.tsx` | 否 |
| 附件上传 | `InsertMenu.tsx` | 否 |
| 笔记关联 | `LinkMemoDialog.tsx` | 否 |
| 位置标记 | `LocationDialog.tsx` | 否 |
| 专注模式 | `useFocusMode.ts` | 否 |
| 自动保存 | `useAutoSave.ts` | 否 |

### 1.3 已有 AI API（后端）

| API | 功能 | 使用场景 |
|-----|------|---------|
| `SemanticSearch` | 向量搜索笔记 | 搜索栏 |
| `SuggestTags` | AI 推荐标签 | **未在编辑器中使用** |
| `GetRelatedMemos` | 查找相关笔记 | MemoDetail 页面 |

**关键发现：**`SuggestTags` API 已实现但**未集成到编辑器**中。

---

## 二、附件管理模块现状

### 2.1 核心组件

```
Attachments.tsx                 # 附件管理页
├── 按月份分组展示
├── 未关联附件区域
├── 搜索过滤（仅文件名）
└── 批量删除未关联附件

AttachmentIcon.tsx              # 图标显示（基于 MIME）
```

### 2.2 数据模型（store/attachment.go）

```go
type Attachment struct {
    // ...
    ExtractedText string // PDF/Office 文本提取（预留）
    OCRText       string // 图片 OCR 文本（预留）
    ThumbnailPath string // 缩略图路径
}
```

**关键发现：**`ExtractedText` 和 `OCRText` 字段**已预留但未实现**。

### 2.3 当前问题

| 问题 | 影响 |
|-----|------|
| 搜索仅支持文件名 | 无法搜索图片内容、PDF 内容 |
| 无内容预览 | 必须下载才能查看 |
| 无 AI 分类 | 手动组织 |

---

## 三、AI 赋能方案

### 3.1 编辑器 AI 辅助（P0 优先级）

#### 3.1.1 AI 标签推荐集成

**现状：**
- `SuggestTags` API 已存在（`ai_service_semantic.go:133`）
- 前端 hook `useSuggestTags` 已存在（`useAIQueries.ts:46`）
- **但编辑器未调用**

**实现方案：**
```tsx
// MemoEditor/Toolbar/AITagButton.tsx
const AITagButton = () => {
  const { mutate: suggestTags, isPending } = useSuggestTags();
  const { state, dispatch, actions } = useEditorContext();
  
  const handleClick = () => {
    suggestTags({ content: state.content }, {
      onSuccess: (tags) => {
        // 展示 Popover 让用户选择
      }
    });
  };
};
```

**UI 设计：**
- 在 `EditorToolbar` 添加 AI 按钮
- 点击后显示推荐标签 Popover
- 用户点击标签直接插入 `#tag`

#### 3.1.2 Inline AI 辅助（选中文本增强）

**功能：**
- 选中文本 -> 浮动工具栏
- 操作：润色 / 扩写 / 缩写 / 解释 / 翻译

**实现方案：**
```tsx
// MemoEditor/Editor/AIFloatingToolbar.tsx
// 监听 selection change，显示浮动按钮
// 调用新增 API: EnhanceText(content, action)
```

**后端新增 API：**
```go
// ai_service_enhance.go
func (s *AIService) EnhanceText(ctx context.Context, req *v1pb.EnhanceTextRequest) (*v1pb.EnhanceTextResponse, error) {
    // action: "polish" | "expand" | "summarize" | "explain" | "translate"
}
```

#### 3.1.3 智能续写（Tab 自动补全）

**功能：**
- 输入暂停 1s -> 显示灰色建议文本
- 按 Tab 接受

**实现思路：**
- 使用 debounce 检测输入暂停
- 调用 LLM 获取续写建议
- 在编辑器显示半透明提示

---

### 3.2 附件 AI 增强（P1 优先级）

#### 3.2.1 图片 OCR 自动提取

**利用已预留字段：**
```go
type Attachment struct {
    OCRText string // 已预留！
}
```

**实现方案：**
1. 上传图片时调用 OCR 服务（Tesseract / 云服务）
2. 将提取文本存入 `OCRText` 字段
3. 搜索时同时匹配 `filename` 和 `OCRText`

**后端新增：**
```go
// plugin/ai/ocr/service.go
type OCRService interface {
    ExtractText(ctx context.Context, imageData []byte) (string, error)
}
```

#### 3.2.2 PDF/文档内容提取

**实现方案：**
1. 上传 PDF/Office 时调用 Tika 或类似服务
2. 存入 `ExtractedText` 字段
3. 支持内容搜索

#### 3.2.3 附件智能搜索

**升级搜索功能：**
```tsx
// 现有：仅文件名
const filterAttachments = (attachments, query) => 
  attachments.filter(a => a.filename.includes(query));

// 升级：文件名 + OCR + 提取文本
const filterAttachments = (attachments, query) => 
  attachments.filter(a => 
    a.filename.includes(query) ||
    a.ocrText?.includes(query) ||
    a.extractedText?.includes(query)
  );
```

#### 3.2.4 图片内容描述（可选）

**功能：**
- 上传图片后，AI 生成描述
- 用于辅助搜索和无障碍

---

### 3.3 笔记 + 附件联动（P2 优先级）

#### 3.3.1 智能笔记创建

**场景：**上传图片 -> 自动生成包含 OCR 内容的笔记

```
用户上传截图
    |
OCR 提取文字
    |
AI 整理格式
    |
自动创建笔记草稿
```

#### 3.3.2 附件内容融入语义搜索

**现状：**语义搜索仅检索笔记内容

**升级：**
- 将 OCRText/ExtractedText 纳入 Embedding
- 搜索时同时匹配笔记和附件

---

## 四、实施优先级

| 阶段 | 功能 | 工作量 | 价值 |
|-----|------|-------|-----|
| **Phase 1** | AI 标签推荐集成 | S（API 已有） | 高 |
| **Phase 1** | Inline AI 工具栏 | M | 高 |
| **Phase 2** | 图片 OCR | M | 中 |
| **Phase 2** | 附件智能搜索 | S | 中 |
| **Phase 3** | PDF 内容提取 | L | 中 |
| **Phase 3** | 智能续写 | L | 高 |

---

## 五、技术建议

### 5.1 编辑器改造重点

**不破坏现有架构：**
- 新增组件放在 `MemoEditor/AI/` 目录
- 使用 `useEditorContext` 获取/修改内容
- 通过 `EditorRefActions` 操作编辑器

**关键接口（已有）：**
```tsx
interface EditorRefActions {
  insertText: (text, prefix?, suffix?) => void;
  getSelectedContent: () => string;
  getCursorPosition: () => number;
}
```

### 5.2 附件改造重点

**后端存储已就绪：**
- `OCRText` 和 `ExtractedText` 字段已定义
- 只需实现提取逻辑和写入

**API 层新增：**
```protobuf
// proto/api/v1/attachment_service.proto
message ExtractTextRequest {
  string name = 1; // attachments/{uid}
}
message ExtractTextResponse {
  string text = 1;
}
```

---

## 六、总结

### 现有资产

| 模块 | 已有 | 待集成 |
|-----|------|-------|
| 笔记编辑器 | 完整的编辑器组件 | AI 辅助功能 |
| AI 服务 | SuggestTags API | 编辑器 UI |
| 附件存储 | OCRText 字段预留 | 提取逻辑 |

### Quick Win

**立即可做：**
1. 将 `useSuggestTags` 集成到 `EditorToolbar`
2. 在附件搜索中支持 OCRText 字段（当有值时）

### 中期目标

1. Inline AI 工具栏（选中文本增强）
2. 图片上传自动 OCR
3. 附件内容纳入语义搜索

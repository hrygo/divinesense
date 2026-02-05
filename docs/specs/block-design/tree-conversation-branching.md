# Spec: 树状会话分支 (Tree-like Conversation Branching)

> **状态**: 🔲 待开发 | **优先级**: P1 (重要) | **投入**: 6-8 人天
> **Sprint**: Sprint 3-4 | **关联 Issue**: 待创建
> **依赖**: [Unified Block Model](./unified-block-model.md) Phase 1-4

---

## 1. 目标与背景

### 1.1 核心目标

基于 `parent_block_id` 实现对话树状分支功能，允许用户编辑历史消息并创建新的对话分支，解决当前线性对话模型无法支持"编辑并重新生成"场景的问题。

### 1.2 用户价值

- **探索多角度**: 用户可以对同一问题尝试不同的表述，AI 给出不同回复
- **对比效果**: 切换不同分支，对比不同参数/模式下的 AI 响应
- **保留思考**: 编辑历史不会丢失原有对话，所有分支都被完整保留
- **调试友好**: 开发者可调试 Prompt 变化对 AI 输出的影响

### 1.3 技术价值

- **数据模型扩展**: 在 Unified Block Model 基础上增加树状支持
- **向前兼容**: `parent_block_id = NULL` 表示原有线性对话
- **查询优化**: 通过 `branch_path` 字段支持高效的子树查询

---

## 2. 依赖关系

### 2.1 前置依赖（必须完成）

- [x] **Issue #71**: Unified Block Model Phase 1-4（数据库、API、前端类型、前端组件）
- [x] **[unified-block-model_improvement](./unified-block-model_improvement.md)**: 必须完成以确保时间戳正确 (P0)
- [ ] **[P1-A006-llm-stats-collection](./P1-A006-llm-stats-collection.md)**: 必须完成 `LLMService` 接口重构 (P1)
- [x] **`ai_block` 表**: 已支持 `round_number`、`event_stream`、`user_inputs` 等

### 2.2 并行依赖（可同步进行）

- [ ] **P2-A003**: 会话持久化服务优化
- [ ] **P2-C001**: 智能标签建议

### 2.3 后续依赖（依赖本 Spec）

- [ ] **分支合并功能**: 将两个分支合并为一个（未来扩展）
- [ ] **对话树可视化**: 图形化展示完整对话树（未来扩展）

---

## 3. 功能设计

### 3.1 架构图

```
线性对话 (当前)                    树状对话 (本功能)
─────────────────────────────────────────────────────────────
Block #0                          Block #0 (root)
  │ round_number=0                  │ round_number=0
  │ parent_block_id=NULL           │ parent_block_id=NULL
  │                                 │ branch_path="0"
  │                                 │
Block #1                          ├─ Block #1 (branch A)
  │ round_number=1                 │ │ round_number=1
  │ parent_block_id=NULL           │ │ parent_block_id=0
  │                                 │ │ branch_path="0/1"
  │                                 │ │
Block #2                          │ │ └─ Block #3 (branch A 继续)
  │ round_number=2                 │ │     round_number=2
  │ parent_block_id=NULL           │ │     parent_block_id=1
  │                                 │ │     branch_path="0/1/2"
  │                                 │ │
  │                                 │ └─ Block #4 (branch B - 用户编辑后重新生成)
  │                                 │     round_number=2
  │                                 │     parent_block_id=1
  │                                 │     branch_path="0/1/2"
  │                                 │
  └─ Block #5 (branch C - 用户在 Block #0 后创建新分支)
      round_number=1
      parent_block_id=0
      branch_path="0/1"
```

### 3.2 核心流程

#### 3.2.1 分支创建流程

```
用户操作：点击历史 Block 的"编辑/重新生成"按钮
    │
    ▼
系统检查：该 Block 是否已有子分支 (parent_block_id = current_block_id)
    │
    ├─ 无子分支 → 直接创建新分支 Block
    │               - parent_block_id = current_block_id
    │               - branch_path 自动计算
    │               - round_number = 父分支的 round_number + 1
    │
    └─ 有子分支 → 显示分支选择器
                    ├─ "覆盖当前分支" → 更新当前活跃分支
                    ├─ "创建新分支" → 创建新的子分支
                    └─ "切换分支" → 加载选中的分支内容
```

#### 3.2.2 分支导航流程

```
用户操作：点击分支点标识（🔀）
    │
    ▼
显示分支选择器 UI
    ├─ 分支列表（显示所有子分支的预览）
    ├─ 当前活跃分支高亮
    └─ "创建新分支"按钮
    │
    ▼
用户选择分支
    │
    ▼
切换视图：更新当前路径标识，重新加载选中分支后的所有 Block
```

### 3.3 关键决策

| 决策点       | 方案 A               | 方案 B                            | 选择  | 理由                           |
| :----------- | :------------------- | :-------------------------------- | :---- | :----------------------------- |
| **分支标识** | 仅 `parent_block_id` | `parent_block_id` + `branch_path` | **B** | `branch_path` 支持高效范围查询 |
| **分支存储** | 独立表               | 同表扩展                          | **B** | 避免跨表 JOIN，简化查询        |
| **UI 展示**  | 树状图               | 内联切换器                        | **B** | 降低前端复杂度，渐进式增强     |
| **分支删除** | 级联删除子分支       | 仅删除当前分支                    | **A** | 保持数据一致性，简化逻辑       |
| **分支合并** | 支持                 | 不支持                            | **B** | 未来扩展，MVP 不需要           |

---

## 4. 技术实现

### 4.1 数据模型

#### 4.1.1 数据库变更

```sql
-- =============================================================================
-- Tree-like Conversation Branching (V0.65.0)
-- =============================================================================

-- 添加树状结构字段
ALTER TABLE ai_block ADD COLUMN parent_block_id BIGINT;
ALTER TABLE ai_block ADD CONSTRAINT fk_ai_block_parent
    FOREIGN KEY (parent_block_id)
    REFERENCES ai_block(id)
    ON DELETE CASCADE;

-- 添加分支路径字段（用于高效查询）
ALTER TABLE ai_block ADD COLUMN branch_path TEXT;
-- 格式: "0/1/2" 表示 root -> block_1 -> block_2 的路径
-- 每个数字表示在该层级的位置（从 0 开始）

-- 添加外键索引
CREATE INDEX idx_ai_block_parent ON ai_block(parent_block_id);

-- 添加分支路径索引（用于范围查询）
CREATE INDEX idx_ai_block_branch_path ON ai_block(branch_path) WHERE branch_path IS NOT NULL;

-- 添加当前活跃路径标识（用于快速定位活跃分支）
-- 修正：存储在 Conversation 表中，避免批量更新 Block Metadata
ALTER TABLE ai_conversation ADD COLUMN current_leaf_block_id BIGINT;

-- =============================================================================
-- 关键修复: 调整 Auto-Round 触发器
-- =============================================================================

-- 原有触发器 (v0.60.1) 是基于 MAX(round) + 1，这在树状结构中是错误的。
-- 修订逻辑：优先使用应用层传入的 Round；如果有 Parent，则使用 Parent.Round + 1。

CREATE OR REPLACE FUNCTION ai_block_auto_round_number()
RETURNS TRIGGER AS $$
DECLARE
    parent_round INTEGER;
BEGIN
    -- 1. 如果应用层已指定 Round (显式插入)，则尊重应用层的值
    IF NEW.round_number IS NOT NULL AND NEW.round_number > 0 THEN
        RETURN NEW;
    END IF;

    -- 2. 树状逻辑：如果有 Parent，Round = Parent.Round + 1
    IF NEW.parent_block_id IS NOT NULL THEN
        SELECT round_number INTO parent_round FROM ai_block WHERE id = NEW.parent_block_id;
        NEW.round_number := COALESCE(parent_round, 0) + 1;
        RETURN NEW;
    END IF;

    -- 3. 兼容逻辑 (根分支追加)：维持原有的 MAX + 1
    -- ... (保留原有逻辑作为 Fallback) ...
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 版本更新
-- =============================================================================

INSERT INTO system_setting (name, value, description) VALUES
('schema_version', '0.65.0', 'Database schema version - Tree-like Conversation Branching')
ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;
```

#### 4.1.2 路径计算 (APP Layer)

由于 `branch_path` 的计算依赖于父节点状态且需要事务保护（DB Trigger 在各类边界条件下难以维护），逻辑上移至 Go 代码层：

```go
func (s *Store) CreateForkBlock(ctx context.Context, parentID int64, ...) {
    return s.db.Transaction(func(tx *Tx) error {
        // 1. Get Parent Info & Path
        parent, _ := tx.GetBlock(parentID)
        
        // 2. Count Siblings
        siblingCount, _ := tx.CountChildren(parentID)
        childIndex := siblingCount + 1
        
        // 3. Construct Path
        newPath := fmt.Sprintf("%s/%d", parent.BranchPath, childIndex)
        
        // 4. Insert Block with Explicit Path and Round
        block.BranchPath = newPath
        block.Round = parent.Round + 1
        return tx.Insert(block)
    })
}
```

### 4.2 接口定义

#### 4.2.1 Proto 扩展

```protobuf
// =============================================================================
// Tree-like Conversation Branching Messages
// =============================================================================

// 扩展 Block 消息
message Block {
  // ... 现有字段

  // Tree structure support (V0.65.0)
  int64 parent_block_id = 17;           // Parent block for branching
  string branch_path = 18;              // Path string like "0/1/2"
  repeated int64 child_block_ids = 19;  // Cached child IDs for UI
  bool is_active_path = 20;             // Whether this is on current active path
}

// ForkBlockRequest creates a new block as a branch of an existing block
message ForkBlockRequest {
  int64 source_block_id = 1;            // Block to fork from
  repeated UserInput user_inputs = 2;   // New user inputs
  BlockMode mode = 3;                    // Mode for the new block
  string metadata = 4;                  // Additional metadata (JSON)
}

// ForkBlockResponse returns the newly created block
message ForkBlockResponse {
  Block block = 1;                      // The newly created block
  string branch_path = 2;               // The path of this new branch
}

// ListBlockBranchesRequest lists all branches from a block
message ListBlockBranchesRequest {
  int64 block_id = 1;                   // Root block to list branches from
  int32 max_depth = 2;                  // Maximum depth to traverse (default: 3)
  int32 conversation_id = 2;            // Conversation ID (for validation)
}

// ListBlockBranchesResponse returns the branch tree
message ListBlockBranchesResponse {
  repeated BlockBranch branches = 1;   // Root-level branches
  string current_path = 2;             // Currently active path (e.g., "0/1/2")
  int32 total_branches = 3;            // Total number of branches
}

message BlockBranch {
  int64 block_id = 1;
  int64 parent_block_id = 2;
  string branch_path = 3;
  int32 round_number = 4;
  BlockType block_type = 5;
  BlockMode mode = 6;
  BlockStatus status = 7;

  // Preview content
  string user_preview = 8;             // First user input preview
  string assistant_preview = 9;        // Assistant content preview
  int64 created_ts = 10;

  // Tree structure
  repeated BlockBranch children = 11;
  bool is_active = 12;                 // Whether this branch is currently selected
  bool has_children = 13;              // Whether this branch has child branches
}

// SwitchBranchRequest switches to a different branch
message SwitchBranchRequest {
  int32 conversation_id = 1;           // Conversation ID
  string branch_path = 2;              // Target branch path (e.g., "0/2/1")
  int64 block_id = 3;                  // Target block ID (alternative to path)
}

// SwitchBranchResponse returns the blocks on the new path
message SwitchBranchResponse {
  repeated Block blocks = 1;           // Blocks on the new path
  string current_path = 2;             // The new active path
}

// DeleteBranchRequest deletes a branch and all its descendants
message DeleteBranchRequest {
  int64 block_id = 1;                  // Block to delete (root of branch)
  bool delete_descendants = 2;         // Whether to delete all descendants (default: true)
}

// RPC 方法扩展
service AIService {
  // ... 现有 RPC

  // ForkBlock creates a new branch from an existing block
  rpc ForkBlock(ForkBlockRequest) returns (ForkBlockResponse) {
    option (google.api.http) = {
      post: "/api/v1/ai/blocks/fork"
      body: "*"
    };
  }

  // ListBlockBranches lists the branch tree
  rpc ListBlockBranches(ListBlockBranchesRequest) returns (ListBlockBranchesResponse) {
    option (google.api.http) = {
      get: "/api/v1/ai/conversations/{conversation_id}/branches"
    };
  }

  // SwitchBranch switches to a different branch
  rpc SwitchBranch(SwitchBranchRequest) returns (SwitchBranchResponse) {
    option (google.api.http) = {
      post: "/api/v1/ai/branches/switch"
      body: "*"
    };
  }

  // DeleteBranch deletes a branch
  rpc DeleteBranch(DeleteBranchRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      delete: "/api/v1/ai/branches/{block_id}"
    };
  }
}
```

#### 4.2.2 Store 接口

```go
// store/block.go 扩展

// ForkBlock creates a new block as a branch of an existing block
type ForkBlock struct {
    SourceBlockID  int64
    NewUserInputs  []UserInput
    NewMode        BlockMode
    ConversationID int32
    Metadata       string
    CreatedTs      int64
}

// GetBlockTree retrieves the tree structure for a conversation
type GetBlockTree struct {
    ConversationID int32
    MaxDepth       int // Limit tree depth for performance
    RootOnly       bool // Only get root blocks
}

// BlockTreeNode represents a node in the conversation tree
type BlockTreeNode struct {
    Block       *AIBlock
    Children    []*BlockTreeNode
    IsExpanded  bool
    IsActive    bool // Whether this is on the current active path
    Depth       int
}

// GetBranchPath retrieves all blocks on a specific branch path
type GetBranchPath struct {
    ConversationID int32
    BranchPath     string // e.g., "0/1/2"
}

// DeleteBranch deletes a branch and optionally all its descendants
type DeleteBranch struct {
    RootBlockID        int64
    DeleteDescendants  bool
    Cascade            bool // Delete all descendants recursively
}

// 扩展 BlockStore 接口
type BlockStore interface {
    // ... 现有方法

    // ForkBlock creates a new branch from an existing block
    ForkBlock(ctx context.Context, fork *ForkBlock) (*AIBlock, error)

    // GetBlockTree retrieves the tree structure
    GetBlockTree(ctx context.Context, get *GetBlockTree) (*BlockTreeNode, error)

    // GetBranchPath retrieves blocks on a specific path
    GetBranchPath(ctx context.Context, get *GetBranchPath) ([]*AIBlock, error)

    // DeleteBranch deletes a branch
    DeleteBranch(ctx context.Context, del *DeleteBranch) error

    // ListChildBlocks lists direct children of a block
    ListChildBlocks(ctx context.Context, parentBlockID int64) ([]*AIBlock, error)

    // GetActivePath retrieves the currently active path for a conversation
    GetActivePath(ctx context.Context, conversationID int32) (string, error)
}
```

### 4.3 关键代码路径

| 文件路径                                                  | 职责                |
| :-------------------------------------------------------- | :------------------ |
| `store/migration/postgres/V0.65.0__tree_branching.up.sql` | 数据库迁移          |
| `store/block.go`                                          | BlockStore 接口扩展 |
| `store/db/postgres/block_tree.go`                         | 树状查询实现        |
| `server/router/api/v1/ai/branch_handler.go`               | 分支 API 处理器     |
| `web/src/types/block.ts`                                  | 前端类型扩展        |
| `web/src/components/AIChat/BranchIndicator.tsx`           | 分支指示器组件      |
| `web/src/components/AIChat/BranchSelector.tsx`            | 分支选择器组件      |
| `web/src/hooks/useBranchTree.ts`                          | 分支树管理 Hook     |

---

## 5. 前端设计

### 5.1 类型定义

```typescript
// web/src/types/block.ts 扩展

/**
 * Block branch information
 */
export interface BlockBranch {
  id: string;
  parentId: string | null;
  block: AIBlock;
  branchPath: string;
  roundNumber: number;
  isActive: boolean;
  hasChildren: boolean;
  children: BlockBranch[];
  depth: number;

  // Preview content for UI
  userPreview: string;
  assistantPreview: string;
}

/**
 * Conversation tree state
 */
export interface ConversationTree {
  rootBlocks: BlockBranch[];
  currentPath: string[]; // Array of block IDs representing active path
  totalBranches: number;
}

/**
 * Branch operation types
 */
export type BranchOperation =
  | { type: 'fork'; sourceBlockId: string; userInput: string }
  | { type: 'switch'; targetPath: string }
  | { type: 'delete'; blockId: string }
  | { type: 'expand'; blockId: string }
  | { type: 'collapse'; blockId: string };
```

### 5.2 组件设计

#### 5.2.1 BranchIndicator（分支指示器）

```typescript
// web/src/components/AIChat/BranchIndicator.tsx

interface BranchIndicatorProps {
  blockId: string;
  hasBranches: boolean;
  branchCount: number;
  isActive: boolean;
  onBranchClick: (blockId: string) => void;
}

// 显示：
// - 无分支: 不显示
// - 有分支: 显示 � 徽章 + 数量
// - 当前分支: 高亮显示
// - 点击: 打开分支选择器
```

#### 5.2.2 BranchSelector（分支选择器）

```typescript
// web/src/components/AIChat/BranchSelector.tsx

interface BranchSelectorProps {
  branches: BlockBranch[];
  currentPath: string;
  onBranchSelect: (branchPath: string) => void;
  onForkBranch: (sourceBlockId: string, userInput: string) => void;
  onDeleteBranch: (branchId: string) => void;
}

// 显示：
// ┌─────────────────────────────────────────┐
// │ 分支选择                    [×]         │
// ├─────────────────────────────────────────┤
// │ ○ 分支 A (当前)                        │
// │   "如何优化 Go 代码？"                  │
// │   这是优化建议...                       │
// ├─────────────────────────────────────────┤
// │ ○ 分支 B                                │
// │   "Go 代码性能调优有哪些技巧？"         │
// │   性能调优技巧...                       │
// ├─────────────────────────────────────────┤
// │ [+ 创建新分支]                         │
// └─────────────────────────────────────────┘
```

#### 5.2.3 EditMessageDialog（编辑消息对话框）

```typescript
// web/src/components/AIChat/EditMessageDialog.tsx

interface EditMessageDialogProps {
  blockId: string;
  currentContent: string;
  onConfirm: (newContent: string, createBranch: boolean) => void;
  onCancel: () => void;
}

// 显示：
// ┌─────────────────────────────────────────┐
// │ 编辑消息                                │
// ├─────────────────────────────────────────┤
// │ [文本框: 原始消息内容]                  │
// │                                         │
// │ ☑ 创建新分支（保留原消息）             │
// │                                         │
// │ [取消]  [保存]                         │
// └─────────────────────────────────────────┘
```

### 5.3 Hook 设计

```typescript
// web/src/hooks/useBranchTree.ts

export interface UseBranchTreeOptions {
  conversationId: number;
}

export interface UseBranchTreeReturn {
  // Tree data
  tree: ConversationTree | null;
  currentPath: string;

  // Operations
  forkBlock: (sourceBlockId: string, userInput: string) => Promise<Block>;
  switchBranch: (branchPath: string) => Promise<void>;
  deleteBranch: (blockId: string) => Promise<void>;
  refreshTree: () => Promise<void>;

  // UI state
  isForking: boolean;
  isSwitching: boolean;
  error: string | null;
}

export function useBranchTree(
  options: UseBranchTreeOptions
): UseBranchTreeReturn;

// 使用示例
const { tree, currentPath, forkBlock, switchBranch } = useBranchTree({
  conversationId: 123,
});

// 创建新分支
await forkBlock('block_1', '修改后的问题内容');

// 切换分支
await switchBranch('0/2/1');
```

### 5.4 UI 交互流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Chat Messages                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │ Block #0: 如何优化 Go 代码？                        [✏️]   │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │ Block #1: 这是优化建议...                       [✏️]  [🔀2]│  │
│  └─────────────────────────────────────────────────────────────┘  │
│     │                                                    │         │
│     ▼                                                    ▼         │
│  ┌─────────────────────────────────────┐    ┌─────────────────────────────┐
│  │ Block #3: 举个例子...             │    │ Block #4: 有哪些工具？       │
│  │ (当前分支 A)                       │    │ (分支 B)                    │
│  └─────────────────────────────────────┘    └─────────────────────────────┘
│                                                                     │
│  用户点击 Block #0 的 [✏️] → 编辑对话框 → 保存 → 创建新分支           │
│  用户点击 Block #1 的 [🔀2] → 分支选择器 → 切换到分支 B              │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.5 国际化

```json
// web/src/locales/zh-Hans.json 新增
{
  "branches": {
    "title": "分支",
    "create": "创建新分支",
    "switch": "切换分支",
    "delete": "删除分支",
    "deleteConfirm": "确定要删除此分支及其所有后续内容吗？",
    "current": "当前分支",
    "fork": "分支",
    "editAndFork": "编辑并分支",
    "editAndForkDesc": "创建新分支以保留原对话",
    "noBranches": "暂无分支",
    "branchCreated": "分支已创建",
    "branchSwitched": "已切换到分支",
    "branchDeleted": "分支已删除"
  }
}

// web/src/locales/en.json
{
  "branches": {
    "title": "Branches",
    "create": "Create Branch",
    "switch": "Switch Branch",
    "delete": "Delete Branch",
    "deleteConfirm": "Are you sure you want to delete this branch and all its descendants?",
    "current": "Current Branch",
    "fork": "Fork",
    "editAndFork": "Edit & Fork",
    "editAndForkDesc": "Create a new branch to preserve the original conversation",
    "noBranches": "No branches yet",
    "branchCreated": "Branch created",
    "branchSwitched": "Switched to branch",
    "branchDeleted": "Branch deleted"
  }
}
```

---

## 6. 交付物清单

### 6.1 代码文件

**后端**:
- [ ] `store/migration/postgres/V0.65.0__tree_branching.up.sql` - 数据库迁移
- [ ] `store/migration/postgres/V0.65.0__tree_branching.down.sql` - 回滚脚本
- [ ] `store/block.go` - ForkBlock、GetBlockTree 等接口定义
- [ ] `store/db/postgres/block_tree.go` - 树状查询实现
- [ ] `server/router/api/v1/ai/branch_handler.go` - 分支 API 处理器

**前端**:
- [ ] `web/src/types/block.ts` - BlockBranch、ConversationTree 类型
- [ ] `web/src/hooks/useBranchTree.ts` - 分支树管理 Hook
- [ ] `web/src/components/AIChat/BranchIndicator.tsx` - 分支指示器
- [ ] `web/src/components/AIChat/BranchSelector.tsx` - 分支选择器
- [ ] `web/src/components/AIChat/EditMessageDialog.tsx` - 编辑对话框
- [ ] `web/src/components/AIChat/ChatMessages.tsx` - 集成分支 UI

### 6.2 Proto 变更

- [ ] `proto/api/v1/ai_service.proto` - 添加 ForkBlock、ListBlockBranches 等 RPC

### 6.3 文档更新

- [ ] `unified-block-model.md` - 添加树状分支章节
- [ ] `../../dev-guides/FRONTEND.md` - 更新前端组件列表
- [ ] `../../dev-guides/ARCHITECTURE.md` - 更新架构图

---

## 7. 测试验收

### 7.1 功能测试

| 场景             | 输入                            | 预期输出                                  |
| :--------------- | :------------------------------ | :---------------------------------------- |
| **创建分支**     | 用户点击历史 Block 的"重新生成" | 新 Block 创建，`parent_block_id` 设置正确 |
| **分支路径计算** | 创建第 2 个子分支               | `branch_path` 为 "0/2"                    |
| **分支列表**     | 调用 ListBlockBranches          | 返回完整的树状结构                        |
| **切换分支**     | 调用 SwitchBranch               | 视图更新到新分支的内容                    |
| **删除分支**     | 调用 DeleteBranch               | 分支及其子分支被删除                      |
| **编辑并分支**   | 编辑用户输入并保存              | 原内容保留，新分支创建                    |
| **根分支查询**   | 查询 `parent_block_id IS NULL`  | 返回所有根分支                            |

### 7.2 性能验收

| 指标                      | 目标值     | 测试方法            |
| :------------------------ | :--------- | :------------------ |
| ForkBlock 延迟            | < 100ms    | 单元测试            |
| ListBlockBranches (深度3) | < 200ms    | 集成测试            |
| SwitchBranch              | < 300ms    | 包含渲染的 E2E 测试 |
| branch_path 索引查询      | < 50ms     | EXPLAIN ANALYZE     |
| 树深度限制                | 最大 10 层 | 应用层限制          |

### 7.3 集成验收

- [ ] 与 Unified Block Model 集成测试通过
- [ ] 前端与后端 API 对接测试通过
- [ ] `make check-i18n` 通过（翻译完整性）
- [ ] `pnpm lint` 通过（前端代码质量）
- [ ] `go vet ./...` 通过（后端代码质量）

### 7.4 E2E 测试场景

```typescript
// web/e2e/tree-branching.spec.ts

test('create branch from historical block', async ({ page }) => {
  // 1. 打开有历史对话的会话
  await page.goto('/chat/123');

  // 2. 点击第二个 Block 的编辑按钮
  await page.click('[data-testid="block-2"] [data-testid="edit-button"]');

  // 3. 修改内容并选择"创建新分支"
  await page.fill('[data-testid="edit-dialog-input"]', '修改后的问题');
  await page.check('[data-testid="create-branch-checkbox"]');
  await page.click('[data-testid="save-button"]');

  // 4. 验证分支指示器显示
  await expect(page.locator('[data-testid="branch-indicator"]')).toHaveCount(1);

  // 5. 验证分支选择器包含新分支
  await page.click('[data-testid="branch-indicator"]');
  await expect(page.locator('[data-testid="branch-option"]')).toHaveCount(2);
});

test('switch between branches', async ({ page }) => {
  // 1. 打开有分支的会话
  await page.goto('/chat/123');

  // 2. 打开分支选择器
  await page.click('[data-testid="branch-indicator"]');

  // 3. 选择另一个分支
  await page.click('[data-testid="branch-option-2"]');

  // 4. 验证视图更新
  await expect(page.locator('[data-testid="chat-messages"]')).toContainText('分支 B 的内容');
});
```

---

## 8. ROI 分析

| 维度         |                   值                   |
| :----------- | :------------------------------------: |
| **开发投入** |                6-8 人天                |
| **预期收益** | 支持对话分支探索，提升高级用户调试效率 |
| **风险评估** |        中（数据模型复杂度增加）        |
| **回报周期** |                2 Sprint                |

### 用户价值量化

- **目标用户**: 高级用户（每天使用 2+ 小时）
- **使用频率**: 每天创建 2-5 个分支
- **效率提升**: 对比 Prompt 效果的时间从 10 分钟降至 2 分钟

---

## 9. 风险与缓解

| 风险               | 概率 | 影响 | 缓解措施                               |
| :----------------- | :--- | :--- | :------------------------------------- |
| **数据模型复杂化** | 中   | 高   | 渐进式实现：先支持单层分支，再支持多层 |
| **前端 UI 复杂度** | 中   | 中   | 复用现有 `UnifiedMessageBlock` 组件    |
| **查询性能下降**   | 低   | 中   | `branch_path` 索引 + 深度限制 + 缓存   |
| **用户困惑**       | 中   | 低   | 清晰的视觉标识 + 渐进式功能展示        |
| **迁移兼容性**     | 低   | 低   | `parent_block_id IS NULL` 视为根分支   |

---

## 10. 实施计划

### 10.1 时间表

| 阶段        | 时间    | 任务                | 交付物                               |
| :---------- | :------ | :------------------ | :----------------------------------- |
| **Phase 1** | 2人天   | 数据库 + Store 层   | Migration SQL + `block_tree.go`      |
| **Phase 2** | 1.5人天 | Proto + API Handler | Proto 定义 + `branch_handler.go`     |
| **Phase 3** | 1.5人天 | 前端类型 + Hook     | `block.ts` 扩展 + `useBranchTree.ts` |
| **Phase 4** | 2人天   | 前端组件            | `BranchIndicator` + `BranchSelector` |
| **Phase 5** | 1人天   | 集成测试 + Bug 修复 | E2E 测试                             |

**总计**: 8 人天

### 10.2 检查点

- [ ] **Checkpoint 1**: Phase 1 完成 - 数据库迁移成功，树状查询测试通过
- [ ] **Checkpoint 2**: Phase 2 完成 - API 定义完成，ForkBlock 成功创建分支
- [ ] **Checkpoint 3**: Phase 3 完成 - 前端类型定义完成，Hook 可用
- [ ] **Checkpoint 4**: Phase 4 完成 - UI 组件渲染正确，交互流畅
- [ ] **Checkpoint 5**: Phase 5 完成 - E2E 测试全部通过

### 10.3 分阶段交付策略

**MVP（最小可用版本）**:
- ✅ 创建分支（ForkBlock）
- ✅ 分支列表（ListBlockBranches）
- ✅ 切换分支（SwitchBranch）
- ❌ 删除分支（后续版本）
- ❌ 分支可视化（后续版本）

**V1.0 完整版**:
- ✅ MVP 所有功能
- ✅ 删除分支（DeleteBranch）
- ✅ 分支指示器 UI
- ❌ 分支合并（未来扩展）
- ❌ 图形化树视图（未来扩展）

---

## 附录

### A. 参考资料

- [Unified Block Model 规格](./unified-block-model.md)
- [Unified Block Model 改进建议](./unified-block-model_improvement.md)
- [Claude AI — Fork Your AI Conversations](https://www.smithstephen.com/p/fork-your-ai-conversations-why-power)
- [ChatGPT Branching Feature](https://arstechnica.com/ai/2025/09/chatgpts-new-branching-feature-is-a-good-reminder-that-ai-chatbots-arent-people/)
- [Vercel AI SDK — Tree-like Chat History](https://github.com/vercel/ai/issues/2929)
- [Issue #57: 会话嵌套模型](https://github.com/hrygo/divinesense/issues/57)

### B. 变更记录

| 日期       | 版本 | 变更内容                | 作者   |
| :--------- | :--- | :---------------------- | :----- |
| 2026-02-05 | v1.0 | 初始版本 - 完整规格文档 | Claude |

---

**文档状态**: 🔍 待审计

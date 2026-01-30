/**
 * AI Chat 流式和优化组件导出
 *
 * Phase 1: 性能优化
 * - StreamingCodeBlock: 防抖优化的代码高亮
 *
 * Phase 2: 流式渲染增强
 * - StreamingMarkdown: 增量 Markdown 渲染
 * - useStreamingStateMachine: 流式状态管理 Hook
 *
 * Phase 3: 高级交互
 * - useIntentPrediction: 意图预判 Hook
 * - MultiStepWizard: 多步向导组件
 * - PersistentToolContainer: 生成式 UI 持久化
 */

// Phase 1
// export { default as StreamingCodeBlock } from "./StreamingCodeBlock";

// Phase 2
export { default as StreamingMarkdown } from "./StreamingMarkdown";
export {
  default as useStreamingStateMachine,
  StreamingStateIndicator,
  isStreamingState,
  isToolUseState,
  isDoneState,
  isErrorState,
} from "./hooks/useStreamingStateMachine";

// Phase 3
export { useIntentPrediction } from "@/hooks/useIntentPrediction";
export { default as MultiStepWizard } from "./MultiStepWizard";
export { default as PersistentToolContainer } from "./PersistentToolContainer";

// 类型导出
export type { StreamingState } from "./hooks/useStreamingStateMachine";
export type {
  IntentPrediction,
  SuggestedAction,
} from "@/hooks/useIntentPrediction";
export type {
  WizardStep,
  WizardStepProps,
  MultiStepWizardProps,
} from "./MultiStepWizard";

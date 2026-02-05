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
 */

// Phase 1
// export { default as StreamingCodeBlock } from "./StreamingCodeBlock";

export type {
  IntentPrediction,
  SuggestedAction,
} from "@/hooks/useIntentPrediction";
// Phase 3
export { useIntentPrediction } from "@/hooks/useIntentPrediction";
// 类型导出
export type { StreamingState } from "./hooks/useStreamingStateMachine";
export {
  default as useStreamingStateMachine,
  isDoneState,
  isErrorState,
  isStreamingState,
  isToolUseState,
  StreamingStateIndicator,
} from "./hooks/useStreamingStateMachine";
export type {
  MultiStepWizardProps,
  WizardStep,
  WizardStepProps,
} from "./MultiStepWizard";
export { default as MultiStepWizard } from "./MultiStepWizard";
// Phase 2
export { default as StreamingMarkdown } from "./StreamingMarkdown";

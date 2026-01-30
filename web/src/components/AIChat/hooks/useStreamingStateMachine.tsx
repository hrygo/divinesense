import { CheckCircle2, Loader2, MessageSquare, Zap } from "lucide-react";
import { useMemo, useState } from "react";
import { cn } from "@/lib/utils";

/**
 * Phase 2: 流式响应状态机
 *
 * 状态流转:
 * idle -> thinking -> [tool_use?] -> streaming -> done
 *
 * 支持:
 * - 精细的状态展示
 * - 工具调用时的专门视觉反馈
 * - 流式内容的增量更新
 */

export type StreamingState =
  | { status: "idle" }
  | { status: "thinking" }
  | { status: "tool_use"; tool: string; toolData?: unknown }
  | { status: "streaming"; content: string; isComplete: boolean }
  | { status: "done"; content: string }
  | { status: "error"; error: string };

export interface StreamingStateProps {
  state: StreamingState;
  className?: string;
  children?: React.ReactNode;
}

export function useStreamingStateMachine() {
  const [state, setState] = useState<StreamingState>({ status: "idle" });
  const [streamContent, setStreamContent] = useState("");

  const startThinking = () => {
    setState({ status: "thinking" });
    setStreamContent("");
  };

  const useTool = (toolName: string, toolData?: unknown) => {
    setState({ status: "tool_use", tool: toolName, toolData });
  };

  const appendContent = (newContent: string) => {
    setStreamContent((prev) => {
      const updated = prev + newContent;
      setState({ status: "streaming", content: updated, isComplete: false });
      return updated;
    });
  };

  const complete = (finalContent?: string) => {
    const content = finalContent || streamContent;
    setStreamContent(content);
    setState({ status: "done", content });
  };

  const error = (errorMessage: string) => {
    setState({ status: "error", error: errorMessage });
  };

  const reset = () => {
    setState({ status: "idle" });
    setStreamContent("");
  };

  const isStreaming = state.status === "streaming";
  const isThinking = state.status === "thinking" || state.status === "tool_use";
  const isDone = state.status === "done";
  const hasError = state.status === "error";

  return {
    state,
    streamContent,
    isStreaming,
    isThinking,
    isDone,
    hasError,
    startThinking,
    useTool,
    appendContent,
    complete,
    error,
    reset,
  };
}

export function StreamingStateIndicator({ state, className }: StreamingStateProps) {
  const [config, setConfig] = useState<{
    icon: React.ReactNode;
    label: string;
    color: string;
    pulse: boolean;
  }>({
    icon: null,
    label: "",
    color: "",
    pulse: false,
  });

  useMemo(() => {
    switch (state.status) {
      case "idle":
        setConfig({
          icon: null,
          label: "",
          color: "",
          pulse: false,
        });
        break;

      case "thinking":
        setConfig({
          icon: <Loader2 className="w-4 h-4 animate-spin" />,
          label: "思考中...",
          color: "text-blue-500",
          pulse: true,
        });
        break;

      case "tool_use":
        setConfig({
          icon: <Zap className="w-4 h-4" />,
          label: `使用工具: ${state.tool}`,
          color: "text-purple-500",
          pulse: true,
        });
        break;

      case "streaming":
        setConfig({
          icon: <MessageSquare className="w-4 h-4" />,
          label: "输入中...",
          color: "text-green-500",
          pulse: false,
        });
        break;

      case "done":
        setConfig({
          icon: <CheckCircle2 className="w-4 h-4" />,
          label: "完成",
          color: "text-green-500",
          pulse: false,
        });
        break;

      case "error":
        setConfig({
          icon: <span className="text-red-500">⚠️</span>,
          label: "出错了",
          color: "text-red-500",
          pulse: false,
        });
        break;
    }
  }, [state]);

  if (state.status === "idle") return null;

  return (
    <div className={cn("flex items-center gap-2 text-sm", config.color, config.pulse && "animate-pulse", className)}>
      {config.icon}
      <span>{config.label}</span>
    </div>
  );
}

// 导出状态类型辅助函数
export function isStreamingState(state: StreamingState): state is Extract<StreamingState, { status: "streaming" }> {
  return state.status === "streaming";
}

export function isToolUseState(state: StreamingState): state is Extract<StreamingState, { status: "tool_use" }> {
  return state.status === "tool_use";
}

export function isDoneState(state: StreamingState): state is Extract<StreamingState, { status: "done" }> {
  return state.status === "done";
}

export function isErrorState(state: StreamingState): state is Extract<StreamingState, { status: "error" }> {
  return state.status === "error";
}

export default useStreamingStateMachine;

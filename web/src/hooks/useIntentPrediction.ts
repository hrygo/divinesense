import { useCallback, useMemo, useRef, useState } from "react";
import { CapabilityType } from "@/types/capability";

/**
 * Phase 3: 意图预判系统
 *
 * 在用户输入时实时预测意图，提前展示相关 UI 工具
 */

export interface IntentPrediction {
  intent: CapabilityType;
  confidence: number;
  suggestedActions: SuggestedAction[];
  estimatedCompletionTime?: number;
}

export interface SuggestedAction {
  id: string;
  label: string;
  icon: string;
  action: () => void;
  variant: "primary" | "secondary";
}

// 意图关键词模式
const INTENT_PATTERNS = {
  schedule: {
    keywords: ["明天", "今天", "后天", "周", "月", "点", "分", "会议", "日程", "提醒", "安排", "预约", "时间", "上午", "下午", "晚上"],
    weight: 1.0,
  },
  memo: {
    keywords: ["笔记", "记录", "写过", "搜索", "找", "关于", "内容", "标题", "标签"],
    weight: 0.9,
  },
  amazing: {
    keywords: ["总结", "分析", "帮我", "如何", "怎么", "建议", "推荐", "计划"],
    weight: 0.8,
  },
};

interface IntentPredictionOptions {
  onPredict?: (prediction: IntentPrediction) => void;
  debounceMs?: number;
  minConfidence?: number;
}

export function useIntentPrediction(options: IntentPredictionOptions = {}) {
  const { onPredict, debounceMs = 300, minConfidence = 0.3 } = options;

  const [currentPrediction, setCurrentPrediction] = useState<IntentPrediction | null>(null);
  const [inputValue, setInputValue] = useState("");
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [isPredicting, setIsPredicting] = useState(false);

  /**
   * 分析用户输入，预测意图
   */
  const analyzeInput = useCallback(
    (text: string): IntentPrediction | null => {
      if (!text.trim() || text.length < 2) return null;

      const lowerText = text.toLowerCase();
      const scores: Record<string, number> = {
        schedule: 0,
        memo: 0,
        amazing: 0,
      };

      // 基于关键词匹配计算分数
      for (const [intent, pattern] of Object.entries(INTENT_PATTERNS)) {
        for (const keyword of pattern.keywords) {
          if (lowerText.includes(keyword)) {
            scores[intent] += pattern.weight;
          }
        }
      }

      // 时间表达式检测（高权重）
      const timePatterns = [
        /\d{1,2}[:点]\d{1,2}/, // 14:30 或 14点30
        /明天|今天|后天|周[一二三四五六七日]/,
        /[上下]午|晚上|凌晨/,
      ];
      if (timePatterns.some((p) => p.test(text))) {
        scores.schedule += 1.5;
      }

      // 问号通常表示询问
      if (text.includes("?") || text.includes("？")) {
        scores.amazing += 0.5;
      }

      // 找出最高分的意图
      const maxScore = Math.max(...Object.values(scores));
      if (maxScore < minConfidence) return null;

      const predictedIntent = Object.entries(scores).find(([_, score]) => score === maxScore)?.[0];
      if (!predictedIntent) return null;

      const confidence = Math.min(maxScore / 2, 1); // 归一化到 0-1

      return {
        intent: predictedIntent as CapabilityType,
        confidence,
        suggestedActions: generateSuggestedActions(predictedIntent as CapabilityType, text),
      };
    },
    [minConfidence],
  );

  /**
   * 生成建议操作
   */
  const generateSuggestedActions = useCallback((intent: CapabilityType, input: string): SuggestedAction[] => {
    const actions: SuggestedAction[] = [];

    switch (intent) {
      case CapabilityType.SCHEDULE:
        actions.push({
          id: "quick-schedule",
          label: "快速创建日程",
          icon: "Calendar",
          action: () => {},
          variant: "primary",
        });
        // 检测到时间时添加时间选择建议
        if (/\d+/.test(input)) {
          actions.push({
            id: "time-picker",
            label: "选择时间",
            icon: "Clock",
            action: () => {},
            variant: "secondary",
          });
        }
        break;

      case CapabilityType.MEMO:
        actions.push({
          id: "search-memo",
          label: "搜索笔记",
          icon: "Search",
          action: () => {},
          variant: "primary",
        });
        break;

      case CapabilityType.AMAZING:
        actions.push({
          id: "analyze",
          label: "开始分析",
          icon: "Sparkles",
          action: () => {},
          variant: "primary",
        });
        break;
    }

    return actions;
  }, []);

  /**
   * 处理输入变化，带防抖
   */
  const handleInputChange = useCallback(
    (text: string) => {
      setInputValue(text);
      setIsPredicting(true);

      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }

      debounceRef.current = setTimeout(() => {
        const prediction = analyzeInput(text);
        setCurrentPrediction(prediction);

        if (prediction && prediction.confidence >= minConfidence) {
          onPredict?.(prediction);
        }

        setIsPredicting(false);
      }, debounceMs);
    },
    [analyzeInput, debounceMs, minConfidence, onPredict],
  );

  /**
   * 清除预测
   */
  const clearPrediction = useCallback(() => {
    setCurrentPrediction(null);
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
  }, []);

  /**
   * 执行建议的操作
   */
  const executeAction = useCallback(
    (actionId: string) => {
      const action = currentPrediction?.suggestedActions.find((a) => a.id === actionId);
      if (action) {
        action.action();
      }
    },
    [currentPrediction],
  );

  // 计算输入进度（用于显示预测置信度）
  const inputProgress = useMemo(() => {
    if (!inputValue) return 0;
    // 基于输入长度和关键词匹配度计算进度
    const hasKeyword = Object.values(INTENT_PATTERNS).some((pattern) =>
      pattern.keywords.some((kw) => inputValue.toLowerCase().includes(kw)),
    );
    return hasKeyword ? Math.min(inputValue.length / 10, 1) : 0;
  }, [inputValue]);

  return {
    currentPrediction,
    isPredicting,
    inputValue,
    inputProgress,
    handleInputChange,
    clearPrediction,
    executeAction,
    setInputValue,
  };
}

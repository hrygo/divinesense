import { ParrotAgentType } from "./parrot";

/**
 * 能力类型 - 私人助手三核心能力
 */
export enum CapabilityType {
  MEMO = "MEMO", // 笔记检索能力
  SCHEDULE = "SCHEDULE", // 日程管理能力
  AMAZING = "AMAZING", // 综合洞察能力
  AUTO = "AUTO", // 自动识别能力（默认，fallback 到 AMAZING）
}

/**
 * 能力状态 - 用于 UI 展示
 */
export type CapabilityStatus = "idle" | "active" | "thinking" | "processing";

/**
 * 单个能力配置
 */
export interface Capability {
  id: CapabilityType;
  parrotId: ParrotAgentType; // 后台对应的 Agent
  name: string;
  nameAlt: string;
  description: string;
  icon: string;
  color: string;
  soundEffects: Record<string, string>;
  catchphrases: string[];
}

/**
 * 能力状态信息
 */
export interface CapabilityState {
  currentCapability: CapabilityType;
  status: CapabilityStatus;
  lastActivatedAt?: number;
  confidence?: number; // 路由置信度 0-1
}

/**
 * 意图识别结果
 */
export interface IntentRecognitionResult {
  capability: CapabilityType;
  confidence: number;
  reasoning?: string;
}

/**
 * 能力配置映射 - 私人助手三核心能力
 */
export const CAPABILITIES: Record<CapabilityType, Omit<Capability, "id">> = {
  [CapabilityType.MEMO]: {
    parrotId: ParrotAgentType.MEMO,
    name: "笔记",
    nameAlt: "Memo",
    description: "搜索与问答",
    icon: "🦜",
    color: "slate",
    soundEffects: {
      thinking: "嘎...",
      searching: "扑棱扑棱",
      found: "嗯嗯~",
      done: "扑棱！",
    },
    catchphrases: ["让我想想...", "笔记里说...", "在记忆里找找..."],
  },
  [CapabilityType.SCHEDULE]: {
    parrotId: ParrotAgentType.SCHEDULE,
    name: "日程",
    nameAlt: "Schedule",
    description: "规划与管理",
    icon: "⏰",
    color: "cyan",
    soundEffects: {
      checking: "滴答滴答",
      confirmed: "咔嚓！",
      scheduled: "安排好了",
      done: "妥妥的",
    },
    catchphrases: ["安排好啦", "时间搞定", "妥妥的"],
  },
  [CapabilityType.AMAZING]: {
    parrotId: ParrotAgentType.AMAZING,
    name: "综合",
    nameAlt: "Amazing",
    description: "笔记 + 日程",
    icon: "🌟",
    color: "emerald",
    soundEffects: {
      searching: "咻...",
      insight: "哇哦~",
      done: "噢！综合完成",
      multiTask: "同时搜索中",
    },
    catchphrases: ["看看这个...", "综合来看", "发现规律了"],
  },
  [CapabilityType.AUTO]: {
    parrotId: ParrotAgentType.AUTO, // 由后端三层路由决定使用哪只鹦鹉
    name: "自动",
    nameAlt: "Auto",
    description: "智能路由",
    icon: "🤖",
    color: "emerald",
    soundEffects: {
      thinking: "嗯...让我想想",
      done: "✓",
    },
    catchphrases: ["看看这个...", "我帮你分析一下"],
  },
};

/**
 * 将 CapabilityType 转换为 ParrotAgentType
 */
export function capabilityToParrotAgent(capability: CapabilityType): ParrotAgentType {
  return CAPABILITIES[capability].parrotId;
}

/**
 * 将 ParrotAgentType 转换为 CapabilityType
 */
export function parrotAgentToCapability(agentType: ParrotAgentType): CapabilityType {
  switch (agentType) {
    case ParrotAgentType.MEMO:
      return CapabilityType.MEMO;
    case ParrotAgentType.SCHEDULE:
      return CapabilityType.SCHEDULE;
    default:
      return CapabilityType.AMAZING;
  }
}

/**
 * 获取能力显示名称
 */
export function getCapabilityName(capability: CapabilityType): string {
  return CAPABILITIES[capability].name;
}

/**
 * 获取能力图标
 */
export function getCapabilityIcon(capability: CapabilityType): string {
  return CAPABILITIES[capability].icon;
}

/**
 * 获取能力拟声词
 */
export function getCapabilitySound(
  capability: CapabilityType,
  context: "thinking" | "searching" | "found" | "done" | "checking" | "confirmed" | "scheduled" | "idea" | "insight",
): string {
  return CAPABILITIES[capability].soundEffects[context] || "";
}

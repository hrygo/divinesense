import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { CapabilityType, capabilityToParrotAgent, IntentRecognitionResult, parrotAgentToCapability } from "@/types/capability";
import { ParrotAgentType } from "@/types/parrot";

/**
 * 智能意图识别
 * 注意：路由逻辑已移至后端 (ChatRouter)，前端仅返回 AUTO 让后端决定
 * @param _input 用户输入 (unused - routing moved to backend)
 * @param currentCapability 当前能力（用于上下文）
 */
export function recognizeIntent(_input: string, currentCapability: CapabilityType = CapabilityType.AUTO): IntentRecognitionResult {
  // 路由逻辑已移至后端，前端始终返回 AUTO
  // 后端 ChatRouter 使用 规则+LLM 混合方式进行更准确的意图识别
  return {
    capability: currentCapability !== CapabilityType.AUTO ? currentCapability : CapabilityType.AUTO,
    confidence: 0.5,
    reasoning: "backend-routing",
  };
}

/**
 * 能力路由 Hook
 * 提供智能路由和能力管理功能
 *
 * 注意：意图识别已迁移至后端 ChatRouter，使用 规则+LLM 混合方式
 * 前端仅提供 UI 辅助函数（能力信息、类型转换）
 */
export function useCapabilityRouter() {
  const { t } = useTranslation();

  // 所有可用能力列表
  const availableCapabilities = useMemo(() => Object.values(CapabilityType).filter((c) => c !== CapabilityType.AUTO), []);

  /**
   * 根据用户输入路由到合适的能力
   * @deprecated 路由逻辑已移至后端，此函数仅返回 AUTO
   */
  const route = (_input: string, currentCapability?: CapabilityType): IntentRecognitionResult => {
    return recognizeIntent(_input, currentCapability);
  };

  /**
   * 获取能力显示信息
   */
  const getCapabilityInfo = (capability: CapabilityType) => {
    switch (capability) {
      case CapabilityType.MEMO:
        return {
          name: t("ai.capability.memo.name"),
          nameAlt: "Memo",
          description: t("ai.capability.memo.description"),
          icon: "🦜",
        };
      case CapabilityType.SCHEDULE:
        return {
          name: t("ai.capability.schedule.name"),
          nameAlt: "Schedule",
          description: t("ai.capability.schedule.description"),
          icon: "⏰",
        };
      case CapabilityType.GENERAL:
        return {
          name: t("ai.capability.general.name"),
          nameAlt: "General",
          description: t("ai.capability.general.description"),
          icon: "🌟",
        };
      case CapabilityType.AUTO:
      default:
        return {
          name: t("ai.capability.auto.name"),
          nameAlt: "Auto",
          description: t("ai.capability.auto.description"),
          icon: "🤖",
        };
    }
  };

  /**
   * 将能力转换为后台 Agent 类型
   */
  const toParrotAgent = (capability: CapabilityType): ParrotAgentType => {
    return capabilityToParrotAgent(capability);
  };

  /**
   * 将 Agent 类型转换为能力
   */
  const fromParrotAgent = (agentType: ParrotAgentType): CapabilityType => {
    return parrotAgentToCapability(agentType);
  };

  return {
    availableCapabilities,
    route,
    getCapabilityInfo,
    toParrotAgent,
    fromParrotAgent,
  };
}

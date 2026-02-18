/**
 * useParrotsList - 获取可用专家代理列表
 *
 * 调用 ListParrots API，过滤出可提及的专家代理
 *
 * @see Issue #259
 */
import { useQuery } from "@tanstack/react-query";
import { aiServiceClient } from "@/connect";
import { ParrotAgentType } from "@/types/parrot";
import { AgentType } from "@/types/proto/api/v1/ai_service_pb";

// 排除的代理类型（不可提及）
const EXCLUDED_AGENT_TYPES: AgentType[] = [
  AgentType.DEFAULT, // AUTO - 由 Orchestrator 自动路由
  // GEEK 和 EVOLUTION 不在 AgentType 枚举中，它们是模式标志
];

// AgentType 到 ParrotAgentType 的映射
const AGENT_TYPE_MAP: Record<number, ParrotAgentType> = {
  [AgentType.MEMO]: ParrotAgentType.MEMO,
  [AgentType.SCHEDULE]: ParrotAgentType.SCHEDULE,
  [AgentType.GENERAL]: ParrotAgentType.GENERAL,
  [AgentType.IDEATION]: ParrotAgentType.IDEATION,
};

/**
 * 专家代理信息（来自 API）
 */
export interface ParrotInfoFromAPI {
  agentType: ParrotAgentType;
  name: string;
  displayName?: string;
  description?: string;
}

/**
 * 获取可提及的专家代理列表
 *
 * @returns 过滤后的专家代理列表
 */
export function useParrotsList() {
  return useQuery({
    queryKey: ["parrots", "list"],
    queryFn: async (): Promise<ParrotInfoFromAPI[]> => {
      const response = await aiServiceClient.listParrots({});

      // 过滤出可提及的代理
      const mentionableParrots = response.parrots.filter((parrot) => {
        // 排除特定类型
        if (EXCLUDED_AGENT_TYPES.includes(parrot.agentType)) {
          return false;
        }

        // 只保留有 self_introduction 的代理
        if (!parrot.selfCognition?.selfIntroduction) {
          return false;
        }

        return true;
      });

      // 转换为前端格式
      return mentionableParrots.map((parrot) => ({
        agentType: AGENT_TYPE_MAP[parrot.agentType] || ParrotAgentType.AUTO,
        name: parrot.name,
        displayName: parrot.selfCognition?.title || parrot.name,
        description: parrot.selfCognition?.selfIntroduction || "",
      }));
    },
    staleTime: 5 * 60 * 1000, // 5 分钟缓存
    gcTime: 30 * 60 * 1000, // 30 分钟垃圾回收
  });
}

/**
 * 检查代理是否可提及
 */
export function isMentionable(agentType: ParrotAgentType): boolean {
  return (
    agentType === ParrotAgentType.MEMO ||
    agentType === ParrotAgentType.SCHEDULE ||
    agentType === ParrotAgentType.GENERAL ||
    agentType === ParrotAgentType.IDEATION
  );
}

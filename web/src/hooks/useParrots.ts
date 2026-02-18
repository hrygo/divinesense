import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PARROT_AGENTS, ParrotAgent, ParrotAgentType } from "@/types/parrot";

/**
 * Extended parrot agent with localized names
 * 带本地化名称的鹦鹉代理扩展
 */
export interface ParrotAgentI18n extends Omit<ParrotAgent, "displayName"> {
  displayName: string;
  displayNameAlt: string;
}

/**
 * Get localized parrot data (not a hook - can be used in useMemo)
 * 获取本地化的鹦鹉数据（非 hook - 可在 useMemo 中使用）
 */
export function getLocalizedParrot(
  agent: ParrotAgent,
  t: (key: string, options?: { returnObjects?: boolean }) => string | unknown,
): ParrotAgentI18n {
  const key = agent.id.toLowerCase();
  return {
    ...agent,
    displayName: t(`ai.parrot.agents.${key}.name`) as string,
    displayNameAlt: t(`ai.parrot.agents.${key}.nameAlt`) as string,
    description: t(`ai.parrot.agents.${key}.description`) as string,
    examplePrompts: (t(`ai.parrot.agents.${key}.examples`, { returnObjects: true }) || agent.examplePrompts || []) as string[],
  };
}

/**
 * Hook to get parrot agents with localized names, descriptions, and example prompts
 * 获取带本地化名称、描述和示例提示的鹦鹉代理
 */
export function useParrots(): ParrotAgentI18n[] {
  const { t } = useTranslation();

  return useMemo(() => {
    return Object.values(PARROT_AGENTS).map((agent) => getLocalizedParrot(agent, t));
  }, [t]);
}

/**
 * Hook to get a single parrot agent with localized data
 * 获取单个带本地化数据的鹦鹉代理
 */
export function useParrot(type: ParrotAgentType): ParrotAgentI18n {
  const { t } = useTranslation();

  return useMemo(() => {
    const agent = PARROT_AGENTS[type] || PARROT_AGENTS[ParrotAgentType.GENERAL];
    return getLocalizedParrot(agent, t);
  }, [t, type]);
}

/**
 * Hook to get available parrot agents with localized data
 * 获取可用的带本地化数据的鹦鹉代理
 */
export function useAvailableParrots(): ParrotAgentI18n[] {
  const parrots = useParrots();
  return useMemo(() => parrots.filter((p) => p.available), [parrots]);
}

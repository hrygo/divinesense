/**
 * Agent Mention 工具函数
 *
 * 用于解析消息中的 @ 提及和判断光标位置
 *
 * @see Issue #259
 */

import { ParrotAgentType } from "@/types/parrot";

// 可提及的专家代理映射（支持中文名和英文名）
// 当前支持：灰灰(MEMO)、时巧(SCHEDULE)、通才(GENERAL)、灵光(IDEATION)
const AGENT_MENTIONS: Record<string, ParrotAgentType> = {
  // 灰灰 - 笔记助手
  灰灰: ParrotAgentType.MEMO,
  memo: ParrotAgentType.MEMO,
  笔记: ParrotAgentType.MEMO,

  // 时巧 - 日程管理
  时巧: ParrotAgentType.SCHEDULE,
  schedule: ParrotAgentType.SCHEDULE,
  日程: ParrotAgentType.SCHEDULE,

  // 通才 - 通用助手
  通才: ParrotAgentType.GENERAL,
  general: ParrotAgentType.GENERAL,

  // 灵光 - 创意助手
  灵光: ParrotAgentType.IDEATION,
  ideation: ParrotAgentType.IDEATION,
};

/**
 * 解析消息中提及的代理
 *
 * @param message 原始消息
 * @returns { agents: 提及的代理名称数组, cleanMessage: 清理后的消息 }
 *
 * @example
 * parseMentionedAgents("@灰灰 @时巧 查笔记并安排会议")
 * // => { agents: ["memo", "schedule"], cleanMessage: "查笔记并安排会议" }
 */
export function parseMentionedAgents(message: string): {
  agents: string[];
  cleanMessage: string;
} {
  const agents: string[] = [];
  const mentionPattern = /@([^\s@]+)/g;
  let cleanMessage = message;

  // 收集所有提及
  const mentions = message.match(mentionPattern) || [];

  for (const mention of mentions) {
    const agentName = mention.slice(1).toLowerCase(); // 移除 @ 并转小写

    // 尝试匹配代理名
    for (const [name, type] of Object.entries(AGENT_MENTIONS)) {
      if (name.toLowerCase() === agentName || agentName.includes(name.toLowerCase())) {
        // 将代理类型转换为小写名称
        const agentKey = type.toLowerCase();
        if (!agents.includes(agentKey)) {
          agents.push(agentKey);
        }
        // 从消息中移除提及
        cleanMessage = cleanMessage.replace(mention, "").trim();
        break;
      }
    }
  }

  return {
    agents,
    cleanMessage: cleanMessage.replace(/\s+/g, " ").trim(), // 清理多余空格
  };
}

/**
 * 判断光标是否在可插入 @ 的位置
 *
 * 规则：仅在消息头部或尾部可以插入 @
 * - 头部：光标位置为 0 或前面只有空白字符
 * - 尾部：光标位置等于文本长度或后面只有空白字符
 *
 * @param text 完整文本
 * @param cursorPosition 光标位置
 * @returns 是否可插入 @
 */
export function canInsertMention(text: string, cursorPosition: number): boolean {
  // 空文本，可插入
  if (!text) return true;

  // 检查头部位置
  const beforeCursor = text.slice(0, cursorPosition);
  const isAtHead = beforeCursor.trim() === "" || beforeCursor.endsWith("\n");

  // 检查尾部位置
  const afterCursor = text.slice(cursorPosition);
  const isAtTail = afterCursor.trim() === "" || afterCursor.startsWith("\n");

  return isAtHead || isAtTail;
}

/**
 * 检测是否应该触发 Agent 选择弹窗
 *
 * 条件：
 * 1. 刚输入了 @ 字符
 * 2. 光标在可插入位置（@ 前面是空白或换行）
 *
 * @param text 完整文本
 * @param cursorPosition 光标位置
 * @returns 是否应该触发弹窗，以及过滤文本（如果有）
 */
export function shouldTriggerMentionPopover(text: string, cursorPosition: number): { shouldTrigger: boolean; filter: string } {
  // 检查光标前一个字符是否是 @
  if (cursorPosition < 1 || text[cursorPosition - 1] !== "@") {
    return { shouldTrigger: false, filter: "" };
  }

  // 检查 @ 前面是否是可插入位置
  // 即 @ 位于头部或尾部
  const atPosition = cursorPosition - 1;
  const beforeAt = text.slice(0, atPosition);

  // @ 前面必须是空白或换行（头部位置），否则不触发
  if (beforeAt.trim() !== "" && !beforeAt.endsWith("\n")) {
    return { shouldTrigger: false, filter: "" };
  }

  // @ 后面是否有过滤文本
  const afterAt = text.slice(cursorPosition);
  const filterMatch = afterAt.match(/^([^\s@]*)/);
  const filter = filterMatch ? filterMatch[1] : "";

  return { shouldTrigger: true, filter };
}

/**
 * 获取 @ 符号的起始位置
 *
 * @param text 完整文本
 * @param cursorPosition 光标位置
 * @returns @ 符号的位置，如果不存在则返回 -1
 */
export function getMentionStartPosition(text: string, cursorPosition: number): number {
  if (cursorPosition < 1 || text[cursorPosition - 1] !== "@") {
    return -1;
  }

  return cursorPosition - 1;
}

/**
 * 在指定位置插入代理提及
 *
 * @param text 原始文本
 * @param position 插入位置
 * @param agentName 代理名称
 * @returns 插入后的文本和新的光标位置
 */
export function insertAgentMention(text: string, position: number, agentName: string): { newText: string; newCursorPos: number } {
  const mention = `@${agentName} `;
  const newText = text.slice(0, position) + mention + text.slice(position);
  const newCursorPos = position + mention.length;

  return { newText, newCursorPos };
}

/**
 * 格式化代理显示名称
 *
 * @param name 代理英文名（memo, schedule, general, ideation 等）
 * @returns 格式化后的名称，如 "@灰灰"
 */
export function formatAgentMention(name: string): string {
  const names: Record<string, string> = {
    memo: "灰灰",
    schedule: "时巧",
    general: "通才",
    ideation: "灵光",
  };

  return `@${names[name] || name}`;
}

/**
 * AgentMentionPopover - @ 符号触发专家 Agent 选择弹窗
 *
 * 设计理念：「鹦鹉栖息」隐喻
 * - 用户输入 @ 时，弹窗如同群鹦鹉栖息枝头等待召唤
 * - 每只鹦鹉都有独特的羽色（对应其主题色）
 * - 选择时的微妙动效如同鹦鹉振翅
 *
 * @see Issue #259
 */
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { ScrollArea } from "@/components/ui/scroll-area";
import { type ParrotInfoFromAPI, useParrotsList } from "@/hooks/useParrotsList";
import { cn } from "@/lib/utils";

// Agent 主题色映射
const AGENT_COLORS: Record<string, { primary: string; bg: string; ring: string; emoji: string }> = {
  memo: {
    primary: "slate",
    bg: "from-slate-100 to-slate-200 dark:from-slate-700 dark:to-slate-800",
    ring: "ring-slate-400 bg-slate-100 dark:bg-slate-800/50",
    emoji: "🪶",
  },
  schedule: {
    primary: "cyan",
    bg: "from-cyan-100 to-cyan-200 dark:from-cyan-800 dark:to-cyan-900",
    ring: "ring-cyan-400 bg-cyan-50 dark:bg-cyan-900/30",
    emoji: "⏰",
  },
  general: {
    primary: "amber",
    bg: "from-amber-100 to-amber-200 dark:from-amber-800 dark:to-amber-900",
    ring: "ring-amber-400 bg-amber-50 dark:bg-amber-900/30",
    emoji: "🤖",
  },
  insight: {
    primary: "violet",
    bg: "from-violet-100 to-violet-200 dark:from-violet-800 dark:to-violet-900",
    ring: "ring-violet-400 bg-violet-50 dark:bg-violet-900/30",
    emoji: "💡",
  },
};

// 默认颜色
const DEFAULT_COLOR = {
  primary: "zinc",
  bg: "from-zinc-100 to-zinc-200 dark:from-zinc-700 dark:to-zinc-800",
  ring: "ring-zinc-400 bg-zinc-100 dark:bg-zinc-800/50",
  emoji: "🦜",
};

// 可提及的代理名称列表（从 API 获取后会过滤）
const MENTIONABLE_NAMES = ["memo", "schedule", "general", "insight"];

interface AgentMentionPopoverProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelect: (agent: ParrotInfoFromAPI) => void;
  anchorElement: HTMLElement | null;
  filter?: string;
}

/**
 * AgentMentionPopover 组件
 *
 * 特性：
 * - 键盘导航（↑↓ + Enter + Esc）
 * - 鼠标点击选择
 * - 主题色区分
 * - 动画效果
 * - Portal 渲染（避免 z-index 问题）
 * - 从 API 动态获取代理列表
 */
export function AgentMentionPopover({ open, onOpenChange, onSelect, anchorElement, filter = "" }: AgentMentionPopoverProps) {
  const { t } = useTranslation();
  const [selectedIndex, setSelectedIndex] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });

  // 从 API 获取代理列表
  const { data: apiAgents = [], isLoading } = useParrotsList();

  // 过滤出可提及的代理
  const mentionableAgents = useMemo(() => {
    return apiAgents.filter((agent) => MENTIONABLE_NAMES.includes(agent.name));
  }, [apiAgents]);

  // 根据 filter 过滤代理
  const filteredAgents = useMemo(() => {
    if (!filter) return mentionableAgents;

    const lowerFilter = filter.toLowerCase();
    return mentionableAgents.filter(
      (agent) => agent.name.toLowerCase().includes(lowerFilter) || (agent.displayName?.toLowerCase().includes(lowerFilter) ?? false),
    );
  }, [mentionableAgents, filter]);

  // 计算弹窗位置
  useEffect(() => {
    if (!open || !anchorElement) return;

    const updatePosition = () => {
      const rect = anchorElement.getBoundingClientRect();
      const popoverHeight = 280;
      const popoverWidth = 288;

      let top = rect.top - popoverHeight - 8;
      let left = rect.left;

      if (top < 0) {
        top = rect.bottom + 8;
      }

      if (left + popoverWidth > window.innerWidth) {
        left = window.innerWidth - popoverWidth - 16;
      }

      if (left < 16) {
        left = 16;
      }

      setPosition({ top, left });
    };

    updatePosition();
    window.addEventListener("scroll", updatePosition, true);
    window.addEventListener("resize", updatePosition);

    return () => {
      window.removeEventListener("scroll", updatePosition, true);
      window.removeEventListener("resize", updatePosition);
    };
  }, [open, anchorElement]);

  // 重置选中索引
  useEffect(() => {
    setSelectedIndex(0);
  }, [filteredAgents.length]);

  // 键盘事件处理
  useEffect(() => {
    if (!open) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          e.stopPropagation();
          setSelectedIndex((prev) => (prev < filteredAgents.length - 1 ? prev + 1 : 0));
          break;
        case "ArrowUp":
          e.preventDefault();
          e.stopPropagation();
          setSelectedIndex((prev) => (prev > 0 ? prev - 1 : filteredAgents.length - 1));
          break;
        case "Enter":
          e.preventDefault();
          e.stopPropagation();
          if (filteredAgents[selectedIndex]) {
            onSelect(filteredAgents[selectedIndex]);
            onOpenChange(false);
          }
          break;
        case "Escape":
          e.preventDefault();
          e.stopPropagation();
          onOpenChange(false);
          break;
      }
    };

    document.addEventListener("keydown", handleKeyDown, true);
    return () => document.removeEventListener("keydown", handleKeyDown, true);
  }, [open, filteredAgents, selectedIndex, onSelect, onOpenChange]);

  // 滚动到选中项
  useEffect(() => {
    if (listRef.current && filteredAgents.length > 0) {
      const selectedElement = listRef.current.children[selectedIndex] as HTMLElement;
      if (selectedElement) {
        selectedElement.scrollIntoView({ block: "nearest", behavior: "smooth" });
      }
    }
  }, [selectedIndex, filteredAgents.length]);

  // 点击外部关闭
  useEffect(() => {
    if (!open) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        onOpenChange(false);
      }
    };

    const timer = setTimeout(() => {
      document.addEventListener("mousedown", handleClickOutside);
    }, 100);

    return () => {
      clearTimeout(timer);
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [open, onOpenChange]);

  // 点击选择
  const handleSelect = useCallback(
    (agent: ParrotInfoFromAPI) => {
      onSelect(agent);
      onOpenChange(false);
    },
    [onSelect, onOpenChange],
  );

  // 获取代理颜色配置
  const getAgentColor = (name: string) => {
    return AGENT_COLORS[name] || DEFAULT_COLOR;
  };

  // 不渲染
  if (!open || !anchorElement) return null;

  const popoverContent = (
    <div
      ref={popoverRef}
      className={cn(
        "fixed z-50 w-72 rounded-lg border border-border bg-popover shadow-lg",
        "animate-in fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95",
        "data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2",
      )}
      style={{
        top: position.top,
        left: position.left,
      }}
      data-state={open ? "open" : "closed"}
      data-side="top"
    >
      {/* 加载中 */}
      {isLoading ? (
        <div className="p-4 text-center">
          <div className="animate-pulse text-sm text-muted-foreground">{t("ai.mention.loading", { defaultValue: "加载中..." })}</div>
        </div>
      ) : filteredAgents.length === 0 ? (
        /* 无匹配结果 */
        <div className="p-4 text-center">
          <p className="text-sm text-muted-foreground">{t("ai.mention.no_match", { defaultValue: "未找到匹配的专家" })}</p>
        </div>
      ) : (
        <>
          {/* 标题区 */}
          <div className="px-3 py-2 border-b border-border bg-muted/30 rounded-t-lg">
            <span className="text-xs font-medium text-muted-foreground">
              {t("ai.mention.select_agent", { defaultValue: "🦜 选择专家代理" })}
            </span>
          </div>

          {/* Agent 列表 */}
          <ScrollArea className="max-h-64">
            <div ref={listRef} className="p-1">
              {filteredAgents.map((agent, index) => (
                <AgentItem
                  key={agent.name}
                  agent={agent}
                  color={getAgentColor(agent.name)}
                  isSelected={index === selectedIndex}
                  onClick={() => handleSelect(agent)}
                  onMouseEnter={() => setSelectedIndex(index)}
                />
              ))}
            </div>
          </ScrollArea>

          {/* 操作提示 */}
          <div className="px-3 py-1.5 border-t border-border bg-muted/20 rounded-b-lg">
            <span className="text-[10px] text-muted-foreground">
              <kbd className="px-1 py-0.5 bg-background rounded text-[9px]">↑↓</kbd> {t("ai.mention.navigate", { defaultValue: "选择" })} ·{" "}
              <kbd className="px-1 py-0.5 bg-background rounded text-[9px]">Enter</kbd> {t("ai.mention.confirm", { defaultValue: "确认" })}{" "}
              · <kbd className="px-1 py-0.5 bg-background rounded text-[9px]">Esc</kbd> {t("ai.mention.close", { defaultValue: "关闭" })}
            </span>
          </div>
        </>
      )}
    </div>
  );

  return createPortal(popoverContent, document.body);
}

/**
 * Agent 选项组件
 */
interface AgentItemProps {
  agent: ParrotInfoFromAPI;
  color: { primary: string; bg: string; ring: string; emoji: string };
  isSelected: boolean;
  onClick: () => void;
  onMouseEnter: () => void;
}

function AgentItem({ agent, color, isSelected, onClick, onMouseEnter }: AgentItemProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={onMouseEnter}
      className={cn(
        "w-full flex items-center gap-3 px-3 py-2.5 rounded-md transition-all duration-150",
        "text-left cursor-pointer",
        isSelected ? ["scale-[1.02]", "ring-2 ring-offset-1", color.ring] : "hover:bg-accent/50",
      )}
    >
      {/* 头像 */}
      <div className={cn("w-9 h-9 rounded-full flex items-center justify-center shrink-0", "bg-gradient-to-br", color.bg)}>
        <span className="text-lg">{color.emoji}</span>
      </div>

      {/* 信息 */}
      <div className="flex-1 min-w-0">
        <div className={cn("font-medium text-sm truncate", `text-${color.primary}-700 dark:text-${color.primary}-200`)}>
          @{agent.displayName || agent.name}
        </div>
        <div className="text-xs text-muted-foreground truncate">{agent.description}</div>
      </div>

      {/* 选中指示器 */}
      {isSelected && <div className={cn("w-1.5 h-1.5 rounded-full shrink-0", `bg-${color.primary}-500`)} />}
    </button>
  );
}

export default AgentMentionPopover;

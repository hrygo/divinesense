/**
 * SlashMenu - 斜杠命令建议菜单
 *
 * 显示斜杠命令列表，支持键盘导航和鼠标选择
 */
import { memo, useCallback, useEffect, useRef } from "react";
import { type SuggestionItem } from "@/components/MemoEditor/core/editor-types";
import { cn } from "@/lib/utils";

/**
 * 单个建议项组件
 */
interface SuggestionItemProps {
  item: SuggestionItem;
  isSelected: boolean;
  onSelect: () => void;
}

const SuggestionMenuItem = memo<SuggestionItemProps>(({ item, isSelected, onSelect }: SuggestionItemProps) => {
  return (
    <div
      role="option"
      aria-selected={isSelected}
      className={cn(
        "flex items-center gap-2 px-3 py-2 rounded-md cursor-pointer transition-colors",
        "hover:bg-accent/5",
        isSelected && "bg-accent/10",
        isSelected && "aria-selected",
      )}
      onClick={onSelect}
    >
      {/* 图标 */}
      {item.icon && <div className="flex-shrink-0 text-muted-foreground">{item.icon}</div>}

      {/* 标签和描述 */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium text-sm">{item.label}</span>
          {item.shortcut && (
            <span className={cn("text-xs text-muted-foreground/60", "rounded px-1.5 py-0.5", "border border-border/30")}>
              {item.shortcut}
            </span>
          )}
        </div>
        {item.description && <div className="text-xs text-muted-foreground/70 mt-0.5">{item.description}</div>}
      </div>
    </div>
  );
});
SuggestionMenuItem.displayName = "SuggestionMenuItem";

/**
 * 建议菜单分组标题
 */
interface GroupHeaderProps {
  title: string;
}

const GroupHeader = memo<GroupHeaderProps>(({ title }: GroupHeaderProps) => {
  return <div className="px-3 py-2 text-xs font-medium text-muted-foreground/60 uppercase tracking-wider">{title}</div>;
});
GroupHeader.displayName = "GroupHeader";

/**
 * 命令分组（用于 COMMAND_GROUPS）
 */
interface CommandGroup {
  title: string;
  key: string;
  commands: SuggestionItem[];
}

// 将命令按分组
function groupCommands(commands: SuggestionItem[]): CommandGroup[] {
  return [
    {
      title: "文本格式",
      key: "formatting",
      commands: commands.filter((cmd) => cmd.id === "bold" || cmd.id === "italic" || cmd.id === "strikethrough" || cmd.id === "code"),
    },
    {
      title: "标题",
      key: "headings",
      commands: commands.filter((cmd) => cmd.id === "heading_1" || cmd.id === "heading_2" || cmd.id === "heading_3"),
    },
    {
      title: "列表",
      key: "lists",
      commands: commands.filter(
        (cmd) => cmd.id === "bullet_list" || cmd.id === "numbered_list" || cmd.id === "todo_list" || cmd.id === "todo_checked",
      ),
    },
    {
      title: "其他",
      key: "other",
      commands: commands.filter(
        (cmd) => cmd.id === "quote" || cmd.id === "link" || cmd.id === "divider" || cmd.id === "table" || cmd.id === "clear",
      ),
    },
  ];
}

/**
 * SlashMenu Props
 */
export interface SlashMenuProps {
  isOpen: boolean;
  items: SuggestionItem[];
  selectedIndex: number;
  position: { top: number; left: number } | null;
  onSelect: (item: SuggestionItem) => void;
  onClose: () => void;
}

/**
 * Slash Menu Component
 */
const SlashMenuComponent = ({ isOpen, items, selectedIndex, position, onSelect, onClose }: SlashMenuProps) => {
  const menuRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLUListElement>(null);

  /**
   * 处理键盘导航
   */
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (!isOpen) return;

      switch (e.key) {
        case "Escape":
          e.preventDefault();
          onClose();
          break;
        case "ArrowDown":
          e.preventDefault();
          onSelectItem((selectedIndex + 1) % items.length);
          break;
        case "ArrowUp":
          e.preventDefault();
          onSelectItem((selectedIndex - 1 + items.length) % items.length);
          break;
        case "Enter":
        case "Tab":
          e.preventDefault();
          if (items[selectedIndex]) {
            onSelect(items[selectedIndex]);
          }
          break;
        case "Home":
          e.preventDefault();
          onSelectItem(0);
          break;
        case "End":
          e.preventDefault();
          onSelectItem(items.length - 1);
          break;
      }
    },
    [isOpen, items, selectedIndex, onClose],
  );

  /**
   * 选择指定索引的项
   */
  const onSelectItem = useCallback(
    (index: number) => {
      if (index >= 0 && index < items.length) {
        const item = items[index];
        if (item) {
          onSelect(item);
        }
      }
    },
    [items, onSelect],
  );

  /**
   * 滚动选中项到可见区域
   */
  const scrollSelectedIntoView = useCallback(() => {
    if (listRef.current) {
      const selectedElement = listRef.current.children[selectedIndex];
      if (selectedElement instanceof HTMLElement) {
        selectedElement.scrollIntoView({
          block: "nearest",
          inline: "nearest",
        });
      }
    }
  }, [selectedIndex, items]);

  // 滚动选中项到可见区域
  useEffect(() => {
    if (isOpen) {
      // 延迟执行，确保 DOM 已更新
      setTimeout(scrollSelectedIntoView, 50);
    }
  }, [isOpen, selectedIndex, scrollSelectedIntoView]);

  // 如果菜单关闭或没有项目，不渲染
  if (!isOpen || items.length === 0) {
    return null;
  }

  // 计算菜单位置
  const menuStyle: React.CSSProperties = {
    position: "absolute",
    ...(position
      ? {
          top: `${position.top}px`,
          left: `${position.left}px`,
        }
      : {
          top: "100%",
          left: "50%",
          transform: "translateX(-50%)",
        }),
    zIndex: 50,
    maxHeight: "300px",
    width: "280px",
  };

  const groupedCommands = groupCommands(items);

  return (
    <div
      ref={menuRef}
      className="bg-popover/95 backdrop-blur-md border border-border/20 rounded-lg shadow-lg overflow-hidden"
      style={menuStyle}
      onKeyDown={handleKeyDown}
    >
      {/* 标题栏 */}
      <div className="border-b border-border/10 px-3 py-2 bg-muted/20">
        <span className="text-sm font-medium text-muted-foreground/80">斜杠命令</span>
        <span className="text-xs text-muted-foreground/50 ml-2">{items.length} 项</span>
      </div>

      {/* 命令列表 */}
      <ul ref={listRef} role="listbox" className="max-h-[252px] overflow-y-auto py-1">
        {groupedCommands.map((group) => (
          <li key={group.key} className="mb-2 last:mb-0">
            {group.title && <GroupHeader title={group.title} />}
            <div role="group" aria-label={group.title}>
              {group.commands.map((item) => {
                // 计算全局索引
                const globalIndex = items.indexOf(item);
                return (
                  <SuggestionMenuItem
                    key={item.id}
                    item={item}
                    isSelected={globalIndex === selectedIndex}
                    onSelect={() => onSelect(item)}
                  />
                );
              })}
            </div>
          </li>
        ))}
      </ul>

      {/* 底部提示 */}
      <div className="border-t border-border/10 px-3 py-2 bg-muted/20">
        <div className="flex items-center gap-3 text-xs text-muted-foreground/60">
          <span>↑↓</span>
          <span>导航</span>
          <span>Enter</span>
          <span>确认</span>
          <span>Esc</span>
          <span>关闭</span>
        </div>
      </div>
    </div>
  );
};

const SlashMenu = memo(SlashMenuComponent);
SlashMenu.displayName = "SlashMenu";

export default SlashMenu;

/**
 * Suggestion Menu Types - 建议菜单类型定义
 *
 * 定义建议菜单的状态和操作接口
 */

import type { SuggestionItem, TriggerType } from "./editor-types";

/**
 * 建议菜单 Ref Actions
 */
export interface SuggestionMenuRefActions {
  open(trigger: TriggerType, position: { top: number; left: number }): void;
  close(): void;
  selectNext(): void;
  selectPrevious(): void;
  confirmSelection(): void;
  updateItems(items: SuggestionItem[]): void;
  updateQuery(query: string): void;
  setSelectedIndex(index: number): void;
}

/**
 * 建议菜单 Props
 */
export interface SuggestionMenuProps {
  isOpen: boolean;
  trigger: TriggerType | null;
  query: string;
  items: SuggestionItem[];
  selectedIndex: number;
  position: { top: number; left: number } | null;
  onSelect: (item: SuggestionItem) => void;
  onClose: () => void;
  onItemHover?: (index: number) => void;
}

/**
 * 建议菜单样式配置
 */
export interface SuggestionMenuStyles {
  container: string;
  item: string;
  itemSelected: string;
  itemIcon: string;
  itemLabel: string;
  itemDescription: string;
  itemShortcut: string;
  separator: string;
}

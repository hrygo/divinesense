/**
 * EditorWithPlugins - 集成插件系统的完整编辑器
 *
 * 整合了：
 * - EnhancedEditor 核心编辑器
 * - Slash Commands (/)
 * - Tag Suggestions (#)
 * - List Autocomplete
 * - 自动高度调整
 * - IME 支持
 */

import { forwardRef, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { matchPath } from "react-router-dom";
import OverflowTip from "@/components/kit/OverflowTip";
import { useTagCounts } from "@/hooks/useUserQueries";
import { Routes } from "@/router";
import { EnhancedEditor } from "../core";
import type { CursorPosition, EnhancedEditorRefActions, SuggestionItem } from "../core/editor-types";
import { TriggerType } from "../core/editor-types";
import { createListAutocompletePlugin, createSlashCommandPlugin, slashCommandTriggerConfig, tagSuggestionTriggerConfig } from "../plugins";
import { useEditorContext } from "../state/context";
import type { EditorContentProps } from "../types";

interface EditorWithPluginsProps extends EditorContentProps {
  isFocusMode?: boolean;
}

interface SuggestionState {
  isOpen: boolean;
  trigger: TriggerType | null;
  query: string;
  items: SuggestionItem[];
  selectedIndex: number;
  cursorPosition: CursorPosition | null;
}

/**
 * EditorWithPlugins Component
 */
export const EditorWithPlugins = forwardRef<EnhancedEditorRefActions, EditorWithPluginsProps>(({ placeholder }, ref) => {
  const { t } = useTranslation();
  const { state, actions, dispatch } = useEditorContext();
  // Internal ref for accessing editor methods within this component
  const editorRef = useRef<EnhancedEditorRefActions>(null);

  // When editor ref updates, sync internal ref
  // In practice, React's useImperativeHandle returns a mutable ref
  // TypeScript sees RefObject.current as readonly, but useRef returns MutableRefObject
  const handleEditorRef = useCallback(
    (instance: EnhancedEditorRefActions | null) => {
      // @ts-expect-error: RefObject.current is readonly in types, but MutableRefObject.current is writable
      editorRef.current = instance;
      // Forward to parent ref
      if (ref) {
        if (typeof ref === "function") {
          ref(instance);
        } else {
          try {
            (ref as unknown as { current?: EnhancedEditorRefActions | null }).current = instance;
          } catch {
            // React may have frozen the ref object, ignore
          }
        }
      }
    },
    [ref],
  );

  // On explore page, show all users' tags; otherwise show current user's tags
  const isExplorePage = Boolean(matchPath(Routes.EXPLORE, window.location.pathname));
  const { data: tagCount = {} } = useTagCounts(!isExplorePage);

  // Sort tags by usage count
  const sortedTags = useMemo(() => {
    return Object.entries(tagCount)
      .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
      .map(([tag]) => tag);
  }, [tagCount]);

  // Suggestion state
  const [suggestions, setSuggestions] = useState<SuggestionState>({
    isOpen: false,
    trigger: null,
    query: "",
    items: [],
    selectedIndex: 0,
    cursorPosition: null,
  });

  // Create plugins with i18n
  // Use ref to store plugins to avoid recreation on every render
  const pluginsRef = useRef<[ReturnType<typeof createSlashCommandPlugin>, ReturnType<typeof createListAutocompletePlugin>] | null>(null);
  if (!pluginsRef.current) {
    pluginsRef.current = [createSlashCommandPlugin(t), createListAutocompletePlugin(t)];
  }
  const plugins = pluginsRef.current;

  // Dispatch function for plugins
  const pluginDispatch = useCallback((action: { type: string; payload?: unknown }) => {
    if (action.type === "SET_SUGGESTIONS" && action.payload) {
      setSuggestions((prev) => ({ ...prev, ...(action.payload as Partial<SuggestionState>) }));
    }
  }, []);

  // Get current word and trigger info
  const getCurrentWord = useCallback((): [word: string, startIndex: number, triggerChar: TriggerType | null] => {
    const editor = editorRef.current;
    if (!editor) return ["", 0, null];

    const content = editor.getContent?.() || "";
    const cursorPos = editor.getSelection?.() || { start: 0, end: 0 };
    const beforeCursor = content.slice(0, cursorPos.start);

    // Check for slash trigger
    const slashIndex = beforeCursor.lastIndexOf("/");
    const maxSlashItems = slashCommandTriggerConfig.maxItems ?? 20;
    if (slashIndex !== -1 && cursorPos.start - slashIndex <= maxSlashItems) {
      const word = beforeCursor.slice(slashIndex);
      return [word, slashIndex, TriggerType.SLASH];
    }

    // Check for hash trigger
    const hashIndex = beforeCursor.lastIndexOf("#");
    const maxHashItems = tagSuggestionTriggerConfig.maxItems ?? 20;
    if (hashIndex !== -1 && cursorPos.start - hashIndex <= maxHashItems) {
      const word = beforeCursor.slice(hashIndex);
      return [word, hashIndex, TriggerType.HASH];
    }

    return ["", 0, null];
  }, []);

  // Use refs to avoid effect re-running when these values change
  const suggestionsRef = useRef(suggestions);
  suggestionsRef.current = suggestions;
  const sortedTagsRef = useRef(sortedTags);
  sortedTagsRef.current = sortedTags;
  const tagCountRef = useRef(tagCount);
  tagCountRef.current = tagCount;

  // Update suggestions based on current state
  useEffect(() => {
    const [word, _startIndex, triggerChar] = getCurrentWord();

    if (!triggerChar || !word.startsWith(triggerChar)) {
      setSuggestions((prev) => ({ ...prev, isOpen: false, trigger: null, cursorPosition: null }));
      return;
    }

    const query = word.slice(triggerChar.length).toLowerCase();
    const cursorPosition = editorRef.current?.getCursorPosition?.() || null;

    if (triggerChar === TriggerType.SLASH) {
      // Get slash command suggestions
      const plugin = plugins[0]; // slash command plugin
      if (plugin?.getSuggestions) {
        const result = plugin.getSuggestions(
          {
            content: state.content,
            editor: editorRef.current,
            cursor: cursorPosition,
            selection: editorRef.current?.getSelection?.() || null,
            suggestions: {
              isOpen: suggestionsRef.current.isOpen,
              trigger: suggestionsRef.current.trigger,
              query: suggestionsRef.current.query,
              items: suggestionsRef.current.items,
              selectedIndex: suggestionsRef.current.selectedIndex,
              position: suggestionsRef.current.cursorPosition
                ? { top: suggestionsRef.current.cursorPosition.top, left: suggestionsRef.current.cursorPosition.left }
                : null,
            },
            dispatch: pluginDispatch,
          },
          query,
        );

        // Handle both sync and async results
        const handleResult = (items: SuggestionItem[]) => {
          setSuggestions((prev) => ({
            ...prev,
            isOpen: items.length > 0,
            trigger: triggerChar,
            query,
            items,
            selectedIndex: 0,
            cursorPosition,
          }));
        };

        if (result instanceof Promise) {
          result.then(handleResult);
        } else {
          handleResult(result);
        }
      }
    } else if (triggerChar === TriggerType.HASH) {
      // Filter tags - use ref to avoid dependency issues
      const currentSortedTags = sortedTagsRef.current;
      const currentTagCount = tagCountRef.current;
      const filteredTags = currentSortedTags.filter((tag) => tag.toLowerCase().includes(query));
      const items: SuggestionItem[] = filteredTags.slice(0, 10).map((tag) => ({
        id: `tag-${tag}`,
        label: tag,
        description: `${currentTagCount[tag] || 0} ${t("editor.chars")}`,
        icon: undefined,
        keywords: undefined,
        action: (editor: EnhancedEditorRefActions): void => {
          editor.replaceTextAtCursor?.("#", "", { selectAfter: true });
          editor.insertText?.(`#${tag} `);
        },
        shortcut: undefined,
      }));

      setSuggestions((prev) => ({
        ...prev,
        isOpen: items.length > 0,
        trigger: triggerChar,
        query,
        items,
        selectedIndex: 0,
        cursorPosition,
      }));
    } else {
      // Check for list suggestions (empty line)
      const cursorContext = editorRef.current?.getContextAtCursor?.();
      if (cursorContext) {
        const { line } = cursorContext;
        if (line.trim() === "" || line.trim() === "-") {
          // Show list suggestions
          const plugin = plugins[1]; // list autocomplete plugin
          if (plugin?.getSuggestions) {
            const result = plugin.getSuggestions(
              {
                content: state.content,
                editor: editorRef.current,
                cursor: cursorPosition,
                selection: editorRef.current?.getSelection?.() || null,
                suggestions: {
                  isOpen: suggestionsRef.current.isOpen,
                  trigger: suggestionsRef.current.trigger,
                  query: suggestionsRef.current.query,
                  items: suggestionsRef.current.items,
                  selectedIndex: suggestionsRef.current.selectedIndex,
                  position: suggestionsRef.current.cursorPosition
                    ? { top: suggestionsRef.current.cursorPosition.top, left: suggestionsRef.current.cursorPosition.left }
                    : null,
                },
                dispatch: pluginDispatch,
              },
              "",
            );

            const handleResult = (items: SuggestionItem[]) => {
              setSuggestions((prev) => ({
                ...prev,
                isOpen: items.length > 0,
                trigger: TriggerType.CUSTOM,
                query: "",
                items,
                selectedIndex: 0,
                cursorPosition,
              }));
            };

            if (result instanceof Promise) {
              result.then(handleResult);
            } else {
              handleResult(result);
            }
          }
          return;
        }
      }

      setSuggestions((prev) => ({ ...prev, isOpen: false, trigger: null, cursorPosition: null }));
    }
  }, [state.content, getCurrentWord, pluginDispatch, plugins, t]);

  // Handle content change
  const handleContentChange = useCallback(
    (content: string) => {
      dispatch(actions.updateContent(content));
    },
    [dispatch, actions],
  );

  // Handle key down
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      // Let suggestion menu handle navigation first
      if (suggestions.isOpen) {
        if (e.key === "Escape") {
          e.preventDefault();
          setSuggestions((prev) => ({ ...prev, isOpen: false, trigger: null, cursorPosition: null }));
          return;
        }
        if (e.key === "ArrowDown") {
          e.preventDefault();
          setSuggestions((prev) => ({
            ...prev,
            selectedIndex: (prev.selectedIndex + 1) % prev.items.length,
          }));
          return;
        }
        if (e.key === "ArrowUp") {
          e.preventDefault();
          setSuggestions((prev) => ({
            ...prev,
            selectedIndex: (prev.selectedIndex - 1 + prev.items.length) % prev.items.length,
          }));
          return;
        }
        if (e.key === "Enter" || e.key === "Tab") {
          e.preventDefault();
          const selectedItem = suggestions.items[suggestions.selectedIndex];
          if (selectedItem?.action && editorRef.current) {
            selectedItem.action(editorRef.current);
          }
          setSuggestions((prev) => ({ ...prev, isOpen: false, trigger: null, cursorPosition: null }));
          return;
        }
      }

      // Let plugins handle event
      for (const plugin of plugins) {
        if (plugin.onKeyDown) {
          const handled = plugin.onKeyDown(
            {
              content: state.content,
              editor: editorRef.current,
              cursor: suggestions.cursorPosition,
              selection: editorRef.current?.getSelection?.() || null,
              suggestions: {
                isOpen: suggestions.isOpen,
                trigger: suggestions.trigger,
                query: suggestions.query,
                items: suggestions.items,
                selectedIndex: suggestions.selectedIndex,
                position: suggestions.cursorPosition
                  ? { top: suggestions.cursorPosition.top, left: suggestions.cursorPosition.left }
                  : null,
              },
              dispatch: pluginDispatch,
            },
            e.nativeEvent,
          );
          if (handled) return;
        }
      }
    },
    [state.content, plugins, suggestions, pluginDispatch],
  );

  // Handle composition events
  const handleCompositionStart = useCallback(() => {
    dispatch(actions.setComposing(true));
  }, [dispatch, actions]);

  const handleCompositionEnd = useCallback(() => {
    dispatch(actions.setComposing(false));
  }, [dispatch, actions]);

  // Handle paste
  const handlePaste = useCallback(
    (e: React.ClipboardEvent) => {
      // Check if any plugin wants to handle paste
      for (const plugin of plugins) {
        if (plugin.onPaste) {
          const handled = plugin.onPaste(
            {
              content: state.content,
              editor: editorRef.current,
              cursor: suggestions.cursorPosition,
              selection: editorRef.current?.getSelection?.() || null,
              suggestions: {
                isOpen: suggestions.isOpen,
                trigger: suggestions.trigger,
                query: suggestions.query,
                items: suggestions.items,
                selectedIndex: suggestions.selectedIndex,
                position: suggestions.cursorPosition
                  ? { top: suggestions.cursorPosition.top, left: suggestions.cursorPosition.left }
                  : null,
              },
              dispatch: pluginDispatch,
            },
            e.nativeEvent as ClipboardEvent,
          );
          if (handled) {
            e.preventDefault();
            return true;
          }
        }
      }
      return false;
    },
    [state.content, plugins, suggestions, pluginDispatch],
  );

  // Close suggestions on click outside
  useEffect(() => {
    if (!suggestions.isOpen) return;

    const handleClick = (e: MouseEvent) => {
      const target = e.target as Node;
      const menu = document.getElementById("suggestion-menu");
      if (menu !== null && !menu.contains(target)) {
        setSuggestions((prev) => ({ ...prev, isOpen: false, trigger: null, cursorPosition: null }));
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [suggestions.isOpen]);

  return (
    <div className="relative w-full h-full">
      <EnhancedEditor
        // @ts-ignore: Ref callback type is valid but TypeScript expects RefObject
        ref={handleEditorRef}
        className="memo-editor-content w-full h-full"
        initialContent={state.content}
        placeholder={placeholder ?? ""}
        onContentChange={handleContentChange}
        onPaste={handlePaste}
        onKeyDown={handleKeyDown}
        onCompositionStart={handleCompositionStart}
        onCompositionEnd={handleCompositionEnd}
      />

      {/* Suggestion Menu */}
      {suggestions.isOpen && suggestions.cursorPosition && (
        <SuggestionMenu
          cursorPosition={suggestions.cursorPosition}
          items={suggestions.items}
          selectedIndex={suggestions.selectedIndex}
          onSelectItem={(item) => {
            item.action?.(editorRef.current!);
            setSuggestions((prev) => ({ ...prev, isOpen: false, trigger: null, cursorPosition: null }));
          }}
        />
      )}
    </div>
  );
});

EditorWithPlugins.displayName = "EditorWithPlugins";

/**
 * Simple wrapper component for overflow tip
 */
const SuggestionMenuItem = ({ item, isSelected, onSelect }: { item: SuggestionItem; isSelected: boolean; onSelect: () => void }) => {
  return (
    <div
      className={`flex items-center gap-2 px-3 py-2 text-sm cursor-pointer transition-colors select-none ${
        isSelected ? "bg-accent text-accent-foreground" : "hover:bg-accent/50"
      }`}
      onMouseDown={(e) => {
        e.preventDefault();
        onSelect();
      }}
    >
      {item.icon && <span className="shrink-0">{item.icon}</span>}
      <div className="flex-1 min-w-0">
        <OverflowTip>
          <span className="font-medium">{item.label}</span>
        </OverflowTip>
        {item.description && <p className="text-xs text-muted-foreground truncate">{item.description}</p>}
      </div>
    </div>
  );
};

/**
 * Suggestion Menu Component
 */
const SuggestionMenu = ({
  cursorPosition,
  items,
  selectedIndex,
  onSelectItem,
}: {
  cursorPosition: CursorPosition;
  items: SuggestionItem[];
  selectedIndex: number;
  onSelectItem: (item: SuggestionItem) => void;
}) => {
  const menuRef = useRef<HTMLDivElement>(null);
  const selectedItemRef = useRef<HTMLDivElement>(null);

  // Scroll selected item into view
  useEffect(() => {
    selectedItemRef.current?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }, [selectedIndex]);

  if (items.length === 0) return null;

  return (
    <div
      id="suggestion-menu"
      ref={menuRef}
      className="z-50 absolute p-1 max-w-64 max-h-60 rounded-md border bg-popover text-popover-foreground shadow-lg overflow-y-auto overflow-x-hidden"
      style={{
        left: cursorPosition.left,
        top: cursorPosition.top + (cursorPosition.height || 20) + 4,
      }}
    >
      {items.map((item, i) => (
        <div key={item.id} ref={i === selectedIndex ? selectedItemRef : null}>
          <SuggestionMenuItem item={item} isSelected={i === selectedIndex} onSelect={() => onSelectItem(item)} />
        </div>
      ))}
    </div>
  );
};

// Export sub-components for use in other parts of app
export { SuggestionMenu, SuggestionMenuItem };

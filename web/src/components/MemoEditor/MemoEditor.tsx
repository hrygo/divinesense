/**
 * MemoEditor - 完整的编辑器容器
 *
 * 集成 EnhancedEditor、SlashMenu 和所有插件系统
 * 支持：
 * - 斜杠命令 (/)
 * - 标签建议 (#)
 * - 列表自动完成 (-, 1., [])
 * - 键盘导航和快捷键
 */
import { memo, useCallback, useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import { EnhancedEditor } from "./core";
import { EnhancedEditorRefActions, type PluginContext, type SuggestionItem, type SuggestionMenuState } from "./core/editor-types";
import { createListAutocompletePlugin, createSlashCommandPlugin } from "./plugins";
import { SlashMenu } from "./plugins/slash-commands";

/**
 * MemoEditor Props
 */
export interface MemoEditorProps {
  className?: string;
  initialValue?: string;
  placeholder?: string;
  autoFocus?: boolean;
  onSubmit?: (content: string) => void;
  /**
   * 是否禁用编辑器
   */
  disabled?: boolean;
}

/**
 * 创建插件上下文
 */
function createPluginContext(
  editor: React.RefObject<EnhancedEditorRefActions>,
  content: string,
  dispatch: React.Dispatch<Record<string, unknown>>,
): PluginContext {
  return {
    editor: editor.current,
    content,
    cursor: null,
    selection: null,
    // biome-ignore lint/suspicious/noExplicitAny: Plugin dispatch uses Record<string, unknown>
    dispatch: dispatch as any,
  };
}

/**
 * MemoEditor Component
 */
const MemoEditorComponent = ({ initialValue = "", placeholder = "", autoFocus = false, onSubmit, disabled = false }: MemoEditorProps) => {
  const t = useTranslate();
  const editorRef = useRef<EnhancedEditorRefActions>(null);
  const [content, setContent] = useState(initialValue);

  // 建议菜单状态
  const [suggestions, setSuggestions] = useState<SuggestionMenuState>({
    isOpen: false,
    trigger: null,
    query: "",
    items: [],
    selectedIndex: -1,
    position: null,
  });

  // 初始化插件
  const plugins = [
    // biome-ignore lint/suspicious/noExplicitAny: Plugin create functions expect any
    createSlashCommandPlugin(t as any),
    // biome-ignore lint/suspicious/noExplicitAny: Plugin create functions expect any
    createListAutocompletePlugin(t as any),
  ];

  /**
   * 处理建议选择
   */
  const handleSuggestionSelect = useCallback(
    (item: SuggestionItem) => {
      item.action(editorRef.current!);
      // 关闭菜单
      setSuggestions({
        isOpen: false,
        trigger: null,
        query: "",
        items: [],
        selectedIndex: -1,
        position: null,
      });
    },
    [editorRef],
  );

  /**
   * 关闭建议菜单
   */
  const closeSuggestions = useCallback(() => {
    setSuggestions({
      isOpen: false,
      trigger: null,
      query: "",
      items: [],
      selectedIndex: -1,
      position: null,
    });
  }, []);

  /**
   * 创建分发函数
   */
  const createDispatch = useCallback(() => {
    return (action: Record<string, unknown>) => {
      switch (action.type) {
        case "SET_SUGGESTIONS":
          setSuggestions(action.payload as SuggestionMenuState);
          break;
        default:
          break;
      }
    };
  }, []);

  // 创建插件上下文
  const pluginContext: PluginContext = createPluginContext(editorRef, content, createDispatch());

  /**
   * 处理内容变化
   */
  const handleContentChange = useCallback(
    (newContent: string) => {
      setContent(newContent);

      // 通知所有插件内容变化
      plugins.forEach((plugin) => {
        plugin.onContentChange?.(pluginContext, newContent);
      });
    },
    [plugins, pluginContext],
  );

  /**
   * 处理键盘事件
   */
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      let handled = false;

      // 先让插件处理键盘事件
      for (const plugin of plugins) {
        if (plugin.onKeyDown) {
          const result = plugin.onKeyDown(pluginContext, e.nativeEvent);
          if (result) {
            handled = true;
            break;
          }
        }
      }

      // 默认键盘处理
      if (!handled) {
        // Ctrl/Cmd + Enter: 提交
        if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
          e.preventDefault();
          onSubmit?.(content);
          return;
        }

        // Esc: 关闭建议菜单
        if (e.key === "Escape" && suggestions.isOpen) {
          closeSuggestions();
          return;
        }
      }
    },
    [pluginContext, suggestions.isOpen, content, onSubmit, plugins, closeSuggestions],
  );

  /**
   * 处理粘贴事件
   */
  const handlePaste = useCallback(
    (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
      // 通知所有插件粘贴事件
      plugins.forEach((plugin) => {
        // biome-ignore lint/suspicious/noExplicitAny: Plugin onPaste types differ
        (plugin.onPaste as any)?.(pluginContext, e);
      });
    },
    [plugins, pluginContext],
  );

  /**
   * 处理 IME 组合开始
   */
  const handleCompositionStart = useCallback(() => {
    // 通知所有插件
    plugins.forEach((plugin) => {
      // biome-ignore lint/suspicious/noExplicitAny: Plugin onCompositionStart types differ
      (plugin.onCompositionStart as any)?.(pluginContext);
    });
  }, [plugins, pluginContext]);

  /**
   * 处理 IME 组合结束
   */
  const handleCompositionEnd = useCallback(() => {
    // 通知所有插件
    plugins.forEach((plugin) => {
      // biome-ignore lint/suspicious/noExplicitAny: Plugin onCompositionEnd types differ
      (plugin.onCompositionEnd as any)?.(pluginContext);
    });
  }, [plugins, pluginContext]);

  /**
   * 自动聚焦
   */
  useEffect(() => {
    if (autoFocus && editorRef.current) {
      editorRef.current.focus();
    }
  }, [autoFocus, editorRef]);

  /**
   * 同步初始内容
   */
  useEffect(() => {
    if (initialValue && initialValue !== content) {
      setContent(initialValue);
    }
  }, [initialValue]);

  return (
    <div className={cn("relative w-full", disabled && "opacity-50 pointer-events-none")}>
      {/* 建议菜单 */}
      <SlashMenu
        isOpen={suggestions.isOpen}
        items={suggestions.items}
        selectedIndex={suggestions.selectedIndex}
        position={suggestions.position}
        onSelect={handleSuggestionSelect}
        onClose={closeSuggestions}
      />

      {/* 编辑器 */}
      <EnhancedEditor
        ref={editorRef}
        className={cn(
          "w-full min-h-[120px] resize-none rounded-md border-border",
          "bg-background px-4 py-3 focus:outline-none focus:ring-2 focus:ring-accent/50",
          "focus:outline-none focus:ring-2 focus:ring-accent/20",
          "placeholder:text-muted-foreground/40",
          "text-foreground",
          disabled && "cursor-not-allowed",
        )}
        initialContent={content}
        placeholder={placeholder}
        onContentChange={handleContentChange}
        onPaste={handlePaste}
        onKeyDown={handleKeyDown}
        onCompositionStart={handleCompositionStart}
        onCompositionEnd={handleCompositionEnd}
      />

      {/* 字符计数 */}
      <div className="absolute bottom-3 right-4 text-xs text-muted-foreground/50 pointer-events-none">
        {content.length} {t("editor.chars")}
      </div>
    </div>
  );
};

const MemoEditor = memo(MemoEditorComponent);
MemoEditor.displayName = "MemoEditor";

export default MemoEditor;

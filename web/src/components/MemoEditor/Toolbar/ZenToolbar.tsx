/**
 * ZenToolbar - 禅意工具栏
 *
 * 设计哲学：「禅意智识」
 * - 呼吸感：工具栏如呼吸般轻柔展开
 * - 留白：功能按钮以禅意间距排列
 * - 不折叠：所有功能直观呈现，无需点击展开
 * - 微妙：所有交互都有柔和的视觉反馈
 *
 * ## 设计规范
 * - 间距：使用 4px 基础单位 (gap-1 = 4px, gap-2 = 8px)
 * - 圆角：统一 rounded-xl (12px)
 * - 高度：按钮 h-9 (36px)，保持一致性
 * - 动画：300ms ease-out，与呼吸同步
 * - 呼吸周期：3000ms，与 logo-breathe-gentle 同步
 *
 * ## 响应式策略
 * - 桌面端：水平工具栏，所有按钮一字排开
 * - 移动端：自动换行，保持拇指操作友好
 */

import { uniqBy } from "lodash-es";
import { FileImage, Link2, Loader2, MapPin, Maximize2, Send, Wand2 } from "lucide-react";
import { lazy, memo, Suspense, useCallback, useEffect, useState } from "react";
import { useDebounce } from "react-use";
import { useReverseGeocoding } from "@/components/map";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import VisibilityIcon from "@/components/VisibilityIcon";
import { cn } from "@/lib/utils";
import type { MemoRelation } from "@/types/proto/api/v1/memo_service_pb";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";
import { LinkMemoDialog } from "../components";
import { useFileUpload, useLinkMemo, useLocation } from "../hooks";
import { validationService } from "../services";
import { useEditorContext } from "../state";
// import type { EditorToolbarProps } from "../types/components";
import type { LocalFile } from "../types/attachment";

const LocationDialog = lazy(() => import("../components/LocationDialog").then((module) => ({ default: module.LocationDialog })));

// ============================================================================
// 设计常量
// ============================================================================

const BREATH_DURATION = 3000; // 与 logo-breathe-gentle 同步
const BUTTON_SIZE = "h-9 w-9"; // 统一按钮尺寸
const BUTTON_SIZE_MOBILE = "h-10 w-10"; // 移动端稍大，便于触摸

// ============================================================================
// 工具按钮组件
// ============================================================================

interface ZenToolButtonProps {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  shortcut?: string;
  is_active?: boolean;
  is_loading?: boolean;
  is_disabled?: boolean;
  tooltip_side?: "top" | "bottom" | "left" | "right";
  on_click: () => void;
  children?: React.ReactNode;
  className?: string;
}

/**
 * ZenToolButton - 禅意工具按钮
 *
 * 设计要点：
 * - 统一的呼吸节奏
 * - 柔和的悬停反馈
 * - 清晰的状态指示
 */
const ZenToolButton = memo(function ZenToolButton({
  icon: Icon,
  label,
  shortcut,
  is_active = false,
  is_loading = false,
  is_disabled = false,
  tooltip_side = "top",
  on_click,
  children,
  className,
}: ZenToolButtonProps) {
  const [isHovered, setIsHovered] = useState(false);

  return (
    <Tooltip delayDuration={300}>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={on_click}
          disabled={is_disabled || is_loading}
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
          className={cn(
            "relative group",
            // 统一尺寸
            "sm:h-9 sm:w-9 h-10 w-10",
            // 圆角
            "rounded-xl",
            // 过渡动画
            "transition-all duration-300 ease-out",
            // 状态样式
            is_active ? "bg-primary/10 text-primary shadow-inner" : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
            is_disabled && "opacity-40 cursor-not-allowed",
            // 聚焦环
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50",
            className,
          )}
          aria-label={label}
        >
          {/* 呼吸光晕 - 仅在悬停/激活时显示 */}
          {(isHovered || is_active) && !is_disabled && (
            <span
              className={cn("absolute inset-0 rounded-xl", "bg-primary/10", "animate-pulse")}
              style={{ animationDuration: `${BREATH_DURATION}ms` }}
            />
          )}

          {/* 图标 */}
          {is_loading ? (
            <Loader2 className="h-4 w-4 sm:h-4 sm:w-4 animate-spin" />
          ) : (
            <Icon className="h-4 w-4 sm:h-4 sm:w-4 relative z-10" />
          )}

          {/* 子元素（如下拉菜单） */}
          {children}
        </button>
      </TooltipTrigger>
      <TooltipContent side={tooltip_side} className="flex items-center gap-2">
        <span>{label}</span>
        {shortcut && <kbd className="rounded-md bg-muted/50 px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground">{shortcut}</kbd>}
      </TooltipContent>
    </Tooltip>
  );
});

// ============================================================================
// 可见性选择器
// ============================================================================

interface ZenVisibilitySelectorProps {
  value: Visibility;
  onChange: (value: Visibility) => void;
}

const VISIBILITY_OPTIONS = [
  { value: Visibility.PRIVATE, icon: "lock", label: "memo.visibility.private" },
  { value: Visibility.PROTECTED, icon: "workspace", label: "memo.visibility.protected" },
  { value: Visibility.PUBLIC, icon: "globe", label: "memo.visibility.public" },
] as const;

/**
 * ZenVisibilitySelector - 禅意可见性选择器
 *
 * 设计要点：
 * - 圆形切换器，而非下拉菜单
 * - 轻柔的状态切换动画
 * - 清晰的视觉反馈
 */
const ZenVisibilitySelector = memo(function ZenVisibilitySelector({ value, onChange }: ZenVisibilitySelectorProps) {
  const t = useTranslate();
  const [isExpanded, setIsExpanded] = useState(false);

  return (
    <div
      className={cn("relative inline-flex items-center", "transition-all duration-300")}
      onMouseEnter={() => setIsExpanded(true)}
      onMouseLeave={() => setIsExpanded(false)}
    >
      {/* 展开的面板 */}
      {isExpanded && (
        <div
          className={cn(
            "absolute bottom-full left-1/2 -translate-x-1/2 mb-2",
            "flex items-center gap-1 p-1",
            "rounded-xl bg-background border border-border/50 shadow-lg",
            "animate-in fade-in slide-in-from-bottom-1 duration-200",
          )}
        >
          {VISIBILITY_OPTIONS.map((option) => {
            const isSelected = value === option.value;
            return (
              <Tooltip key={option.value} delayDuration={200}>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    onClick={() => onChange(option.value)}
                    className={cn(
                      BUTTON_SIZE,
                      "rounded-lg flex items-center justify-center",
                      "transition-all duration-200",
                      isSelected ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
                    )}
                  >
                    <VisibilityIcon visibility={option.value} className="h-4 w-4" />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="top">{t(option.label)}</TooltipContent>
              </Tooltip>
            );
          })}
        </div>
      )}

      {/* 当前选择的按钮 */}
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>
          <button
            type="button"
            onClick={() => {
              // 循环切换可见性
              const currentIndex = VISIBILITY_OPTIONS.findIndex((opt) => opt.value === value);
              const nextIndex = (currentIndex + 1) % VISIBILITY_OPTIONS.length;
              onChange(VISIBILITY_OPTIONS[nextIndex].value);
            }}
            className={cn(
              "relative",
              BUTTON_SIZE,
              "rounded-xl flex items-center justify-center",
              "text-muted-foreground hover:text-foreground hover:bg-muted/50",
              "transition-all duration-300",
            )}
          >
            <VisibilityIcon visibility={value} className="h-4 w-4" />
          </button>
        </TooltipTrigger>
        <TooltipContent side="top">
          <div className="flex items-center gap-2">
            <span>
              {t(`memo.visibility.${value === Visibility.PRIVATE ? "private" : value === Visibility.PROTECTED ? "protected" : "public"}`)}
            </span>
            <span className="text-muted-foreground">→ 点击切换</span>
          </div>
        </TooltipContent>
      </Tooltip>
    </div>
  );
});

// ============================================================================
// AI 标签建议按钮
// ============================================================================

interface ZenAITagButtonProps {
  disabled: boolean;
  on_insert: (tags: string[]) => void;
}

/**
 * ZenAITagButton - AI 标签建议按钮
 *
 * 设计要点：
 * - 魔法棒图标代表 AI
 * - 点击后展开标签建议
 */
const ZenAITagButton = memo(function ZenAITagButton({ disabled, on_insert }: ZenAITagButtonProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [suggestions] = useState(["想法", "待办", "笔记", "灵感", "研究", "项目", "生活", "工作"]);

  return (
    <div className="relative">
      <ZenToolButton icon={Wand2} label="AI 标签" is_disabled={disabled} on_click={() => setIsOpen(!isOpen)} />

      {/* 展开的标签面板 */}
      {isOpen && (
        <div
          className={cn(
            "absolute bottom-full left-0 mb-2",
            "flex flex-wrap gap-1.5 p-3",
            "max-w-[200px] sm:max-w-none",
            "rounded-xl bg-background border border-border/50 shadow-lg",
            "animate-in fade-in slide-in-from-bottom-1 duration-200",
          )}
        >
          <p className="w-full text-xs text-muted-foreground mb-2">点击添加标签</p>
          {suggestions.map((tag) => (
            <button
              key={tag}
              type="button"
              onClick={() => {
                on_insert([tag]);
                setIsOpen(false);
              }}
              className={cn(
                "px-2.5 py-1 text-sm",
                "rounded-lg bg-muted/50 hover:bg-muted",
                "text-muted-foreground hover:text-foreground",
                "transition-colors duration-200",
              )}
            >
              #{tag}
            </button>
          ))}
        </div>
      )}
    </div>
  );
});

// ============================================================================
// 主工具栏组件
// ============================================================================

/**
 * ZenToolbar - 禅意工具栏
 *
 * 布局策略：
 * - 桌面端：左中右三段布局
 *   - 左侧：插入功能（文件、链接、位置、专注模式）
 *   - 中间：AI 辅助（标签建议）
 *   - 右侧：可见性 + 操作按钮
 * - 移动端：两行布局
 *   - 第一行：插入功能 + AI
 *   - 第二行：可见性 + 操作
 */
export const ZenToolbar = memo(function ZenToolbar({ onSave, onCancel, memoName }: ZenToolbarProps) {
  const t = useTranslate();
  const { state, actions, dispatch } = useEditorContext();
  const { valid } = validationService.canSave(state);

  const is_saving = state.ui.isLoading.saving;
  const [linkDialogOpen, setLinkDialogOpen] = useState(false);
  const [locationDialogOpen, setLocationDialogOpen] = useState(false);

  // 文件上传
  const { fileInputRef, selectingFlag, handleFileInputChange, handleUploadClick } = useFileUpload((newFiles: LocalFile[]) => {
    newFiles.forEach((file) => dispatch(actions.addLocalFile(file)));
  });

  // 链接笔记
  const linkMemo = useLinkMemo({
    isOpen: linkDialogOpen,
    currentMemoName: memoName,
    existingRelations: state.metadata.relations,
    onAddRelation: (relation: MemoRelation) => {
      dispatch(
        actions.setMetadata({
          relations: uniqBy([...state.metadata.relations, relation], (r) => r.relatedMemo?.name),
        }),
      );
      setLinkDialogOpen(false);
    },
  });

  // 位置
  const location = useLocation(state.metadata.location);
  const [debouncedPosition, setDebouncedPosition] = useState<{ lat: number; lng: number } | undefined>(undefined);

  useDebounce(
    () => {
      setDebouncedPosition(location.state.position);
    },
    1000,
    [location.state.position],
  );

  const { data: displayName } = useReverseGeocoding(debouncedPosition?.lat, debouncedPosition?.lng);

  useEffect(() => {
    if (displayName) {
      location.setPlaceholder(displayName);
    }
  }, [displayName]);

  const isUploading = selectingFlag || state.ui.isLoading.uploading;

  // 处理位置点击
  const handleLocationClick = useCallback(() => {
    setLocationDialogOpen(true);
    if (!state.metadata.location && !location.locationInitialized) {
      if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(
          (position) => {
            location.handlePositionChange({
              lat: position.coords.latitude,
              lng: position.coords.longitude,
            });
          },
          (error) => {
            console.error("Geolocation error:", error);
          },
        );
      }
    }
  }, [state.metadata.location, location]);

  const handleLocationConfirm = useCallback(() => {
    const newLocation = location.getLocation();
    if (newLocation) {
      dispatch(actions.setMetadata({ location: newLocation }));
      setLocationDialogOpen(false);
    }
  }, [location, dispatch]);

  const handleLocationCancel = useCallback(() => {
    location.reset();
    setLocationDialogOpen(false);
  }, [location]);

  const handlePositionChange = useCallback(
    (position: { lat: number; lng: number }) => {
      location.handlePositionChange(position);
    },
    [location],
  );

  const handleToggleFocusMode = useCallback(() => {
    dispatch(actions.toggleFocusMode());
  }, [dispatch]);

  const handleVisibilityChange = useCallback(
    (visibility: Visibility) => {
      dispatch(actions.setMetadata({ visibility }));
    },
    [dispatch],
  );

  const handleInsertTags = useCallback(
    (tags: string[]) => {
      if (tags.length > 0) {
        const newTags = tags.map((tag) => `#${tag}`).join(" ");
        const newContent = state.content + (state.content.endsWith("\n") ? "" : "\n") + newTags;
        dispatch(actions.updateContent(newContent));
      }
    },
    [state.content, dispatch],
  );

  // 移动端：检测是否为小屏幕（保留但未使用，可由 CSS 媒体查询处理）
  // const [isMobile, setIsMobile] = useState(false);
  // useEffect(() => {
  //   const checkMobile = () => setIsMobile(window.innerWidth < 640);
  //   checkMobile();
  //   window.addEventListener("resize", checkMobile);
  //   return () => window.removeEventListener("resize", checkMobile);
  // }, []);

  return (
    <>
      {/* 桌面端工具栏 */}
      <div className="hidden sm:flex items-center justify-between gap-3">
        {/* 左侧：插入功能 */}
        <div className="flex items-center gap-1">
          {/* 文件上传 */}
          <ZenToolButton icon={FileImage} label={t("common.upload")} is_loading={isUploading} on_click={handleUploadClick} />

          {/* 关联笔记 */}
          <ZenToolButton icon={Link2} label={t("tooltip.link-memo")} on_click={() => setLinkDialogOpen(true)} />

          {/* 添加位置 */}
          <ZenToolButton
            icon={MapPin}
            label={t("tooltip.select-location")}
            is_active={!!state.metadata.location}
            on_click={handleLocationClick}
          />

          {/* 专注模式 */}
          <ZenToolButton icon={Maximize2} label={t("editor.focus-mode")} shortcut="⌘⇧F" on_click={handleToggleFocusMode} />
        </div>

        {/* 中间：AI 辅助 */}
        <ZenAITagButton disabled={is_saving} on_insert={handleInsertTags} />

        {/* 右侧：可见性 + 操作 */}
        <div className="flex items-center gap-1">
          <ZenVisibilitySelector value={state.metadata.visibility} onChange={handleVisibilityChange} />

          {onCancel && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onCancel}
              disabled={is_saving}
              className="h-9 px-3 text-muted-foreground hover:text-foreground"
            >
              {t("common.cancel")}
            </Button>
          )}

          <Button onClick={onSave} disabled={!valid || is_saving} className="h-9 px-4">
            {is_saving ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                {t("common.saving")}
              </span>
            ) : (
              <span className="flex items-center gap-2">
                {t("editor.save")}
                <Send className="h-3.5 w-3.5" />
              </span>
            )}
          </Button>
        </div>
      </div>

      {/* 移动端工具栏 */}
      <div className="sm:hidden">
        {/* 第一行：插入功能 + AI */}
        <div className="flex items-center justify-between gap-2 mb-2">
          <div className="flex items-center gap-1">
            <ZenToolButton
              icon={FileImage}
              label={t("common.upload")}
              is_loading={isUploading}
              on_click={handleUploadClick}
              className={BUTTON_SIZE_MOBILE}
            />
            <ZenToolButton
              icon={Link2}
              label={t("tooltip.link-memo")}
              on_click={() => setLinkDialogOpen(true)}
              className={BUTTON_SIZE_MOBILE}
            />
            <ZenToolButton
              icon={MapPin}
              label={t("tooltip.select-location")}
              is_active={!!state.metadata.location}
              on_click={handleLocationClick}
              className={BUTTON_SIZE_MOBILE}
            />
            <ZenToolButton
              icon={Maximize2}
              label={t("editor.focus-mode")}
              on_click={handleToggleFocusMode}
              className={BUTTON_SIZE_MOBILE}
            />
          </div>

          <ZenAITagButton disabled={is_saving} on_insert={handleInsertTags} />
        </div>

        {/* 第二行：可见性 + 操作 */}
        <div className="flex items-center justify-between gap-2">
          <ZenVisibilitySelector value={state.metadata.visibility} onChange={handleVisibilityChange} />

          <div className="flex items-center gap-2 flex-1 justify-end">
            {onCancel && (
              <button
                type="button"
                onClick={onCancel}
                disabled={is_saving}
                className={cn(
                  "h-10 px-4 rounded-xl",
                  "text-muted-foreground hover:text-foreground hover:bg-muted/50",
                  "transition-colors duration-200",
                  "text-sm font-medium",
                )}
              >
                {t("common.cancel")}
              </button>
            )}

            <button
              type="button"
              onClick={onSave}
              disabled={!valid || is_saving}
              className={cn(
                "h-10 px-4 rounded-xl flex items-center gap-2",
                "transition-all duration-200",
                valid && !is_saving
                  ? "bg-primary text-primary-foreground shadow-md shadow-primary/20 hover:bg-primary/90 active:scale-95"
                  : "bg-muted text-muted-foreground cursor-not-allowed opacity-50",
              )}
            >
              {is_saving ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span className="text-sm">{t("common.saving")}</span>
                </>
              ) : (
                <>
                  <span className="text-sm">{t("editor.save")}</span>
                  <Send className="h-4 w-4" />
                </>
              )}
            </button>
          </div>
        </div>
      </div>

      {/* 隐藏的文件输入 */}
      <input
        className="hidden"
        ref={fileInputRef}
        disabled={isUploading}
        onChange={handleFileInputChange}
        type="file"
        multiple
        accept="*"
      />

      {/* 链接笔记对话框 */}
      <LinkMemoDialog
        open={linkDialogOpen}
        onOpenChange={setLinkDialogOpen}
        searchText={linkMemo.searchText}
        onSearchChange={linkMemo.setSearchText}
        filteredMemos={linkMemo.filteredMemos}
        isFetching={linkMemo.isFetching}
        onSelectMemo={linkMemo.addMemoRelation}
      />

      {/* 位置选择对话框 */}
      {locationDialogOpen && (
        <Suspense fallback={null}>
          <LocationDialog
            open={locationDialogOpen}
            onOpenChange={setLocationDialogOpen}
            state={location.state}
            locationInitialized={location.locationInitialized}
            onPositionChange={handlePositionChange}
            onUpdateCoordinate={location.updateCoordinate}
            onPlaceholderChange={location.setPlaceholder}
            onCancel={handleLocationCancel}
            onConfirm={handleLocationConfirm}
          />
        </Suspense>
      )}
    </>
  );
});

ZenToolbar.displayName = "ZenToolbar";

// ============================================================================
// 类型别名
// ============================================================================

export interface ZenToolbarProps {
  onSave: () => void;
  onCancel?: () => void;
  memoName?: string;
}

// ============================================================================
// LocationDialog 类型修复
// ============================================================================

// import type { LocationState } from "../types/insert-menu";

// ============================================================================
// 全局样式注入
// ============================================================================

if (typeof document !== "undefined") {
  const styleId = "zen-toolbar-animations";
  if (!document.getElementById(styleId)) {
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = `
      /* 呼吸动画 - 与 logo 同步 */
      @keyframes zen-breathe {
        0%, 100% {
          opacity: 0.3;
          transform: scale(1);
        }
        50% {
          opacity: 0.6;
          transform: scale(1.05);
        }
      }

      /* 工具栏进入动画 */
      @keyframes zen-slide-in {
        from {
          opacity: 0;
          transform: translateY(8px);
        }
        to {
          opacity: 1;
          transform: translateY(0);
        }
      }
    `;
    document.head.appendChild(style);
  }
}

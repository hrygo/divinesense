import { create } from "@bufbuild/protobuf";
import { useMutation } from "@tanstack/react-query";
import { uniqBy } from "lodash-es";
import { Minimize2Icon } from "lucide-react";
import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { memoServiceClient } from "@/connect";
import { handleError } from "@/lib/error";
import { cn } from "@/lib/utils";
import { Location, MemoRelation, MemoSchema, Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";
import { LinkMemoDialog } from "./components";
import { FocusModeOverlay as FocusModeOverlayComponent } from "./components/FocusModeOverlay";
import Editor from "./Editor";
import { useFileUpload, useLinkMemo, useLocation } from "./hooks";
import { StandardToolbar } from "./StandardToolbar";

const LocationDialogLazy = lazy(() => import("./components/LocationDialog").then((module) => ({ default: module.LocationDialog })));

interface FocusModeEditorProps {
  /** Callback when exited */
  onExit: () => void;
  /** Optional initial content for editing */
  initialContent?: string;
  /** Callback when memo is successfully created/updated */
  onSuccess?: (memoName: string) => void;
  /** Custom placeholder text */
  placeholder?: string;
  /** Additional CSS classes */
  className?: string;
}

/**
 * FocusModeEditor - 全屏无干扰编辑模式（带动画优化）
 *
 * Features:
 * - 进入/退出动画状态管理
 * - Fullscreen overlay with backdrop blur
 * - Centered editor container
 * - Complete toolbar functionality
 * - ESC to exit
 * - Click outside to exit
 *
 * 优化:
 * - 使用动画状态提升视觉体验
 * - 保存/恢复滚动位置
 * - 平滑的进入/退出过渡
 */
export function FocusModeEditor({ onExit, initialContent = "", onSuccess, placeholder, className }: FocusModeEditorProps) {
  const t = useTranslate();
  const [content, setContent] = useState(initialContent);
  const [visibility, setVisibility] = useState(Visibility.PRIVATE);
  const [location, setLocation] = useState<Location | undefined>();
  const [relations, setRelations] = useState<MemoRelation[]>([]);

  // 动画状态
  const [isExiting, setIsExiting] = useState(false);
  const [isVisible, setIsVisible] = useState(false);

  // 组件挂载时触发进入动画
  useEffect(() => {
    // 保存当前滚动位置
    const scrollY = window.scrollY;
    const scrollX = window.scrollX;

    // 短暂延迟后显示内容，触发进入动画
    const showTimer = requestAnimationFrame(() => {
      setIsVisible(true);
    });

    return () => {
      cancelAnimationFrame(showTimer);
      // 退出时恢复滚动位置
      window.scrollTo(scrollX, scrollY);
    };
  }, []);

  // Dialogs state
  const [linkDialogOpen, setLinkDialogOpen] = useState(false);
  const [locationDialogOpen, setLocationDialogOpen] = useState(false);

  // Location hook
  const locationHook = useLocation(location);

  // File upload hook
  const { fileInputRef, selectingFlag, handleFileInputChange, handleUploadClick } = useFileUpload(() => {
    console.log("[FocusModeEditor] Files selected");
  });

  // Link memo hook
  const linkMemo = useLinkMemo({
    isOpen: linkDialogOpen,
    currentMemoName: undefined,
    existingRelations: relations,
    onAddRelation: (relation: MemoRelation) => {
      setRelations((prev) => uniqBy([...prev, relation], (r) => r.relatedMemo?.name));
      setLinkDialogOpen(false);
    },
  });

  // Memo creation mutation
  const createMemo = useMutation({
    mutationFn: async (contentParam: string) => {
      const memo = create(MemoSchema, {
        content: contentParam,
        visibility,
        location: locationHook.getLocation(),
        relations,
      });

      const response = await memoServiceClient.createMemo({ memo });
      return response;
    },
    onSuccess: (data) => {
      setContent("");
      setLocation(undefined);
      setRelations([]);
      onSuccess?.(data.name);
      // 保存成功后退出焦点模式
      handleExit();
    },
    onError: (error) => {
      handleError(error, toast.error, {
        context: "Failed to create memo",
        fallbackMessage: "创建笔记失败，请重试",
      });
    },
  });

  const handleSave = useCallback(() => {
    if (!content.trim()) return;
    createMemo.mutate(content);
  }, [content, createMemo]);

  const handleInsertTags = useCallback((tags: string[]) => {
    if (tags.length > 0) {
      const newTags = tags.map((tag) => `#${tag}`).join(" ");
      setContent((prev) => prev + (prev.endsWith("\n") ? "" : "\n") + newTags);
    }
  }, []);

  const handleLocationChange = useCallback((newLocation: Location) => {
    setLocation(newLocation);
  }, []);

  const handleLocationConfirm = useCallback(() => {
    const newLocation = locationHook.getLocation();
    if (newLocation) {
      handleLocationChange(newLocation);
      setLocationDialogOpen(false);
    }
  }, [locationHook, handleLocationChange]);

  // 退出焦点模式（带动画）
  const handleExit = useCallback(() => {
    setIsExiting(true);
    // 等待退出动画完成后调用 onExit
    setTimeout(() => {
      onExit();
    }, 200);
  }, [onExit]);

  const isValid = content.trim().length > 0;
  const isUploading = selectingFlag || createMemo.isPending;

  // ESC 键退出
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !isExiting) {
        e.preventDefault();
        handleExit();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isExiting, handleExit]);

  return (
    <>
      {/* Backdrop overlay with exit animation */}
      <FocusModeOverlayComponent isActive={!isExiting} isExiting={isExiting} onToggle={handleExit} />

      {/* Main focus mode container with animation */}
      <div
        className={cn(
          "fixed z-50 w-auto max-w-5xl mx-auto shadow-lg border-border bg-background rounded-lg overflow-hidden",
          // 进入动画
          isVisible && "animate-in fade-in-0 zoom-in-95 duration-300",
          // 退出动画
          isExiting && "animate-out fade-out-0 zoom-out-95 duration-200",
          // 过渡动画
          "transition-all duration-300 ease-in-out",
          // 定位
          "top-2 left-2 right-2 bottom-2 sm:top-4 sm:left-4 sm:right-4 sm:bottom-4",
          "md:top-8 md:left-8 md:right-8 md:bottom-8",
          className,
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-2 border-b border-border bg-muted/50">
          <span className="text-sm font-medium text-muted-foreground">Focus Mode</span>
          <button
            onClick={handleExit}
            className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors px-2 py-1 rounded hover:bg-accent"
          >
            <Minimize2Icon className="w-4 h-4" />
            Exit (ESC)
          </button>
        </div>

        {/* Editor Content */}
        <div className="flex-1 min-h-[300px] max-h-[60vh] overflow-y-auto px-6 py-4">
          <Editor
            className="min-h-[300px]"
            initialContent={content}
            placeholder={placeholder || t("editor.focus-mode-placeholder")}
            onContentChange={setContent}
            onPaste={() => {}}
            isFocusMode={true}
          />
        </div>

        {/* Toolbar */}
        <StandardToolbar
          content={content}
          isLoading={isUploading}
          isValid={isValid}
          visibility={visibility}
          onVisibilityChange={setVisibility}
          onInsertTags={handleInsertTags}
          onSave={handleSave}
          onUploadFile={handleUploadClick}
          onLinkMemo={() => setLinkDialogOpen(true)}
          onAddLocation={() => {
            setLocationDialogOpen(true);
            if (!location && !locationHook.locationInitialized) {
              if (navigator.geolocation) {
                navigator.geolocation.getCurrentPosition(
                  (position) => {
                    locationHook.handlePositionChange({
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
          }}
          onToggleFocusMode={handleExit}
        />
      </div>

      {/* Hidden file input */}
      <input
        className="hidden"
        ref={fileInputRef}
        disabled={isUploading}
        onChange={handleFileInputChange}
        type="file"
        multiple
        accept="*"
      />

      {/* Link Memo Dialog */}
      <LinkMemoDialog
        open={linkDialogOpen}
        onOpenChange={setLinkDialogOpen}
        searchText={linkMemo.searchText}
        onSearchChange={linkMemo.setSearchText}
        filteredMemos={linkMemo.filteredMemos}
        isFetching={linkMemo.isFetching}
        onSelectMemo={linkMemo.addMemoRelation}
      />

      {/* Location Dialog */}
      {locationDialogOpen && (
        <Suspense fallback={null}>
          <LocationDialogLazy
            open={locationDialogOpen}
            onOpenChange={setLocationDialogOpen}
            state={locationHook.state}
            locationInitialized={locationHook.locationInitialized}
            onPositionChange={locationHook.handlePositionChange}
            onUpdateCoordinate={locationHook.updateCoordinate}
            onPlaceholderChange={locationHook.setPlaceholder}
            onCancel={() => {
              locationHook.reset();
              setLocationDialogOpen(false);
            }}
            onConfirm={handleLocationConfirm}
          />
        </Suspense>
      )}
    </>
  );
}

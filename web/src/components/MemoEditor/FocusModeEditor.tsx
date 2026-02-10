import { create } from "@bufbuild/protobuf";
import { useMutation } from "@tanstack/react-query";
import { uniqBy } from "lodash-es";
import { Minimize2Icon } from "lucide-react";
import { lazy, Suspense, useCallback, useState } from "react";
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
 * FocusModeEditor - 全屏无干扰编辑模式
 *
 * Features:
 * - Fullscreen overlay with backdrop blur
 * - Centered editor container
 * - Complete toolbar functionality
 * - ESC to exit
 * - Click outside to exit
 */
export function FocusModeEditor({ onExit, initialContent = "", onSuccess, placeholder, className }: FocusModeEditorProps) {
  const t = useTranslate();
  const [content, setContent] = useState(initialContent);
  const [visibility, setVisibility] = useState(Visibility.PRIVATE);
  const [location, setLocation] = useState<Location | undefined>();
  const [relations, setRelations] = useState<MemoRelation[]>([]);

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
      onExit(); // Exit focus mode after save
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
  }, [content]);

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

  const isValid = content.trim().length > 0;
  const isUploading = selectingFlag || createMemo.isPending;

  return (
    <>
      {/* Backdrop overlay */}
      <FocusModeOverlayComponent isActive onToggle={onExit} />

      {/* Main focus mode container */}
      <div
        className={cn(
          "fixed z-50 w-auto max-w-5xl mx-auto shadow-lg border-border bg-background rounded-lg overflow-hidden",
          "transition-all duration-300 ease-in-out",
          "top-2 left-2 right-2 bottom-2 sm:top-4 sm:left-4 sm:right-4 sm:bottom-4",
          "md:top-8 md:left-8 md:right-8 md:bottom-8",
          className,
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-2 border-b border-border bg-muted/50">
          <span className="text-sm font-medium text-muted-foreground">Focus Mode</span>
          <button
            onClick={onExit}
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
          onToggleFocusMode={onExit}
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

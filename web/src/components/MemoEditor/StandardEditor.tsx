import { create } from "@bufbuild/protobuf";
import { useMutation } from "@tanstack/react-query";
import { uniqBy } from "lodash-es";
import { lazy, Suspense, useCallback, useState } from "react";
import { toast } from "react-hot-toast";
import { memoServiceClient } from "@/connect";
import { handleError } from "@/lib/error";
import { cn } from "@/lib/utils";
import { Location, MemoRelation, MemoSchema } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";
import { LinkMemoDialog } from "./components";
import Editor from "./Editor";
import { useFileUpload, useLinkMemo, useLocation } from "./hooks";
import { StandardToolbar } from "./StandardToolbar";

const LocationDialogLazy = lazy(() => import("./components/LocationDialog").then((module) => ({ default: module.LocationDialog })));

interface StandardEditorProps {
  /** Optional initial content for editing */
  initialContent?: string;
  /** Callback when memo is successfully created/updated */
  onSuccess?: (memoName: string) => void;
  /** Callback when cancelled */
  onCancel?: () => void;
  /** Custom placeholder text */
  placeholder?: string;
  /** Additional CSS classes */
  className?: string;
  /** Callback when focus mode is toggled */
  onToggleFocusMode?: () => void;
}

/**
 * StandardEditor - PC端标准模式完整编辑器
 *
 * Features:
 * - Full toolbar with all features
 * - Rich text editing with Markdown support
 * - AI tag suggestion
 * - Visibility control
 * - File upload, link memo, location support
 */
export function StandardEditor({
  initialContent = "",
  onSuccess,
  onCancel,
  placeholder,
  className,
  onToggleFocusMode,
}: StandardEditorProps) {
  const t = useTranslate();
  const [content, setContent] = useState(initialContent);
  const [visibility, setVisibility] = useState(1); // PRIVATE
  const [location, setLocation] = useState<Location | undefined>();
  const [relations, setRelations] = useState<MemoRelation[]>([]);

  // Dialogs state
  const [linkDialogOpen, setLinkDialogOpen] = useState(false);
  const [locationDialogOpen, setLocationDialogOpen] = useState(false);

  // Location hook
  const locationHook = useLocation(location);

  // File upload hook
  const { fileInputRef, selectingFlag, handleFileInputChange, handleUploadClick } = useFileUpload(() => {
    // Files are added to the state, but actual upload will be handled separately
    console.log("[StandardEditor] Files selected, upload to be implemented");
  });

  // Link memo hook
  const linkMemo = useLinkMemo({
    isOpen: linkDialogOpen,
    currentMemoName: undefined, // New memo, no existing name
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
        relations: relations,
      });

      const response = await memoServiceClient.createMemo({ memo });
      return response;
    },
    onSuccess: (data) => {
      setContent("");
      setLocation(undefined);
      setRelations([]);
      onSuccess?.(data.name);
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
      <div className={cn("flex flex-col border rounded-lg bg-background shadow-sm", className)}>
        {/* Editor Content */}
        <div className="flex-1 min-h-[120px] max-h-[400px] overflow-y-auto">
          <Editor
            className="min-h-[120px]"
            initialContent={content}
            placeholder={placeholder || t("editor.quick-placeholder")}
            onContentChange={setContent}
            onPaste={() => {}}
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
          onCancel={onCancel}
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
          onToggleFocusMode={onToggleFocusMode}
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

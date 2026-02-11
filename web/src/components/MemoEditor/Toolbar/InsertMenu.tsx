/**
 * InsertMenu - 禅意插入菜单
 *
 * 设计哲学：「禅意智识」
 * - 微妙的触发器：+ 号如呼吸般存在
 * - 流动：菜单项自然排列，有呼吸的韵律
 * - 渐进：高级功能隐于"更多"中
 * - 意图：每个图标都有清晰的含义
 *
 * ## 功能保留
 * - 上传文件
 * - 关联笔记
 * - 添加位置
 * - 专注模式
 * - Slash Commands 提示
 */

import { uniqBy } from "lodash-es";
import { File, Link, Loader2, MapPin, Maximize2, MoreHorizontal, Plus } from "lucide-react";
import { lazy, Suspense, useEffect, useState } from "react";
import { useDebounce } from "react-use";
import { useReverseGeocoding } from "@/components/map";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
  useDropdownMenuSubHoverDelay,
} from "@/components/ui/dropdown-menu";
import type { MemoRelation } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";
import { LinkMemoDialog } from "../components";
import { useFileUpload, useLinkMemo, useLocation } from "../hooks";
import { useEditorContext } from "../state";
import type { InsertMenuProps } from "../types";
import type { LocalFile } from "../types/attachment";

const LocationDialog = lazy(() => import("../components/LocationDialog").then((module) => ({ default: module.LocationDialog })));

const InsertMenu = (props: InsertMenuProps) => {
  const t = useTranslate();
  const { state, actions, dispatch } = useEditorContext();

  const [linkDialogOpen, setLinkDialogOpen] = useState(false);
  const [locationDialogOpen, setLocationDialogOpen] = useState(false);
  const [moreSubmenuOpen, setMoreSubmenuOpen] = useState(false);

  const { handleTriggerEnter, handleTriggerLeave, handleContentEnter, handleContentLeave } = useDropdownMenuSubHoverDelay(
    150,
    setMoreSubmenuOpen,
  );

  const { fileInputRef, selectingFlag, handleFileInputChange, handleUploadClick } = useFileUpload((newFiles: LocalFile[]) => {
    newFiles.forEach((file) => dispatch(actions.addLocalFile(file)));
  });

  const linkMemo = useLinkMemo({
    isOpen: linkDialogOpen,
    currentMemoName: props.memoName,
    existingRelations: state.metadata.relations,
    onAddRelation: (relation: MemoRelation) => {
      dispatch(actions.setMetadata({ relations: uniqBy([...state.metadata.relations, relation], (r) => r.relatedMemo?.name) }));
      setLinkDialogOpen(false);
    },
  });

  const location = useLocation(props.location);

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

  const isUploading = selectingFlag || props.isUploading;

  const handleLocationClick = () => {
    setLocationDialogOpen(true);
    if (!props.location && !location.locationInitialized) {
      if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(
          (position) => {
            location.handlePositionChange({ lat: position.coords.latitude, lng: position.coords.longitude });
          },
          (error) => {
            console.error("Geolocation error:", error);
          },
        );
      }
    }
  };

  const handleLocationConfirm = () => {
    const newLocation = location.getLocation();
    if (newLocation) {
      props.onLocationChange(newLocation);
      setLocationDialogOpen(false);
    }
  };

  const handleLocationCancel = () => {
    location.reset();
    setLocationDialogOpen(false);
  };

  const handlePositionChange = (position: { lat: number; lng: number }) => {
    location.handlePositionChange(position);
  };

  return (
    <>
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="h-9 w-9 text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
            disabled={isUploading}
          >
            {isUploading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-48">
          {/* 主要功能 */}
          <DropdownMenuItem onClick={handleUploadClick} className="cursor-pointer">
            <File className="h-4 w-4 text-muted-foreground" />
            <span className="ml-2">{t("common.upload")}</span>
          </DropdownMenuItem>

          <DropdownMenuItem onClick={() => setLinkDialogOpen(true)} className="cursor-pointer">
            <Link className="h-4 w-4 text-muted-foreground" />
            <span className="ml-2">{t("tooltip.link-memo")}</span>
          </DropdownMenuItem>

          <DropdownMenuItem onClick={handleLocationClick} className="cursor-pointer">
            <MapPin className="h-4 w-4 text-muted-foreground" />
            <span className="ml-2">{t("tooltip.select-location")}</span>
          </DropdownMenuItem>

          <DropdownMenuSeparator />

          {/* 更多功能子菜单 */}
          <DropdownMenuSub open={moreSubmenuOpen} onOpenChange={setMoreSubmenuOpen}>
            <DropdownMenuSubTrigger onPointerEnter={handleTriggerEnter} onPointerLeave={handleTriggerLeave} className="cursor-pointer">
              <MoreHorizontal className="h-4 w-4 text-muted-foreground" />
              <span className="ml-2 flex-1">{t("common.more")}</span>
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent onPointerEnter={handleContentEnter} onPointerLeave={handleContentLeave} className="w-40">
              <DropdownMenuItem
                className="cursor-pointer"
                onClick={() => {
                  props.onToggleFocusMode?.();
                  setMoreSubmenuOpen(false);
                }}
              >
                <Maximize2 className="h-4 w-4 text-muted-foreground" />
                <span className="ml-2 flex-1">{t("editor.focus-mode")}</span>
                <span className="text-xs text-muted-foreground/50">⌘⇧F</span>
              </DropdownMenuItem>
            </DropdownMenuSubContent>
          </DropdownMenuSub>

          {/* 底部提示 */}
          <div className="px-2 py-1.5 text-xs text-muted-foreground/50 text-center">/ {t("editor.slash-commands")}</div>
        </DropdownMenuContent>
      </DropdownMenu>

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

      <LinkMemoDialog
        open={linkDialogOpen}
        onOpenChange={setLinkDialogOpen}
        searchText={linkMemo.searchText}
        onSearchChange={linkMemo.setSearchText}
        filteredMemos={linkMemo.filteredMemos}
        isFetching={linkMemo.isFetching}
        onSelectMemo={linkMemo.addMemoRelation}
      />

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
};

export default InsertMenu;

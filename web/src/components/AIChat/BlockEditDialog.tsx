/**
 * BlockEditDialog - 编辑用户输入对话框
 *
 * 功能：
 * 1. 展示原始用户消息（支持多条输入）
 * 2. 提供编辑区域（Textarea）
 * 3. 确认/取消按钮
 * 4. 编辑完成后返回修改内容（不再创建分支）
 *
 * @note 分支功能已暂时移除，编辑现在仅返回修改后的文本
 */

import { Pencil } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

interface BlockEditDialogProps {
  /** 原始用户消息内容（多条已合并） */
  originalMessage: string;
  /** 对话框是否打开 */
  open: boolean;
  /** 关闭对话框 */
  onOpenChange: (open: boolean) => void;
  /** 确认编辑回调 - 返回编辑后的内容 */
  onConfirm?: (editedMessage: string) => void;
}

/**
 * BlockEditDialog 组件
 *
 * 允许用户编辑已发送的消息。
 * 编辑完成后，返回修改后的文本给调用方处理。
 *
 * 简化设计：
 * - 直接在编辑区显示原始消息
 * - 移除分支创建相关功能
 * - 编辑后仅返回文本，由调用方决定如何处理
 */
export function BlockEditDialog({ originalMessage, open, onOpenChange, onConfirm }: BlockEditDialogProps) {
  const { t } = useTranslation();
  const [editedMessage, setEditedMessage] = useState(originalMessage);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // 当对话框打开时，重置编辑内容为原始消息
  useEffect(() => {
    if (open) {
      setEditedMessage(originalMessage);
    }
  }, [open, originalMessage]);

  // 重置编辑内容当对话框关闭时
  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      setIsSubmitting(false);
    }
    onOpenChange(newOpen);
  };

  // 确认编辑
  const handleConfirm = async () => {
    const trimmed = editedMessage.trim();
    if (!trimmed || trimmed === originalMessage.trim()) {
      return;
    }

    setIsSubmitting(true);

    try {
      // 调用父组件传入的回调，仅返回编辑后的内容
      await onConfirm?.(trimmed);

      // 关闭对话框
      onOpenChange(false);
    } finally {
      setIsSubmitting(false);
    }
  };

  const hasChanges = editedMessage.trim() !== originalMessage.trim();
  const isValid = editedMessage.trim().length > 0;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-[32rem]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Pencil className="w-5 h-5" />
            {t("ai.unified_block.edit_title")}
          </DialogTitle>
          <DialogDescription>{t("ai.unified_block.edit_description_simple") || t("ai.unified_block.edit_description")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* 编辑区域 - 直接显示原始消息，可编辑 */}
          <div className="space-y-2">
            <label htmlFor="edit-input" className="text-sm font-medium">
              {t("ai.unified_block.message")}
            </label>
            <Textarea
              id="edit-input"
              value={editedMessage}
              onChange={(e) => setEditedMessage(e.target.value)}
              placeholder={t("ai.unified_block.edit_placeholder")}
              rows={5}
              className={cn("resize-y", !hasChanges && "border-muted-foreground/50")}
              autoFocus
            />
          </div>
        </div>

        <DialogFooter className="gap-2">
          <Button type="button" variant="outline" onClick={() => handleOpenChange(false)} disabled={isSubmitting}>
            {t("ai.unified_block.cancel")}
          </Button>
          <Button type="button" onClick={handleConfirm} disabled={!hasChanges || !isValid || isSubmitting}>
            {isSubmitting ? t("states.processing") : t("common.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/**
 * Hook: useBlockEditDialog
 *
 * 管理编辑对话框状态的便捷 Hook
 */
export function useBlockEditDialog() {
  const [open, setOpen] = useState(false);
  const [originalMessage, setOriginalMessage] = useState("");

  /**
   * 打开编辑对话框
   * @param message - 原始消息（如果有多条，调用方已合并）
   */
  const openDialog = (message: string) => {
    setOriginalMessage(message);
    setOpen(true);
  };

  const closeDialog = () => {
    setOpen(false);
  };

  return {
    open,
    originalMessage,
    openDialog,
    closeDialog,
    setOpen,
  };
}

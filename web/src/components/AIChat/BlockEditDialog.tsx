/**
 * BlockEditDialog - 编辑用户输入对话框
 *
 * 功能：
 * 1. 显示原始用户消息
 * 2. 提供编辑区域（Textarea）
 * 3. 解释编辑将创建新分支
 * 4. 确认/取消按钮
 * 5. 调用 ForkBlock API（待实现）
 *
 * @see docs/specs/block-design/ai-chat-interface-gap-analysis.md P0-A001
 */

import { AlertTriangle, Pencil } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

interface BlockEditDialogProps {
  /** 原始用户消息内容 */
  originalMessage: string;
  /** Block ID（用于 Fork API） */
  blockId: bigint;
  /** 对话 ID */
  conversationId: number;
  /** 对话框是否打开 */
  open: boolean;
  /** 关闭对话框 */
  onOpenChange: (open: boolean) => void;
  /** 确认编辑回调 */
  onConfirm?: (editedMessage: string, blockId: bigint, conversationId: number) => void;
}

/**
 * BlockEditDialog 组件
 *
 * 允许用户编辑已发送的消息，创建新分支并重新生成。
 * 这是实现对话分支功能的关键组件。
 */
export function BlockEditDialog({ originalMessage, blockId, conversationId, open, onOpenChange, onConfirm }: BlockEditDialogProps) {
  const { t } = useTranslation();
  const [editedMessage, setEditedMessage] = useState(originalMessage);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // 重置编辑内容当对话框打开时
  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      // 关闭时重置
      setEditedMessage(originalMessage);
      setIsSubmitting(false);
    }
    onOpenChange(newOpen);
  };

  // 确认编辑
  const handleConfirm = async () => {
    if (!editedMessage.trim() || editedMessage === originalMessage) {
      return;
    }

    setIsSubmitting(true);

    try {
      // 调用父组件传入的回调（传递 blockId 和 conversationId 用于 Fork API）
      await onConfirm?.(editedMessage, blockId, conversationId);

      // 关闭对话框
      onOpenChange(false);
    } finally {
      setIsSubmitting(false);
    }
  };

  const hasChanges = editedMessage !== originalMessage;
  const isValid = editedMessage.trim().length > 0;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-[32rem]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Pencil className="w-5 h-5" />
            {t("ai.unified_block.edit_title")}
          </DialogTitle>
          <DialogDescription>{t("ai.unified_block.edit_description")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* 原始消息 */}
          <div className="space-y-2">
            <label className="text-sm font-medium text-muted-foreground">{t("ai.unified_block.original_message")}</label>
            <div className="p-3 rounded-md bg-muted/50 text-sm text-muted-foreground">{originalMessage}</div>
          </div>

          {/* 编辑区域 */}
          <div className="space-y-2">
            <label htmlFor="edit-input" className="text-sm font-medium">
              {t("ai.unified_block.edited_message")}
            </label>
            <Textarea
              id="edit-input"
              value={editedMessage}
              onChange={(e) => setEditedMessage(e.target.value)}
              placeholder={t("ai.unified_block.edited_message")}
              rows={4}
              className={cn("resize-none", !hasChanges && "border-muted-foreground/50")}
              autoFocus
            />
          </div>

          {/* 警告信息 */}
          <div className="flex items-start gap-2 p-3 rounded-md bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800">
            <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-500 mt-0.5 shrink-0" />
            <p className="text-xs text-amber-800 dark:text-amber-200">{t("ai.unified_block.edit_warning")}</p>
          </div>
        </div>

        <DialogFooter className="gap-2">
          <Button type="button" variant="outline" onClick={() => handleOpenChange(false)} disabled={isSubmitting}>
            {t("ai.unified_block.cancel")}
          </Button>
          <Button type="button" onClick={handleConfirm} disabled={!hasChanges || !isValid || isSubmitting}>
            {isSubmitting ? t("states.processing") : t("ai.unified_block.create_branch")}
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
  const [blockId, setBlockId] = useState<bigint>(BigInt(0));
  const [conversationId, setConversationId] = useState(0);
  const [originalMessage, setOriginalMessage] = useState("");

  const openDialog = (id: bigint, convId: number, message: string) => {
    setBlockId(id);
    setConversationId(convId);
    setOriginalMessage(message);
    setOpen(true);
  };

  const closeDialog = () => {
    setOpen(false);
  };

  return {
    open,
    blockId,
    conversationId,
    originalMessage,
    openDialog,
    closeDialog,
    setOpen,
  };
}

import { Check, GitBranch, Trash2, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { type BlockBranch } from "@/types/block";

interface BranchSelectorProps {
  branches: BlockBranch[];
  currentPath: string;
  isOpen: boolean;
  onClose: () => void;
  onSwitchBranch: (path: string) => void;
  onDeleteBranch?: (path: string) => void;
  onCreateBranch?: () => void;
  className?: string;
}

/**
 * BranchSelector - Modal dialog for switching between conversation branches
 *
 * Displays a tree structure of available branches with options to:
 * - Switch to a different branch
 * - Delete a branch (with cascade option)
 * - Create a new branch
 */
export function BranchSelector({
  branches,
  currentPath,
  isOpen,
  onClose,
  onSwitchBranch,
  onDeleteBranch,
  onCreateBranch,
  className,
}: BranchSelectorProps) {
  const { t } = useTranslation();
  const [confirmDeletePath, setConfirmDeletePath] = React.useState<string | null>(null);

  const handleSwitchBranch = (path: string) => {
    onSwitchBranch(path);
    onClose();
  };

  const handleDeleteBranch = (path: string) => {
    if (onDeleteBranch) {
      onDeleteBranch(path);
      setConfirmDeletePath(null);
      onClose();
    }
  };

  // Recursive render of branch tree
  const renderBranch = (branch: BlockBranch, depth = 0) => {
    const isActive = branch.isActive;
    const hasChildren = branch.children && branch.children.length > 0;
    const isConfirming = confirmDeletePath === branch.branchPath;

    return (
      <div key={branch.branchPath} className="w-full">
        <div
          className={cn(
            "flex items-center gap-2 py-2 px-3 rounded-lg transition-colors",
            "group hover:bg-muted/50",
            isActive && "bg-primary/10 hover:bg-primary/20",
          )}
          style={{ marginLeft: `${depth * 16}px` }}
        >
          {/* Branch icon */}
          <GitBranch className={cn("w-4 h-4 flex-shrink-0", isActive ? "text-primary" : "text-muted-foreground")} />

          {/* Branch path */}
          <span className="flex-1 text-sm font-mono text-muted-foreground">{branch.branchPath || "/"}</span>

          {/* Active badge */}
          {isActive && <span className="text-xs text-primary font-medium">{t("chat.branches.current-branch")}</span>}

          {/* Actions */}
          <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
            {!isActive && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-7 w-7 p-0"
                onClick={() => handleSwitchBranch(branch.branchPath)}
                title={t("chat.branches.switch-to")}
              >
                <Check className="w-3.5 h-3.5" />
              </Button>
            )}

            {onDeleteBranch && !isActive && (
              <>
                {isConfirming ? (
                  <>
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      className="h-7 px-2 text-xs"
                      onClick={() => handleDeleteBranch(branch.branchPath)}
                    >
                      {t("common.confirm")}
                    </Button>
                    <Button type="button" variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => setConfirmDeletePath(null)}>
                      <X className="w-3.5 h-3.5" />
                    </Button>
                  </>
                ) : (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-7 w-7 p-0 text-red-500 hover:text-red-600 hover:bg-red-100 dark:hover:bg-red-950/30"
                    onClick={() => setConfirmDeletePath(branch.branchPath)}
                    title={t("chat.branches.delete-branch")}
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </Button>
                )}
              </>
            )}
          </div>
        </div>

        {/* Render children */}
        {hasChildren && <div className="mt-0.5">{branch.children!.map((child) => renderBranch(child, depth + 1))}</div>}
      </div>
    );
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className={cn("max-w-[28rem]", className)}>
        <DialogHeader>
          <DialogTitle className="flex items-center justify-between">
            <span>{t("chat.branches.title")}</span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-8"
              onClick={() => {
                onCreateBranch?.();
                onClose();
              }}
            >
              <GitBranch className="w-4 h-4 mr-1.5" />
              {t("chat.branches.create-branch")}
            </Button>
          </DialogTitle>
        </DialogHeader>

        <div className="mt-4">
          {branches.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">{t("chat.branches.no-branches")}</div>
          ) : (
            <div className="space-y-0.5 max-h-[400px] overflow-y-auto">{branches.map((branch) => renderBranch(branch))}</div>
          )}
        </div>

        {/* Current path info */}
        {currentPath && (
          <div className="mt-4 pt-4 border-t border-border/50">
            <div className="text-xs text-muted-foreground">
              <span className="font-medium">{t("chat.branches.current-branch")}:</span>{" "}
              <span className="font-mono">{currentPath || "/"}</span>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

// Import React
import React from "react";

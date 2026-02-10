import { useCallback, useState } from "react";
import { useNavigate } from "react-router-dom";
import { MemoList } from "@/components/Memo";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useDeleteMemo, useMemoFilters, useMemoSorting, useUpdateMemo } from "@/hooks";
import useCurrentUser from "@/hooks/useCurrentUser";
import { State } from "@/types/proto/api/v1/common_pb";
import type { Memo } from "@/types/proto/api/v1/memo_service_pb";

const Archived = () => {
  const user = useCurrentUser();
  const navigate = useNavigate();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [memoToDelete, setMemoToDelete] = useState<{ name: string; content: string } | null>(null);

  // Memo mutations
  const deleteMemo = useDeleteMemo();
  const updateMemo = useUpdateMemo();

  // Build filter using unified hook (no shortcuts or pinned filter)
  const memoFilter = useMemoFilters({
    creatorName: user?.name,
    includeShortcuts: false,
    includePinned: false,
  });

  // Get sorting logic using unified hook (pinned first, archived state)
  const { orderBy } = useMemoSorting({
    pinnedFirst: true,
    state: State.ARCHIVED,
  });

  // Handle memo actions
  const handleEdit = useCallback(
    (memo: Memo) => {
      navigate(`/m/${memo.name}`);
    },
    [navigate],
  );

  const handleDelete = useCallback((memo: Memo) => {
    setMemoToDelete({ name: memo.name, content: memo.content });
    setDeleteDialogOpen(true);
  }, []);

  const confirmDelete = useCallback(() => {
    if (memoToDelete) {
      deleteMemo.mutate(memoToDelete.name);
      setDeleteDialogOpen(false);
      setMemoToDelete(null);
    }
  }, [memoToDelete, deleteMemo]);

  const handleArchive = useCallback(
    (name: string, archived: boolean) => {
      updateMemo.mutate({
        update: { name, state: archived ? 2 : 0 },
        updateMask: ["state"],
      });
    },
    [updateMemo],
  );

  const handlePin = useCallback(
    (name: string, pinned: boolean) => {
      updateMemo.mutate({
        update: { name, pinned },
        updateMask: ["pinned"],
      });
    },
    [updateMemo],
  );

  const handleCopy = useCallback((content: string) => {
    navigator.clipboard.writeText(content);
  }, []);

  return (
    <>
      <MemoList
        state={State.ARCHIVED}
        orderBy={orderBy}
        filter={memoFilter}
        showCreator
        onEdit={handleEdit}
        onDelete={handleDelete}
        onArchive={handleArchive}
        onPin={handlePin}
        onCopy={handleCopy}
      />

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Memo?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete the memo.
              {memoToDelete?.content && (
                <div className="mt-2 p-2 bg-muted rounded text-sm">
                  {memoToDelete.content.slice(0, 100)}
                  {memoToDelete.content.length > 100 && "..."}
                </div>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={confirmDelete} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
};

export default Archived;

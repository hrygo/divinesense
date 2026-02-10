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
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";

const Explore = () => {
  const currentUser = useCurrentUser();
  const navigate = useNavigate();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [memoToDelete, setMemoToDelete] = useState<{ name: string; content: string } | null>(null);

  // Memo mutations
  const deleteMemo = useDeleteMemo();
  const updateMemo = useUpdateMemo();

  // Determine visibility filter based on authentication status
  // - Logged-in users: Can see PUBLIC and PROTECTED memos
  // - Visitors: Can only see PUBLIC memos
  const visibilities = currentUser ? [Visibility.PUBLIC, Visibility.PROTECTED] : [Visibility.PUBLIC];

  // Build filter using unified hook (no creator scoping for Explore)
  const memoFilter = useMemoFilters({
    includeShortcuts: false,
    includePinned: false,
    visibilities,
  });

  // Get sorting logic using unified hook (no pinned sorting)
  const { orderBy } = useMemoSorting({
    pinnedFirst: false,
    state: State.NORMAL,
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
        state={State.NORMAL}
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

export default Explore;

import { useCallback, useState } from "react";
import { useNavigate } from "react-router-dom";
import { HeroSection, MemoList } from "@/components/Memo";
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

const Home = () => {
  const user = useCurrentUser();
  const navigate = useNavigate();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [memoToDelete, setMemoToDelete] = useState<{ name: string; content: string } | null>(null);

  // Memo mutations
  const deleteMemo = useDeleteMemo();
  const updateMemo = useUpdateMemo();

  // Build filter using unified hook
  const memoFilter = useMemoFilters({
    creatorName: user?.name,
    includeShortcuts: true,
    includePinned: true,
  });

  // Get sorting logic using unified hook
  const { orderBy } = useMemoSorting({
    pinnedFirst: true,
    state: State.NORMAL,
  });

  // Handle create memo - open the fixed editor
  const handleCreateMemo = useCallback(() => {
    // Dispatch event to open editor in MemoLayout
    window.dispatchEvent(new CustomEvent("memo-editor-open"));
    // Focus on editor
    setTimeout(() => {
      const editor = document.querySelector<HTMLTextAreaElement>("textarea[placeholder*='thought'], textarea[placeholder*='想法']");
      editor?.focus();
    }, 100);
  }, []);

  // Handle search
  const handleSearch = useCallback(() => {
    navigate("/explore");
  }, [navigate]);

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
        update: { name, state: archived ? 2 : 0 }, // State.ARCHIVED = 2, State.NORMAL = 0
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
    <div className="w-full min-h-full bg-background text-foreground">
      {/* Hero Section */}
      <HeroSection onCreateMemo={handleCreateMemo} onSearch={handleSearch} />

      {/* Memo List - Single Column Timeline */}
      <MemoList
        orderBy={orderBy}
        filter={memoFilter}
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
    </div>
  );
};

export default Home;

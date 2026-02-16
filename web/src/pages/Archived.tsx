import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { MemoListV3 } from "@/components/Memo";
import { useMemoFilters, useMemoSorting } from "@/hooks";
import useCurrentUser from "@/hooks/useCurrentUser";
import { State } from "@/types/proto/api/v1/common_pb";

const Archived = () => {
  const user = useCurrentUser();
  const navigate = useNavigate();

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

  // Handle memo edit - other actions are handled by MemoBlock
  const handleEdit = useCallback(
    (memoName: string) => {
      const memoId = memoName.split("/").pop() || memoName;
      navigate(`/m/${memoId}`);
    },
    [navigate],
  );

  return (
    <div className="w-full min-h-full bg-background text-foreground">
      {/* Unified width container - matches AIChat responsive width */}
      <div className="mx-auto max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl px-4 sm:px-6 pb-8">
        <MemoListV3 state={State.ARCHIVED} orderBy={orderBy} filter={memoFilter} onEdit={handleEdit} />
      </div>
    </div>
  );
};

export default Archived;

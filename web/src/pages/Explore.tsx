import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { MemoListV3 } from "@/components/Memo";
import { ExploreHeroSection } from "@/components/Memo/ExploreHeroSection";
import { useMemoFilters, useMemoSorting } from "@/hooks";
import useCurrentUser from "@/hooks/useCurrentUser";
import { State } from "@/types/proto/api/v1/common_pb";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";

const Explore = () => {
  const currentUser = useCurrentUser();
  const navigate = useNavigate();

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

  // Handle memo edit - other actions are handled by MemoBlock
  const handleEdit = useCallback(
    (memoName: string) => {
      const memoId = memoName.split("/").pop() || memoName;
      navigate(`/m/${memoId}`);
    },
    [navigate],
  );

  return (
    <div className="w-full min-h-full text-foreground">
      {/* Unified width container - matches Home page */}
      <div className="mx-auto max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl px-4 sm:px-6 pb-8">
        {/* Hero Section - 探索页首屏 */}
        <ExploreHeroSection />

        {/* Memo List - filtered by visibility */}
        <MemoListV3 state={State.NORMAL} orderBy={orderBy} filter={memoFilter} onEdit={handleEdit} />
      </div>
    </div>
  );
};

export default Explore;

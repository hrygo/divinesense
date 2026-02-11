import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { HeroSection, MemoList } from "@/components/Memo";
import { FixedEditor } from "@/components/Memo/FixedEditor";
import { useMemoFilters, useMemoSorting } from "@/hooks";
import useCurrentUser from "@/hooks/useCurrentUser";
import { State } from "@/types/proto/api/v1/common_pb";
import type { Memo } from "@/types/proto/api/v1/memo_service_pb";

const Home = () => {
  const { t } = useTranslation();
  const user = useCurrentUser();
  const navigate = useNavigate();

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

  // Handle memo edit
  const handleEdit = useCallback(
    (memo: Memo) => {
      const memoId = memo.name.split("/").pop() || memo.name;
      navigate(`/m/${memoId}`);
    },
    [navigate],
  );

  return (
    <div className="w-full min-h-full text-foreground">
      {/* Unified width container for all sections - matches AIChat responsive width */}
      <div className="mx-auto max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl px-4 sm:px-6 pb-8">
        {/* Hero Section with integrated intelligent search */}
        <HeroSection />

        {/* Memo List - filtered by search query */}
        <MemoList orderBy={orderBy} filter={memoFilter} onEdit={handleEdit} />
      </div>

      {/* Fixed Editor - outside container to handle its own width */}
      <FixedEditor placeholder={t("editor.any-thoughts") || t("editor.placeholder")} />
    </div>
  );
};

export default Home;

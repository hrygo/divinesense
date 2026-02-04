import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertCircle, Brain, CheckCircle2, ChevronRight, Clock, RefreshCw, Sparkles, Target, Trophy } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { aiServiceClient } from "@/connect";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { ReviewQuality } from "@/types/proto/api/v1/ai_service_pb";
import { useTranslate } from "@/utils/i18n";

const Review = () => {
  const t = useTranslate();
  const sm = useMediaQuery("sm");
  const queryClient = useQueryClient();
  const [currentIndex, setCurrentIndex] = useState(0);
  const [showAnswer, setShowAnswer] = useState(false);

  // Fetch due reviews
  const {
    data: dueReviews,
    isLoading: isLoadingReviews,
    isError: isErrorReviews,
    refetch: refetchReviews,
  } = useQuery({
    queryKey: ["reviews", "due"],
    queryFn: async () => {
      const response = await aiServiceClient.getDueReviews({ limit: 20 });
      return response;
    },
  });

  // Fetch review stats
  const {
    data: stats,
    isLoading: isLoadingStats,
    isError: isErrorStats,
    refetch: refetchStats,
  } = useQuery({
    queryKey: ["reviews", "stats"],
    queryFn: async () => {
      const response = await aiServiceClient.getReviewStats({});
      return response;
    },
  });

  // Record review mutation
  const recordReviewMutation = useMutation({
    mutationFn: async ({ memoUid, quality }: { memoUid: string; quality: ReviewQuality }) => {
      await aiServiceClient.recordReview({ memoUid, quality });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["reviews"] });
      setShowAnswer(false);
      if (dueReviews && currentIndex < dueReviews.items.length - 1) {
        setCurrentIndex((prev) => prev + 1);
      }
    },
  });

  const currentItem = dueReviews?.items[currentIndex];
  const totalDue = dueReviews?.totalDue || 0;
  const progress = totalDue > 0 ? ((currentIndex + 1) / totalDue) * 100 : 0;

  const handleQualityClick = (quality: ReviewQuality) => {
    if (!currentItem) return;
    recordReviewMutation.mutate({ memoUid: currentItem.memoUid, quality });
  };

  const qualityButtons = [
    {
      quality: ReviewQuality.AGAIN,
      label: t("review.quality.again"),
      color: "bg-red-500 hover:bg-red-600",
      desc: t("review.quality.again-desc"),
    },
    {
      quality: ReviewQuality.HARD,
      label: t("review.quality.hard"),
      color: "bg-orange-500 hover:bg-orange-600",
      desc: t("review.quality.hard-desc"),
    },
    {
      quality: ReviewQuality.GOOD,
      label: t("review.quality.good"),
      color: "bg-green-500 hover:bg-green-600",
      desc: t("review.quality.good-desc"),
    },
    {
      quality: ReviewQuality.EASY,
      label: t("review.quality.easy"),
      color: "bg-blue-500 hover:bg-blue-600",
      desc: t("review.quality.easy-desc"),
    },
  ];

  if (isLoadingReviews || isLoadingStats) {
    return (
      <div className="w-full h-full flex items-center justify-center">
        <RefreshCw className="w-8 h-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (isErrorReviews || isErrorStats) {
    return (
      <div className="w-full h-full flex flex-col items-center justify-center gap-4">
        <AlertCircle className="w-12 h-12 text-destructive" />
        <p className="text-muted-foreground">{t("review.load-error")}</p>
        <Button
          onClick={() => {
            refetchReviews();
            refetchStats();
          }}
        >
          {t("common.retry") || "Retry"}
        </Button>
      </div>
    );
  }

  return (
    <div className="w-full space-y-6">
      {/* Header - Hidden on Mobile */}
      {sm && (
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-primary/10 rounded-lg">
              <Brain className="w-6 h-6 text-primary" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">{t("review.title")}</h1>
              <p className="text-sm text-muted-foreground">{t("review.subtitle")}</p>
            </div>
          </div>
        </div>
      )}

      {/* Stats Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="p-4 rounded-lg border border-border bg-card">
          <div className="flex items-center gap-2">
            <Target className="w-4 h-4 text-orange-500" />
            <span className="text-sm text-muted-foreground">{t("review.stats.due-today")}</span>
          </div>
          <p className="text-2xl font-bold mt-1">{stats?.dueToday || 0}</p>
        </div>
        <div className="p-4 rounded-lg border border-border bg-card">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="w-4 h-4 text-green-500" />
            <span className="text-sm text-muted-foreground">{t("review.stats.reviewed-today")}</span>
          </div>
          <p className="text-2xl font-bold mt-1">{stats?.reviewedToday || 0}</p>
        </div>
        <div className="p-4 rounded-lg border border-border bg-card">
          <div className="flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-blue-500" />
            <span className="text-sm text-muted-foreground">{t("review.stats.new-memos")}</span>
          </div>
          <p className="text-2xl font-bold mt-1">{stats?.newMemos || 0}</p>
        </div>
        <div className="p-4 rounded-lg border border-border bg-card">
          <div className="flex items-center gap-2">
            <Trophy className="w-4 h-4 text-yellow-500" />
            <span className="text-sm text-muted-foreground">{t("review.stats.mastered")}</span>
          </div>
          <p className="text-2xl font-bold mt-1">{stats?.masteredMemos || 0}</p>
        </div>
      </div>

      {/* Review Card */}
      {currentItem ? (
        <div className="rounded-lg border border-border bg-card overflow-hidden">
          <div className="p-4 border-b border-border">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold">{currentItem.title || t("review.untitled")}</h3>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Clock className="w-4 h-4" />
                <span>{t("review.review-count", { count: currentItem.reviewCount })}</span>
              </div>
            </div>
            <div className="flex items-center gap-2 mt-2">
              {currentItem.tags?.map((tag) => (
                <span key={tag} className="px-2 py-0.5 bg-muted rounded-full text-xs">
                  #{tag}
                </span>
              ))}
            </div>
          </div>
          <div className="p-4 space-y-4">
            {/* Progress */}
            <div className="space-y-1">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">{t("review.progress")}</span>
                <span className="font-medium">
                  {currentIndex + 1} / {totalDue}
                </span>
              </div>
              <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
                <div className="h-full bg-primary transition-all duration-300" style={{ width: `${progress}%` }} />
              </div>
            </div>

            {/* Content Preview */}
            <div className={cn("p-4 bg-muted/50 rounded-lg min-h-[120px] transition-all", !showAnswer && "blur-sm select-none")}>
              <p className="text-sm whitespace-pre-wrap">{currentItem.snippet}</p>
            </div>

            {/* Show Answer / Quality Buttons */}
            {!showAnswer ? (
              <Button className="w-full" size="lg" onClick={() => setShowAnswer(true)}>
                {t("review.show-answer")}
              </Button>
            ) : (
              <div className="space-y-3">
                <p className="text-sm text-center text-muted-foreground">{t("review.how-well")}</p>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                  {qualityButtons.map(({ quality, label, color, desc }) => (
                    <Button
                      key={quality}
                      variant="outline"
                      className={cn("flex-col h-auto py-3 gap-1 text-white border-0", color)}
                      onClick={() => handleQualityClick(quality)}
                      disabled={recordReviewMutation.isPending}
                    >
                      <span className="font-medium">{label}</span>
                      <span className="text-xs opacity-80">{desc}</span>
                    </Button>
                  ))}
                </div>
                <Link to={`/memos/${currentItem.memoUid}`} className="block">
                  <Button variant="ghost" className="w-full gap-2">
                    {t("review.view-memo")}
                    <ChevronRight className="w-4 h-4" />
                  </Button>
                </Link>
              </div>
            )}
          </div>
        </div>
      ) : (
        <div className="rounded-lg border border-border bg-card p-12 text-center">
          <Trophy className="w-12 h-12 mx-auto text-yellow-500 mb-4" />
          <h3 className="text-xl font-semibold mb-2">{t("review.all-done")}</h3>
          <p className="text-muted-foreground">{t("review.all-done-desc")}</p>
        </div>
      )}
    </div>
  );
};

export default Review;

import { create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";
import { useQueryClient } from "@tanstack/react-query";
import { ArrowUpLeftFromCircleIcon, MessageCircleIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "react-hot-toast";
import { Link, useLocation, useParams } from "react-router-dom";
import { CommentEditor } from "@/components/CommentEditor";
import MemoRelatedList from "@/components/MemoRelatedList";
import MemoView from "@/components/MemoView/MemoView";
import { Button } from "@/components/ui/button";
import { memoNamePrefix } from "@/helpers/resource-names";
import useCurrentUser from "@/hooks/useCurrentUser";
import { memoKeys, useCreateMemo, useMemo, useMemoComments } from "@/hooks/useMemoQueries";
import useNavigateTo from "@/hooks/useNavigateTo";
import { MemoSchema, Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";

const MemoDetail = () => {
  const t = useTranslate();
  const params = useParams();
  const navigateTo = useNavigateTo();
  const { state: locationState } = useLocation();
  const currentUser = useCurrentUser();
  const queryClient = useQueryClient();
  const uid = params.uid;
  const memoName = `${memoNamePrefix}${uid}`;
  const [showCommentEditor, setShowCommentEditor] = useState(false);

  // Create memo mutation for comments
  const createMemoMutation = useCreateMemo();

  // Fetch main memo with React Query
  const { data: memo, error, isLoading } = useMemo(memoName, { enabled: !!memoName });

  // Handle errors
  if (error) {
    toast.error((error as ConnectError).message);
    navigateTo("/403");
  }

  // Fetch parent memo if exists
  const { data: parentMemo } = useMemo(memo?.parent || "", {
    enabled: !!memo?.parent,
  });

  // Fetch all comments for this memo in a single query
  const { data: commentsResponse } = useMemoComments(memoName, {
    enabled: !!memo,
  });
  const comments = commentsResponse?.memos || [];

  const showCreateCommentButton = currentUser && !showCommentEditor;

  if (isLoading || !memo) {
    return null;
  }

  const handleShowCommentEditor = () => {
    setShowCommentEditor(true);
  };

  const handleCommentCreated = async (content: string) => {
    // Create comment memo
    const commentMemo = create(MemoSchema, {
      content,
      visibility: Visibility.PRIVATE,
      parent: memo.name,
    });

    await createMemoMutation.mutateAsync(commentMemo);

    // Invalidate comments query to refresh the list
    queryClient.invalidateQueries({ queryKey: memoKeys.comments(memo.name) });

    setShowCommentEditor(false);
  };

  return (
    <div className="w-full min-h-full text-foreground">
      {/* Unified width container - matches Home/Explore pages */}
      <div className="mx-auto max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl px-4 sm:px-6 pb-8">
        {/* Parent memo link */}
        {parentMemo && (
          <div className="w-auto inline-block mb-2">
            <Link
              className="px-3 py-1 border border-border rounded-lg max-w-xs w-auto text-sm flex flex-row justify-start items-center flex-nowrap text-muted-foreground hover:shadow hover:opacity-80"
              to={`/${parentMemo.name}`}
              state={locationState}
              viewTransition
            >
              <ArrowUpLeftFromCircleIcon className="w-4 h-auto shrink-0 opacity-60 mr-2" />
              <span className="truncate">{parentMemo.content}</span>
            </Link>
          </div>
        )}

        {/* Main memo */}
        <MemoView
          key={`${memo.name}-${memo.displayTime}`}
          className="shadow hover:shadow-sm transition-all"
          memo={memo}
          compact={false}
          parentPage={locationState?.from}
          showCreator
          showVisibility
          showPinned
          showNsfwContent
        />

        {/* Comments section */}
        <div className="pt-6 w-full">
          <h2 id="comments" className="sr-only">
            {t("memo.comment.self")}
          </h2>
          <div className="w-full flex flex-col gap-y-2">
            {comments.length === 0 ? (
              showCreateCommentButton && (
                <div className="w-full flex flex-row justify-center items-center py-4">
                  <Button variant="ghost" onClick={handleShowCommentEditor}>
                    <span className="text-muted-foreground">{t("memo.comment.write-a-comment")}</span>
                    <MessageCircleIcon className="ml-2 w-5 h-auto text-muted-foreground" />
                  </Button>
                </div>
              )
            ) : (
              <>
                <div className="w-full flex flex-row justify-between items-center h-8 pl-3 mb-2">
                  <div className="flex flex-row justify-start items-center">
                    <MessageCircleIcon className="w-5 h-auto text-muted-foreground mr-1" />
                    <span className="text-muted-foreground text-sm">{t("memo.comment.self")}</span>
                    <span className="text-muted-foreground text-sm ml-1">({comments.length})</span>
                  </div>
                  {showCreateCommentButton && (
                    <Button variant="ghost" className="text-muted-foreground" onClick={handleShowCommentEditor}>
                      {t("memo.comment.write-a-comment")}
                    </Button>
                  )}
                </div>
                {comments.map((comment) => (
                  <MemoView
                    key={`${comment.name}-${comment.displayTime}`}
                    memo={comment}
                    parentPage={locationState?.from}
                    showCreator
                    compact
                  />
                ))}
              </>
            )}
            {showCommentEditor && (
              <div className="w-full mt-4">
                <CommentEditor
                  placeholder={t("editor.add-your-comment-here")}
                  autoFocus
                  onSend={handleCommentCreated}
                  onCancel={() => setShowCommentEditor(false)}
                />
              </div>
            )}
          </div>
        </div>

        {/* Related memos */}
        <MemoRelatedList memoName={memoName} />
      </div>
    </div>
  );
};

export default MemoDetail;

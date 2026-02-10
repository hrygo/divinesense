import { timestampDate } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { PaperclipIcon, SearchIcon, Trash } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "react-hot-toast";
import { Link } from "react-router-dom";
import AttachmentIcon from "@/components/AttachmentIcon";
import ConfirmDialog from "@/components/ConfirmDialog";
import Empty from "@/components/Empty";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { attachmentServiceClient } from "@/connect";
import { useDeleteAttachment } from "@/hooks/useAttachmentQueries";
import useDialog from "@/hooks/useDialog";
import useLoading from "@/hooks/useLoading";
import useMediaQuery from "@/hooks/useMediaQuery";
import i18n from "@/i18n";
import { handleError } from "@/lib/error";
import type { Attachment } from "@/types/proto/api/v1/attachment_service_pb";
import { useTranslate } from "@/utils/i18n";

const PAGE_SIZE = 50;

/**
 * Group attachments by date (YYYY-MM format)
 */
const groupAttachmentsByDate = (attachments: Attachment[]): Map<string, Attachment[]> => {
  const grouped = new Map<string, Attachment[]>();
  const sorted = [...attachments].sort((a, b) => {
    const aTime = a.createTime ? timestampDate(a.createTime) : undefined;
    const bTime = b.createTime ? timestampDate(b.createTime) : undefined;
    return dayjs(bTime).unix() - dayjs(aTime).unix();
  });

  for (const attachment of sorted) {
    const createTime = attachment.createTime ? timestampDate(attachment.createTime) : undefined;
    const monthKey = dayjs(createTime).format("YYYY-MM");
    const group = grouped.get(monthKey) ?? [];
    group.push(attachment);
    grouped.set(monthKey, group);
  }

  return grouped;
};

/**
 * Filter attachments by search query
 */
const filterAttachments = (attachments: Attachment[], searchQuery: string): Attachment[] => {
  if (!searchQuery.trim()) return attachments;
  const query = searchQuery.toLowerCase();
  return attachments.filter((attachment) => attachment.filename.toLowerCase().includes(query));
};

/**
 * Attachment Item Component
 * Uses unified card styling matching MemoBlock
 */
interface AttachmentItemProps {
  attachment: Attachment;
}

const AttachmentItem = ({ attachment }: AttachmentItemProps) => (
  <div className="w-24 sm:w-32 h-auto flex flex-col justify-start items-start">
    <div className="w-24 h-24 flex justify-center items-center sm:w-32 sm:h-32 border border-border overflow-hidden rounded-lg cursor-pointer hover:shadow-sm hover:opacity-80 transition-all">
      <AttachmentIcon attachment={attachment} strokeWidth={0.5} />
    </div>
    <div className="w-full max-w-full flex flex-row justify-between items-center mt-1 px-1">
      <p className="text-xs shrink text-muted-foreground truncate" title={attachment.filename}>
        {attachment.filename}
      </p>
      {attachment.memo && (
        <Link to={`/${attachment.memo}`} className="text-primary hover:opacity-80 transition-opacity shrink-0 ml-1" aria-label="View memo">
          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
            />
          </svg>
        </Link>
      )}
    </div>
  </div>
);

/**
 * Attachments Page
 * Unified design matching MemoBlock and Chat styles
 */
const Attachments = () => {
  const t = useTranslate();
  const sm = useMediaQuery("sm");
  const loadingState = useLoading();
  const deleteUnusedAttachmentsDialog = useDialog();
  const { mutateAsync: deleteAttachment } = useDeleteAttachment();

  const [searchQuery, setSearchQuery] = useState("");
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [nextPageToken, setNextPageToken] = useState("");
  const [isLoadingMore, setIsLoadingMore] = useState(false);

  // Memoized computed values
  const filteredAttachments = useMemo(() => filterAttachments(attachments, searchQuery), [attachments, searchQuery]);
  const usedAttachments = useMemo(() => filteredAttachments.filter((attachment) => attachment.memo), [filteredAttachments]);
  const unusedAttachments = useMemo(() => filteredAttachments.filter((attachment) => !attachment.memo), [filteredAttachments]);
  const groupedAttachments = useMemo(() => groupAttachmentsByDate(usedAttachments), [usedAttachments]);

  // Fetch initial attachments
  useEffect(() => {
    const fetchInitialAttachments = async () => {
      try {
        const { attachments: fetchedAttachments, nextPageToken: token } = await attachmentServiceClient.listAttachments({
          pageSize: PAGE_SIZE,
        });
        setAttachments(fetchedAttachments);
        setNextPageToken(token ?? "");
      } catch (error) {
        handleError(error, toast.error, {
          context: "Failed to fetch attachments",
          fallbackMessage: "Failed to load attachments. Please try again.",
        });
      } finally {
        loadingState.setFinish();
      }
    };

    fetchInitialAttachments();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Load more attachments with pagination
  const handleLoadMore = useCallback(async () => {
    if (!nextPageToken || isLoadingMore) return;

    setIsLoadingMore(true);
    try {
      const { attachments: fetchedAttachments, nextPageToken: newPageToken } = await attachmentServiceClient.listAttachments({
        pageSize: PAGE_SIZE,
        pageToken: nextPageToken,
      });
      setAttachments((prev) => [...prev, ...fetchedAttachments]);
      setNextPageToken(newPageToken ?? "");
    } catch (error) {
      handleError(error, toast.error, {
        context: "Failed to load more attachments",
        fallbackMessage: "Failed to load more attachments. Please try again.",
      });
    } finally {
      setIsLoadingMore(false);
    }
  }, [nextPageToken, isLoadingMore]);

  // Refetch all attachments from the beginning
  const handleRefetch = useCallback(async () => {
    try {
      loadingState.setLoading();
      const { attachments: fetchedAttachments, nextPageToken: token } = await attachmentServiceClient.listAttachments({
        pageSize: PAGE_SIZE,
      });
      setAttachments(fetchedAttachments);
      setNextPageToken(token ?? "");
      loadingState.setFinish();
    } catch (error) {
      handleError(error, toast.error, {
        context: "Failed to refetch attachments",
        fallbackMessage: "Failed to refresh attachments. Please try again.",
        onError: () => loadingState.setError(),
      });
    }
  }, [loadingState]);

  // Delete all unused attachments
  const handleDeleteUnusedAttachments = useCallback(async () => {
    try {
      let allUnusedAttachments: Attachment[] = [];
      let nextPageToken = "";
      do {
        const response = await attachmentServiceClient.listAttachments({
          pageSize: 1000,
          pageToken: nextPageToken,
          filter: "memo_id == null",
        });
        allUnusedAttachments = [...allUnusedAttachments, ...response.attachments];
        nextPageToken = response.nextPageToken;
      } while (nextPageToken);

      await Promise.all(allUnusedAttachments.map((attachment) => deleteAttachment(attachment.name)));
      toast.success(t("resource.delete-all-unused-success"));
    } catch (error) {
      handleError(error, toast.error, {
        context: "Failed to delete unused attachments",
        fallbackMessage: t("resource.delete-all-unused-error"),
      });
    } finally {
      await handleRefetch();
    }
  }, [t, handleRefetch, deleteAttachment]);

  // Handle search input change
  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchQuery(e.target.value);
  }, []);

  return (
    <>
      {/* Header Card - Unified style */}
      <div className="border border-border rounded-lg bg-card shadow-sm overflow-hidden">
        {/* Desktop Header */}
        {sm && (
          <div className="relative w-full flex flex-row justify-between items-center px-4 py-3 border-b border-border/50">
            <div className="flex items-center gap-2">
              <PaperclipIcon className="w-5 h-5 text-muted-foreground" />
              <h1 className="text-lg font-medium">{t("common.attachments")}</h1>
            </div>
            <div className="relative max-w-48">
              <SearchIcon className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input className="pl-9 h-9" placeholder={t("common.search")} value={searchQuery} onChange={handleSearchChange} />
            </div>
          </div>
        )}

        {/* Content */}
        <div className="w-full flex flex-col justify-start items-start py-6">
          {loadingState.isLoading ? (
            <div className="w-full h-32 flex flex-col justify-center items-center">
              <p className="w-full text-center text-base my-6">{t("resource.fetching-data")}</p>
            </div>
          ) : (
            <>
              {filteredAttachments.length === 0 ? (
                <div className="w-full mt-8 mb-8 flex flex-col justify-center items-center">
                  <Empty />
                  <p className="mt-4 text-muted-foreground">{t("message.no-data")}</p>
                </div>
              ) : (
                <>
                  <div className="w-full h-auto flex flex-col justify-start items-start gap-y-8">
                    {Array.from(groupedAttachments.entries()).map(([monthStr, monthAttachments]) => (
                      <div key={monthStr} className="w-full flex flex-row justify-start items-start">
                        {/* Date Label */}
                        <div className="w-16 sm:w-24 pt-4 sm:pl-4 flex flex-col justify-start items-start shrink-0">
                          <span className="text-sm text-muted-foreground/60">{dayjs(monthStr).year()}</span>
                          <span className="font-medium text-xl text-foreground">
                            {dayjs(monthStr).toDate().toLocaleString(i18n.language, { month: "short" })}
                          </span>
                        </div>
                        {/* Attachments Grid */}
                        <div className="w-full max-w-[calc(100%-4rem)] sm:max-w-[calc(100%-6rem)] flex flex-row justify-start items-start gap-4 flex-wrap">
                          {monthAttachments.map((attachment) => (
                            <AttachmentItem key={attachment.name} attachment={attachment} />
                          ))}
                        </div>
                      </div>
                    ))}

                    {/* Unused Attachments Section */}
                    {unusedAttachments.length > 0 && (
                      <>
                        <Separator />
                        <div className="w-full flex flex-row justify-start items-start">
                          <div className="w-16 sm:w-24 sm:pl-4 flex flex-col justify-start items-start shrink-0"></div>
                          <div className="w-full max-w-[calc(100%-4rem)] sm:max-w-[calc(100%-6rem)] flex flex-col gap-4">
                            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
                              <div className="flex flex-row items-center gap-2">
                                <span className="text-muted-foreground">{t("resource.unused-resources")}</span>
                                <span className="text-muted-foreground/70">({unusedAttachments.length})</span>
                              </div>
                              <Button variant="destructive" size="sm" onClick={() => deleteUnusedAttachmentsDialog.open()}>
                                <Trash className="w-4 h-4 mr-1" />
                                {t("resource.delete-all-unused")}
                              </Button>
                            </div>
                            <div className="flex flex-row justify-start items-start gap-4 flex-wrap">
                              {unusedAttachments.map((attachment) => (
                                <AttachmentItem key={attachment.name} attachment={attachment} />
                              ))}
                            </div>
                          </div>
                        </div>
                      </>
                    )}
                  </div>

                  {/* Load More Button */}
                  {nextPageToken && (
                    <div className="w-full flex flex-row justify-center items-center mt-4">
                      <Button variant="outline" size="sm" onClick={handleLoadMore} disabled={isLoadingMore}>
                        {isLoadingMore ? t("resource.fetching-data") : t("memo.load-more")}
                      </Button>
                    </div>
                  )}
                </>
              )}
            </>
          )}
        </div>
      </div>

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        open={deleteUnusedAttachmentsDialog.isOpen}
        onOpenChange={deleteUnusedAttachmentsDialog.setOpen}
        title={t("resource.delete-all-unused-confirm")}
        confirmLabel={t("common.delete")}
        cancelLabel={t("common.cancel")}
        onConfirm={handleDeleteUnusedAttachments}
        confirmVariant="destructive"
      />
    </>
  );
};

export default Attachments;

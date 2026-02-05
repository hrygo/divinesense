import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef } from "react";
import { aiServiceClient } from "@/connect";
import { ParrotAgentType, parrotToProtoAgentType } from "@/types/parrot";
import {
  ChatRequestSchema,
  DetectDuplicatesRequestSchema,
  GetKnowledgeGraphRequestSchema,
  GetRelatedMemosRequestSchema,
  LinkMemosRequestSchema,
  MergeMemosRequestSchema,
  SemanticSearchRequestSchema,
  SuggestTagsRequestSchema,
} from "@/types/proto/api/v1/ai_service_pb";
// Phase 4: Import blockKeys for invalidating blocks on updates
import { blockKeys } from "./useBlockQueries";

// Event metadata types for Geek/Evolution mode observability
interface EventMetadata {
  durationMs?: number;
  totalDurationMs?: number;
  toolName?: string;
  toolId?: string;
  status?: string;
  errorMsg?: string;
  inputTokens?: number;
  outputTokens?: number;
  cacheWriteTokens?: number;
  cacheReadTokens?: number;
  inputSummary?: string;
  outputSummary?: string;
  filePath?: string;
  lineCount?: number;
}

// Block summary for Geek/Evolution modes (per-block statistics, not session-level)
interface BlockSummary {
  sessionId?: string;
  totalDurationMs?: number;
  thinkingDurationMs?: number;
  toolDurationMs?: number;
  generationDurationMs?: number;
  totalInputTokens?: number;
  totalOutputTokens?: number;
  totalCacheWriteTokens?: number;
  totalCacheReadTokens?: number;
  toolCallCount?: number;
  toolsUsed?: string[];
  filesModified?: number;
  filePaths?: string[];
  status?: string;
  errorMsg?: string;
}

// Safe conversion from protobuf bigint to JavaScript number
// Protobuf int64 becomes bigint in TypeScript, which needs safe conversion
const safeBigintToNumber = (value: bigint | undefined): number | undefined => {
  if (value === undefined) return undefined;
  // Check if value is within safe integer range
  const maxSafe = BigInt(Number.MAX_SAFE_INTEGER);
  const minSafe = BigInt(Number.MIN_SAFE_INTEGER);
  if (value > maxSafe || value < minSafe) {
    if (import.meta.env.DEV) {
      console.warn("[AI Chat] Duration value exceeds safe integer range", { value: value.toString() });
    }
    // Return max safe value as fallback
    return value > maxSafe ? Number.MAX_SAFE_INTEGER : Number.MIN_SAFE_INTEGER;
  }
  return Number(value);
};

// Constants for AI chat
const STREAM_TIMEOUT_MS = 10 * 60 * 1000; // 10 minutes
const SEMANTIC_SEARCH_LIMIT = 10; // Default search results limit
const STALE_TIME_SHORT_MS = 60 * 1000; // 1 minute
const STALE_TIME_LONG_MS = 5 * 60 * 1000; // 5 minutes
const EVENT_DATA_PREVIEW_LENGTH = 100; // Preview length for event data

// Query keys factory for consistent cache management
export const aiKeys = {
  all: ["ai"] as const,
  search: () => [...aiKeys.all, "search"] as const,
  searchQuery: (query: string) => [...aiKeys.search(), query] as const,
  related: (name: string) => [...aiKeys.all, "related", name] as const,
  knowledgeGraph: (filter: { tags: string[]; minImportance: number; clusters: number[] }) =>
    [...aiKeys.all, "knowledgeGraph", filter] as const,
};

/**
 * useSemanticSearch performs semantic search on memos.
 * @param query - Search query string
 * @param options.enabled - Whether the query is enabled
 */
export function useSemanticSearch(query: string, options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: aiKeys.searchQuery(query),
    queryFn: async () => {
      const request = create(SemanticSearchRequestSchema, {
        query,
        limit: SEMANTIC_SEARCH_LIMIT,
      });
      return await aiServiceClient.semanticSearch(request);
    },
    enabled: (options.enabled ?? true) && query.length > 2,
    staleTime: STALE_TIME_SHORT_MS, // 1 minute
  });
}

/**
 * useSuggestTags suggests tags for memo content using AI.
 */
export function useSuggestTags() {
  return useMutation({
    mutationFn: async (params: { content: string; limit?: number }) => {
      const request = create(SuggestTagsRequestSchema, {
        content: params.content,
        limit: params.limit ?? 5,
      });
      const response = await aiServiceClient.suggestTags(request);
      return response.tags;
    },
  });
}

/**
 * useRelatedMemos finds memos related to a specific memo.
 * @param name - Memo name in format "memos/{uid}"
 * @param options.enabled - Whether the query is enabled
 * @param options.limit - Maximum number of related memos to return
 */
export function useRelatedMemos(name: string, options: { enabled?: boolean; limit?: number } = {}) {
  return useQuery({
    queryKey: aiKeys.related(name),
    queryFn: async () => {
      const request = create(GetRelatedMemosRequestSchema, {
        name,
        limit: options.limit ?? 5,
      });
      return await aiServiceClient.getRelatedMemos(request);
    },
    enabled: (options.enabled ?? true) && !!name && name.startsWith("memos/"),
    staleTime: STALE_TIME_LONG_MS, // 5 minutes
  });
}

/**
 * useChat streams a chat response using AI agents.
 * Uses Connect RPC streaming to receive responses in real-time.
 *
 * @returns An object with stream function and callbacks
 */
export function useChat() {
  const queryClient = useQueryClient();
  const abortControllerRef = useRef<AbortController | null>(null);
  const timeoutIdRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  return {
    /**
     * Stream chat with memos as context.
     * @param params - Chat parameters
     * @param callbacks - Optional callbacks for streaming events
     * @returns A promise that resolves when streaming completes
     */
    stream: async (
      params: {
        message: string;
        history?: string[];
        agentType?: ParrotAgentType;
        userTimezone?: string;
        conversationId?: number;
        geekMode?: boolean;
        evolutionMode?: boolean;
      },
      callbacks?: {
        onContent?: (content: string) => void;
        onSources?: (sources: string[]) => void;
        onDone?: () => void;
        onError?: (error: Error) => void;
        onScheduleIntent?: (intent: { detected: boolean; scheduleDescription: string }) => void;
        onScheduleQueryResult?: (result: {
          detected: boolean;
          schedules: Array<{
            uid: string;
            title: string;
            startTs: bigint;
            endTs: bigint;
            allDay: boolean;
            location: string;
            recurrenceRule: string;
            status: string;
          }>;
          timeRangeDescription: string;
          queryType: string;
        }) => void;
        // Parrot-specific callbacks
        onThinking?: (message: string) => void;
        onToolUse?: (toolName: string, meta?: EventMetadata) => void;
        onToolResult?: (result: string, meta?: EventMetadata) => void;
        onMemoQueryResult?: (result: {
          memos: Array<{ uid: string; content: string; score: number }>;
          query: string;
          count: number;
        }) => void;
        // Observability callbacks (Geek/Evolution modes)
        onBlockSummary?: (summary: BlockSummary) => void;
      },
    ) => {
      const request = create(ChatRequestSchema, {
        message: params.message,
        history: params.history ?? [],
        agentType: params.agentType !== undefined ? parrotToProtoAgentType(params.agentType) : undefined,
        userTimezone: params.userTimezone,
        conversationId: params.conversationId,
        geekMode: params.geekMode ?? false,
        evolutionMode: params.evolutionMode ?? false,
        deviceContext: JSON.stringify({
          userAgent: navigator.userAgent,
          isMobile: /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent),
          screenWidth: window.screen.width,
          screenHeight: window.screen.height,
          windowWidth: window.innerWidth,
          windowHeight: window.innerHeight,
          language: navigator.language,
          platform: (navigator as Navigator).platform || "unknown",
        }),
      });

      // WORKAROUND: Manually set evolutionMode if create() didn't include it
      // Root cause: @bufbuild/protobuf create() omits default bool values (false)
      // in JSON serialization, but backend expects explicit false for mode routing.
      // This is a known limitation of protobuf JSON serialization.
      // Track: https://github.com/bufbuild/protobuf/issues
      if (params.evolutionMode && request.evolutionMode === undefined) {
        (request as unknown as { evolutionMode?: boolean }).evolutionMode = true;
      }

      // Cancel any existing request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      abortControllerRef.current = new AbortController();
      const signal = abortControllerRef.current.signal;

      // Set up timeout for the entire stream operation
      // Clear any existing timeout first
      if (timeoutIdRef.current) {
        clearTimeout(timeoutIdRef.current);
      }
      timeoutIdRef.current = setTimeout(() => {
        if (abortControllerRef.current) {
          abortControllerRef.current.abort();
        }
        timeoutIdRef.current = null;
        if (import.meta.env.DEV) {
          console.warn("[AI Chat] Stream timeout exceeded", { timeoutMs: STREAM_TIMEOUT_MS });
        }
      }, STREAM_TIMEOUT_MS);

      const startTime = Date.now();

      try {
        // Use the streaming method from Connect RPC client
        const stream = aiServiceClient.chat(request, { signal });

        const sources: string[] = [];
        let fullContent = "";
        let doneCalled = false;
        let responseCount = 0;

        if (import.meta.env.DEV) {
          console.log("[AI Chat] Starting stream loop", { message: params.message.slice(0, 50) });
        }

        for await (const response of stream) {
          responseCount++;
          if (import.meta.env.DEV) {
            console.log("[AI Chat] Stream response", {
              responseCount,
              hasContent: !!response.content,
              hasEventType: !!response.eventType,
              hasEventMeta: !!response.eventMeta,
              done: response.done,
              hasBlockSummary: !!response.blockSummary,
              hasBlockId: response.blockId !== undefined && response.blockId !== 0n,
              blockId: response.blockId,
              eventType: response.eventType,
            });
          }

          // Phase 4: Handle Block ID - invalidate blocks cache to trigger refetch
          // When we receive a block_id, it means a new block was created/updated
          // Invalidate the blocks query to fetch the latest data
          const blockId = response.blockId;
          if (blockId !== undefined && blockId !== 0n && params.conversationId) {
            if (import.meta.env.DEV) {
              console.log("[AI Chat] Received block_id, invalidating blocks cache", {
                blockId: blockId.toString(),
                conversationId: params.conversationId,
              });
            }
            // Invalidate blocks cache for this conversation to trigger refetch
            queryClient.invalidateQueries({
              queryKey: blockKeys.list(params.conversationId),
            });
          }

          // Handle sources (sent in first response)
          if (response.sources.length > 0) {
            sources.push(...response.sources);
            callbacks?.onSources?.(response.sources);
          }

          // Handle content chunks
          if (response.content) {
            fullContent += response.content;
            callbacks?.onContent?.(response.content);
          }

          // Handle schedule creation intent (sent in final chunk)
          if (response.scheduleCreationIntent?.detected) {
            callbacks?.onScheduleIntent?.({
              detected: response.scheduleCreationIntent.detected,
              scheduleDescription: response.scheduleCreationIntent.scheduleDescription || "",
            });
          }

          // Handle schedule query result (sent in final chunk)
          if (response.scheduleQueryResult?.detected) {
            const schedules = (response.scheduleQueryResult.schedules || []).map((sched) => ({
              uid: sched.uid || "",
              title: sched.title || "",
              startTs: sched.startTs ? BigInt(sched.startTs) : BigInt(0),
              endTs: sched.endTs ? BigInt(sched.endTs) : BigInt(0),
              allDay: sched.allDay || false,
              location: sched.location || "",
              recurrenceRule: sched.recurrenceRule || "",
              status: sched.status || "ACTIVE",
            }));

            callbacks?.onScheduleQueryResult?.({
              detected: response.scheduleQueryResult.detected,
              schedules,
              timeRangeDescription: response.scheduleQueryResult.timeRangeDescription || "",
              queryType: response.scheduleQueryResult.queryType || "",
            });
          }

          // Handle parrot-specific events
          if (response.eventType && response.eventData) {
            if (import.meta.env.DEV) {
              console.debug("[AI Chat] Parrot event", {
                eventType: response.eventType,
                eventDataLength: response.eventData.length,
                eventDataPreview: response.eventData.slice(0, EVENT_DATA_PREVIEW_LENGTH),
                eventMeta: response.eventMeta,
              });
            }
            switch (response.eventType) {
              case "thinking":
                callbacks?.onThinking?.(response.eventData);
                break;
              case "tool_use": {
                // Convert proto EventMetadata (bigint fields) to local EventMetadata (number fields)
                const toolMeta = response.eventMeta
                  ? {
                      durationMs: safeBigintToNumber(response.eventMeta.durationMs),
                      totalDurationMs: safeBigintToNumber(response.eventMeta.totalDurationMs),
                      toolName: response.eventMeta.toolName,
                      toolId: response.eventMeta.toolId,
                      status: response.eventMeta.status,
                      errorMsg: response.eventMeta.errorMsg,
                      inputTokens: response.eventMeta.inputTokens,
                      outputTokens: response.eventMeta.outputTokens,
                      cacheWriteTokens: response.eventMeta.cacheWriteTokens,
                      cacheReadTokens: response.eventMeta.cacheReadTokens,
                      inputSummary: response.eventMeta.inputSummary,
                      outputSummary: response.eventMeta.outputSummary,
                      filePath: response.eventMeta.filePath,
                      lineCount: response.eventMeta.lineCount,
                    }
                  : undefined;
                callbacks?.onToolUse?.(response.eventData, toolMeta);
                break;
              }
              case "tool_result": {
                // Convert proto EventMetadata (bigint fields) to local EventMetadata (number fields)
                const resultMeta = response.eventMeta
                  ? {
                      durationMs: safeBigintToNumber(response.eventMeta.durationMs),
                      totalDurationMs: safeBigintToNumber(response.eventMeta.totalDurationMs),
                      toolName: response.eventMeta.toolName,
                      toolId: response.eventMeta.toolId,
                      status: response.eventMeta.status,
                      errorMsg: response.eventMeta.errorMsg,
                      inputTokens: response.eventMeta.inputTokens,
                      outputTokens: response.eventMeta.outputTokens,
                      cacheWriteTokens: response.eventMeta.cacheWriteTokens,
                      cacheReadTokens: response.eventMeta.cacheReadTokens,
                      inputSummary: response.eventMeta.inputSummary,
                      outputSummary: response.eventMeta.outputSummary,
                      filePath: response.eventMeta.filePath,
                      lineCount: response.eventMeta.lineCount,
                    }
                  : undefined;
                callbacks?.onToolResult?.(response.eventData, resultMeta);
                break;
              }
              case "answer":
                // Handle final answer from agent (when no tool is used)
                fullContent += response.eventData;
                callbacks?.onContent?.(response.eventData);
                break;
              case "memo_query_result":
                try {
                  const result = JSON.parse(response.eventData) as {
                    memos: Array<{ uid: string; content: string; score: number }>;
                    query: string;
                    count: number;
                  };
                  callbacks?.onMemoQueryResult?.(result);
                } catch (e) {
                  if (import.meta.env.DEV) {
                    console.error("[useAIQueries] Failed to parse memo_query_result:", e);
                  }
                }
                break;
              case "schedule_query_result":
                try {
                  const result = JSON.parse(response.eventData) as {
                    schedules: Array<{
                      uid: string;
                      title: string;
                      start_ts: number;
                      end_ts: number;
                      all_day: boolean;
                      location?: string;
                      status: string;
                    }>;
                    time_range_description?: string;
                    query_type?: string;
                  };
                  // Transform to the expected format with bigint conversion
                  const transformedResult = {
                    detected: true,
                    schedules: (result.schedules || []).map((s) => ({
                      uid: s.uid || "",
                      title: s.title || "",
                      startTs: BigInt(s.start_ts || 0),
                      endTs: BigInt(s.end_ts || 0),
                      allDay: s.all_day || false,
                      location: s.location || "",
                      recurrenceRule: "",
                      status: s.status || "ACTIVE",
                    })),
                    timeRangeDescription: result.time_range_description || "",
                    queryType: result.query_type || "range",
                  };
                  callbacks?.onScheduleQueryResult?.(transformedResult);
                } catch (e) {
                  if (import.meta.env.DEV) {
                    console.error("[useAIQueries] Failed to parse schedule_query_result:", e);
                  }
                }
                break;
            }
          }

          // Handle completion
          if (response.done === true) {
            if (import.meta.env.DEV) {
              console.log("[AI Chat] Received done=true signal", {
                responseCount,
                hasBlockSummary: !!response.blockSummary,
                sessionId: response.blockSummary?.sessionId,
                blockSummary: response.blockSummary,
                fullResponse: response,
              });
            }
            doneCalled = true;
            // Send block summary if available (Geek/Evolution modes)
            if (response.blockSummary) {
              // Convert proto BlockSummary (bigint fields) to local BlockSummary (number fields)
              // Note: mode is NOT included here - use Block.mode as the single source of truth
              const summary = {
                sessionId: response.blockSummary.sessionId,
                totalDurationMs: response.blockSummary.totalDurationMs ? Number(response.blockSummary.totalDurationMs) : undefined,
                thinkingDurationMs: response.blockSummary.thinkingDurationMs ? Number(response.blockSummary.thinkingDurationMs) : undefined,
                toolDurationMs: response.blockSummary.toolDurationMs ? Number(response.blockSummary.toolDurationMs) : undefined,
                generationDurationMs: response.blockSummary.generationDurationMs
                  ? Number(response.blockSummary.generationDurationMs)
                  : undefined,
                totalInputTokens: response.blockSummary.totalInputTokens,
                totalOutputTokens: response.blockSummary.totalOutputTokens,
                totalCacheWriteTokens: response.blockSummary.totalCacheWriteTokens,
                totalCacheReadTokens: response.blockSummary.totalCacheReadTokens,
                toolCallCount: response.blockSummary.toolCallCount,
                toolsUsed: response.blockSummary.toolsUsed,
                filesModified: response.blockSummary.filesModified,
                filePaths: response.blockSummary.filePaths,
                totalCostUSD: response.blockSummary.totalCostUsd,
                status: response.blockSummary.status,
                errorMsg: response.blockSummary.errorMsg,
              };
              callbacks?.onBlockSummary?.(summary);
            }
            callbacks?.onDone?.();
            break;
          }
        }

        // Fallback: if stream ended without done signal, call onDone
        if (!doneCalled) {
          if (import.meta.env.DEV) {
            console.warn("[AI Chat] Stream ended without done=true signal", {
              responseCount,
              doneCalled,
              callingFallback: true,
            });
          }
          callbacks?.onDone?.();
        } else if (import.meta.env.DEV) {
          console.log("[AI Chat] Stream completed successfully", { responseCount, doneCalled });
        }

        // Clear timeout on successful completion
        if (timeoutIdRef.current) {
          clearTimeout(timeoutIdRef.current);
          timeoutIdRef.current = null;
        }

        const duration = Date.now() - startTime;
        if (import.meta.env.DEV) {
          console.debug("[AI Chat] Stream completed successfully", {
            durationMs: duration,
            contentLength: fullContent.length,
            sourcesCount: sources.length,
          });
        }

        return { content: fullContent, sources };
      } catch (error) {
        // Clear timeout on error
        if (timeoutIdRef.current) {
          clearTimeout(timeoutIdRef.current);
          timeoutIdRef.current = null;
        }

        const duration = Date.now() - startTime;

        // Check if it's an abort error (timeout)
        if (error instanceof Error && error.name === "AbortError") {
          if (import.meta.env.DEV) {
            console.error("[AI Chat] Stream timeout", { durationMs: duration, timeoutMs: STREAM_TIMEOUT_MS });
          }
          const timeoutErr = new Error(`AI chat timeout after ${STREAM_TIMEOUT_MS}ms`);
          callbacks?.onError?.(timeoutErr);
          throw timeoutErr;
        }

        if (import.meta.env.DEV) {
          console.error("[AI Chat] Stream error", {
            error,
            durationMs: duration,
            errorMessage: error instanceof Error ? error.message : String(error),
          });
        }

        const err = error instanceof Error ? error : new Error(String(error));
        callbacks?.onError?.(err);
        throw err;
      }
    },
    /**
     * Stop the current chat stream.
     */
    stop: () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
        abortControllerRef.current = null;
      }
      if (timeoutIdRef.current) {
        clearTimeout(timeoutIdRef.current);
        timeoutIdRef.current = null;
      }
      if (import.meta.env.DEV) {
        console.debug("[AI Chat] Stream manually stopped");
      }
    },
    /**
     * Invalidate AI-related queries after chat
     */
    invalidate: () => {
      queryClient.invalidateQueries({ queryKey: aiKeys.all });
    },
  };
}

// Type exports for convenience
export type SemanticSearchResult = Awaited<ReturnType<typeof aiServiceClient.semanticSearch>>;
export type SuggestTagsResult = string[];
export type RelatedMemosResult = Awaited<ReturnType<typeof aiServiceClient.getRelatedMemos>>;
export type DetectDuplicatesResult = Awaited<ReturnType<typeof aiServiceClient.detectDuplicates>>;

/**
 * useDetectDuplicates checks for duplicate or related memos.
 */
export function useDetectDuplicates() {
  return useMutation({
    mutationFn: async (params: { title?: string; content: string; tags?: string[]; topK?: number }) => {
      const request = create(DetectDuplicatesRequestSchema, {
        title: params.title ?? "",
        content: params.content,
        tags: params.tags ?? [],
        topK: params.topK ?? 5,
      });
      return await aiServiceClient.detectDuplicates(request);
    },
  });
}

/**
 * useMergeMemos merges source memo into target memo.
 */
export function useMergeMemos() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (params: { sourceName: string; targetName: string }) => {
      const request = create(MergeMemosRequestSchema, {
        sourceName: params.sourceName,
        targetName: params.targetName,
      });
      return await aiServiceClient.mergeMemos(request);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["memos"] });
    },
  });
}

/**
 * useLinkMemos creates a bidirectional relation between two memos.
 */
export function useLinkMemos() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (params: { memoName1: string; memoName2: string }) => {
      const request = create(LinkMemosRequestSchema, {
        memoName1: params.memoName1,
        memoName2: params.memoName2,
      });
      return await aiServiceClient.linkMemos(request);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["memos"] });
    },
  });
}

/**
 * useKnowledgeGraph fetches the knowledge graph for visualization.
 * @param filter - Graph filter options
 * @param options.enabled - Whether the query is enabled
 */
export function useKnowledgeGraph(
  filter: { tags: string[]; minImportance: number; clusters: number[] },
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: aiKeys.knowledgeGraph(filter),
    queryFn: async () => {
      const request = create(GetKnowledgeGraphRequestSchema, {
        tags: filter.tags,
        minImportance: filter.minImportance,
        clusters: filter.clusters,
      });
      return await aiServiceClient.getKnowledgeGraph(request);
    },
    enabled: options.enabled ?? true,
    staleTime: STALE_TIME_LONG_MS, // 5 minutes
  });
}

// Knowledge graph types
export type KnowledgeGraphResult = Awaited<ReturnType<typeof aiServiceClient.getKnowledgeGraph>>;

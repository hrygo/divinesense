import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { aiServiceClient } from "@/connect";
import { ParrotAgentType, parrotToProtoAgentType } from "@/types/parrot";
import type { Block } from "@/types/proto/api/v1/ai_service_pb";
import {
  BlockMode,
  BlockStatus,
  BlockType,
  ChatRequestSchema,
  DetectDuplicatesRequestSchema,
  GetKnowledgeGraphRequestSchema,
  GetRelatedMemosRequestSchema,
  LinkMemosRequestSchema,
  MergeMemosRequestSchema,
  SemanticSearchRequestSchema,
  SessionStatsSchema,
  SuggestTagsRequestSchema,
} from "@/types/proto/api/v1/ai_service_pb";
// Phase 4: Import blockKeys for invalidating blocks on updates
import { blockKeys } from "./useBlockQueries";

/**
 * Get BlockMode from mode flags
 * Priority: Geek > Evolution > Normal
 */
function getBlockModeFromFlags(geekMode?: boolean, evolutionMode?: boolean): BlockMode {
  if (geekMode) return BlockMode.GEEK;
  if (evolutionMode) return BlockMode.EVOLUTION;
  return BlockMode.NORMAL;
}

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

  // Cleanup on unmount to prevent memory leaks
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
        abortControllerRef.current = null;
      }
      if (timeoutIdRef.current) {
        clearTimeout(timeoutIdRef.current);
        timeoutIdRef.current = null;
      }
    };
  }, []);

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

        for await (const response of stream) {
          responseCount++;

          // DEBUG: Log all events to diagnose streaming issues
          if (import.meta.env.DEV && response.eventType) {
            console.log(
              "[AI Chat] Received event:",
              response.eventType,
              "blockId:",
              String(response.blockId),
              "eventData:",
              response.eventData,
            );
          }

          // Phase 4: Handle Block ID - create optimistic block immediately for instant UI feedback
          // When we receive a block_id, it means a new block was created/updated
          // Instead of just invalidating, we create an optimistic block for instant rendering
          const blockId = response.blockId;

          // Helper function to update optimistic block's eventStream during streaming
          const updateBlockEventStream = (event: {
            type: string;
            content?: string;
            toolName?: string;
            toolId?: string;
            inputSummary?: string;
            outputSummary?: string;
            filePath?: string;
            duration?: number;
            isError?: boolean;
            timestamp: number;
          }) => {
            if (import.meta.env.DEV) {
              console.log("[AI Chat] updateBlockEventStream called:", {
                eventType: event.type,
                blockId: String(blockId),
                blockIdIsZero: blockId === 0n,
                conversationId: params.conversationId,
              });
            }
            if (!blockId || blockId === 0n || !params.conversationId) return;

            // Create new event object
            const newEvent = {
              $typeName: "memos.api.v1.BlockEvent" as const,
              type: event.type,
              content: event.content || "",
              timestamp: BigInt(event.timestamp),
              meta: JSON.stringify({
                tool_name: event.toolName,
                tool_id: event.toolId,
                input_summary: event.inputSummary,
                output_summary: event.outputSummary,
                file_path: event.filePath,
                duration: event.duration,
                is_error: event.isError,
              }),
            };

            // Direct cache update (SSE is serial, no race conditions)
            queryClient.setQueryData(blockKeys.list(params.conversationId), (old) => {
              const existing = old as { blocks?: Block[]; totalCount?: number } | undefined;
              const existingBlocks = existing?.blocks || [];
              const blockMap = new Map(existingBlocks.map((b) => [b.id, b]));
              const currentBlock = blockMap.get(blockId);

              if (currentBlock) {
                const updated = {
                  ...currentBlock,
                  eventStream: [...(currentBlock.eventStream || []), newEvent],
                  updatedTs: BigInt(Date.now()),
                };
                blockMap.set(blockId, updated);
              }

              return {
                blocks: Array.from(blockMap.values()),
                totalCount: blockMap.size,
              };
            });

            // Also update detail cache
            queryClient.setQueryData(blockKeys.detail(Number(blockId)), (old: Block | undefined) => {
              if (!old || old.id !== blockId) return old;
              return {
                ...old,
                eventStream: [...(old.eventStream || []), newEvent],
                updatedTs: BigInt(Date.now()),
              };
            });
          };

          if (blockId !== undefined && blockId !== 0n && params.conversationId) {
            // CRITICAL: Only create optimistic block if it doesn't exist yet!
            // If the block already exists in cache, we should NOT overwrite it,
            // as that would reset eventStream and lose accumulated events.
            const existing = queryClient.getQueryData(blockKeys.list(params.conversationId)) as
              | { blocks?: Block[]; totalCount?: number }
              | undefined;
            const existingBlocks = existing?.blocks || [];
            const blockAlreadyExists = existingBlocks.some((b) => b.id === blockId);

            if (import.meta.env.DEV) {
              console.log("[AI Chat] Optimistic block creation check:", {
                blockId: String(blockId),
                blockAlreadyExists,
                existingBlocksCount: existingBlocks.length,
                eventType: response.eventType,
                action: blockAlreadyExists ? "Skipping - block already exists" : "Creating new optimistic block",
              });
            }

            // CRITICAL FIX: Only create if block doesn't exist
            if (!blockAlreadyExists) {
              // Get the current roundNumber from existing blocks to prevent flicker
              const optimisticRoundNumber =
                existingBlocks.length > 0 ? Math.max(...existingBlocks.map((b) => Number(b.roundNumber))) + 1 : 1;

              const now = BigInt(Date.now());
              const optimisticBlock: Block = {
                $typeName: "memos.api.v1.Block" as const,
                id: blockId,
                uid: `optimistic-${blockId}`, // Will be replaced by backend data
                conversationId: params.conversationId, // number type
                roundNumber: optimisticRoundNumber, // Use predicted roundNumber (number type) to prevent flicker
                mode: getBlockModeFromFlags(params.geekMode, params.evolutionMode),
                blockType: BlockType.MESSAGE,
                userInputs: [
                  {
                    $typeName: "memos.api.v1.UserInput" as const,
                    content: params.message,
                    timestamp: now,
                    metadata: "{}",
                  },
                ],
                assistantContent: "", // Will be filled during streaming
                eventStream: [], // Start empty, will be populated by event handlers
                status: BlockStatus.STREAMING,
                metadata: "{}",
                createdTs: now,
                updatedTs: now,
                assistantTimestamp: now,
                ccSessionId: "",
                parentBlockId: BigInt(0),
                branchPath: "",
                costEstimate: BigInt(0),
                modelVersion: "",
                userFeedback: "",
                regenerationCount: 0,
                errorMessage: "",
                archivedAt: BigInt(0),
                sessionStats: undefined,
              };

              // CRITICAL: 同步写入缓存（不走队列），确保后续事件能找到这个 block
              queryClient.setQueryData(blockKeys.list(params.conversationId), (old) => {
                const existing = old as { blocks?: Block[]; totalCount?: number } | undefined;
                const existingBlocksInner = existing?.blocks || [];

                // Use Map for O(1) lookup and automatic deduplication
                const blockMap = new Map(existingBlocksInner.map((b) => [b.id, b]));
                blockMap.set(blockId, optimisticBlock); // Set will overwrite if exists, ensuring no duplicates

                const newBlocks = Array.from(blockMap.values());
                return {
                  blocks: newBlocks,
                  totalCount: newBlocks.length,
                };
              });

              // Also cache the individual block detail
              queryClient.setQueryData(blockKeys.detail(Number(blockId)), optimisticBlock);
            }
          }

          // Handle sources (sent in first response)
          if (response.sources.length > 0) {
            sources.push(...response.sources);
            callbacks?.onSources?.(response.sources);
          }

          // Handle content chunks
          if (response.content) {
            if (import.meta.env.DEV) {
              console.log("[AI Chat] response.content found:", {
                content: response.content,
                blockId: String(blockId),
                note: "response.content should not be set by backend!",
              });
            }
            fullContent += response.content;
            callbacks?.onContent?.(response.content);

            // CRITICAL: Update optimistic block's assistantContent during streaming
            // CRITICAL: Preserve eventStream to avoid losing accumulated events
            if (blockId !== undefined && blockId !== 0n && params.conversationId) {
              queryClient.setQueryData(blockKeys.list(params.conversationId), (old) => {
                const existing = old as { blocks?: Block[]; totalCount?: number } | undefined;
                const existingBlocks = existing?.blocks || [];
                const blockMap = new Map(existingBlocks.map((b) => [b.id, b]));
                const currentBlock = blockMap.get(blockId);

                if (currentBlock) {
                  const updated = {
                    ...currentBlock,
                    assistantContent: fullContent,
                    eventStream: currentBlock.eventStream || [], // Preserve eventStream!
                    updatedTs: BigInt(Date.now()),
                  };
                  blockMap.set(blockId, updated);
                }

                return {
                  blocks: Array.from(blockMap.values()),
                  totalCount: blockMap.size,
                };
              });
            }
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
          // Note: eventData may be empty for some events (e.g., tool_use with no params), so check eventType only
          if (response.eventType) {
            switch (response.eventType) {
              case "thinking":
                callbacks?.onThinking?.(response.eventData);
                // Update optimistic block's eventStream to show thinking in UI
                updateBlockEventStream({
                  type: "thinking",
                  content: response.eventData,
                  timestamp: Date.now(),
                });
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
                // Update optimistic block's eventStream to show tool_use in UI
                // CRITICAL FIX: Use toolMeta.toolName instead of response.eventData (which is the JSON parameters)
                updateBlockEventStream({
                  type: "tool_use",
                  toolName: toolMeta?.toolName,
                  toolId: toolMeta?.toolId,
                  inputSummary: toolMeta?.inputSummary,
                  outputSummary: toolMeta?.outputSummary,
                  filePath: toolMeta?.filePath,
                  timestamp: Date.now(),
                });
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
                // Update optimistic block's eventStream to show tool_result in UI
                updateBlockEventStream({
                  type: "tool_result",
                  content: response.eventData,
                  toolName: resultMeta?.toolName,
                  toolId: resultMeta?.toolId,
                  outputSummary: resultMeta?.outputSummary,
                  duration: resultMeta?.durationMs,
                  isError: !!resultMeta?.errorMsg,
                  timestamp: Date.now(),
                });
                break;
              }
              case "answer":
                // Handle final answer from agent (when no tool is used)
                fullContent += response.eventData;
                callbacks?.onContent?.(response.eventData);
                // CRITICAL: Real-time update assistantContent for streaming UI
                // CRITICAL: Preserve eventStream to avoid losing accumulated events
                if (blockId !== undefined && blockId !== 0n && params.conversationId) {
                  queryClient.setQueryData(blockKeys.list(params.conversationId), (old) => {
                    const existing = old as { blocks?: Block[]; totalCount?: number } | undefined;
                    const existingBlocks = existing?.blocks || [];
                    const blockMap = new Map(existingBlocks.map((b) => [b.id, b]));
                    const currentBlock = blockMap.get(blockId);

                    if (currentBlock) {
                      const updated = {
                        ...currentBlock,
                        assistantContent: fullContent,
                        status: BlockStatus.STREAMING,
                        eventStream: currentBlock.eventStream || [], // Preserve eventStream!
                        updatedTs: BigInt(Date.now()),
                      };
                      blockMap.set(blockId, updated);
                    }

                    return {
                      blocks: Array.from(blockMap.values()),
                      totalCount: blockMap.size,
                    };
                  });
                }
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
            doneCalled = true;

            // CRITICAL: Mark block as COMPLETED and update final content
            if (blockId !== undefined && blockId !== 0n && params.conversationId) {
              // Build SessionStats from response.blockSummary for persistence
              // This ensures BlockSummary shows up immediately without page refresh (#55)
              let sessionStats: ReturnType<typeof create<typeof SessionStatsSchema>> | undefined;
              if (response.blockSummary) {
                const now = BigInt(Date.now());
                const nowSec = BigInt(Math.floor(Date.now() / 1000));
                sessionStats = create(SessionStatsSchema, {
                  id: BigInt(0), // Will be set by backend
                  sessionId: response.blockSummary.sessionId || "",
                  conversationId: BigInt(params.conversationId),
                  userId: 0, // Will be set by backend
                  agentType: "auto", // Will be set by backend
                  startedAt: nowSec,
                  endedAt: nowSec,
                  totalDurationMs: response.blockSummary.totalDurationMs || 0n,
                  thinkingDurationMs: response.blockSummary.thinkingDurationMs || 0n,
                  toolDurationMs: response.blockSummary.toolDurationMs || 0n,
                  generationDurationMs: response.blockSummary.generationDurationMs || 0n,
                  inputTokens: response.blockSummary.totalInputTokens || 0,
                  outputTokens: response.blockSummary.totalOutputTokens || 0,
                  cacheWriteTokens: response.blockSummary.totalCacheWriteTokens || 0,
                  cacheReadTokens: response.blockSummary.totalCacheReadTokens || 0,
                  // FIX: totalTokens expects number in proto but calculate as number
                  totalTokens: (response.blockSummary.totalInputTokens || 0) + (response.blockSummary.totalOutputTokens || 0),
                  totalCostUsd: response.blockSummary.totalCostUsd || 0,
                  toolCallCount: response.blockSummary.toolCallCount || 0,
                  toolsUsed: response.blockSummary.toolsUsed?.length ? response.blockSummary.toolsUsed : [],
                  filesModified: response.blockSummary.filesModified || 0,
                  filePaths: response.blockSummary.filePaths?.length ? response.blockSummary.filePaths : [],
                  modelUsed: "",
                  isError: response.blockSummary.status === "error",
                  errorMessage: response.blockSummary.errorMsg || "",
                  createdAt: now,
                  updatedAt: now,
                });
              }

              queryClient.setQueryData(blockKeys.list(params.conversationId), (old) => {
                const existing = old as { blocks?: Block[]; totalCount?: number } | undefined;
                const existingBlocks = existing?.blocks || [];
                const blockMap = new Map(existingBlocks.map((b) => [b.id, b]));
                const currentBlock = blockMap.get(blockId);

                if (currentBlock) {
                  const updated = {
                    ...currentBlock,
                    assistantContent: fullContent,
                    status: BlockStatus.COMPLETED,
                    eventStream: currentBlock.eventStream || [], // Preserve eventStream!
                    sessionStats, // FIX #55: Store sessionStats for BlockSummary display
                    updatedTs: BigInt(Date.now()),
                  };
                  blockMap.set(blockId, updated);
                }

                return {
                  blocks: Array.from(blockMap.values()),
                  totalCount: blockMap.size,
                };
              });

              // Also update the detail cache
              queryClient.setQueryData(blockKeys.detail(Number(blockId)), (oldBlock: Block | undefined) => {
                if (!oldBlock || oldBlock.id !== blockId) return oldBlock;
                return {
                  ...oldBlock,
                  assistantContent: fullContent,
                  status: BlockStatus.COMPLETED,
                  eventStream: oldBlock.eventStream || [],
                  sessionStats, // FIX #55: Store sessionStats for BlockSummary display
                  updatedTs: BigInt(Date.now()),
                };
              });
            }

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
        }

        // Clear timeout on successful completion
        if (timeoutIdRef.current) {
          clearTimeout(timeoutIdRef.current);
          timeoutIdRef.current = null;
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

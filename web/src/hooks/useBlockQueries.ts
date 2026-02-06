/**
 * Block API Hooks for Unified Block Model
 *
 * Provides React Query hooks for interacting with the Block API.
 * Optimized with caching, optimistic updates, and retry strategies.
 *
 * Phase 4: Frontend Block hooks (Optimized)
 * @see docs/specs/unified-block-model.md
 */

import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ApiError } from "@/config/errors";
import { createRetryDelay, shouldRetryError } from "@/config/errors";
import { aiServiceClient } from "@/connect";
import type {
  AppendEventRequest,
  AppendUserInputRequest,
  Block,
  CreateBlockRequest,
  DeleteBlockRequest,
  ListBlocksRequest,
  UpdateBlockRequest,
  UserInput,
} from "@/types/proto/api/v1/ai_service_pb";
import {
  AppendEventRequestSchema,
  AppendUserInputRequestSchema,
  BlockMode,
  BlockStatus,
  BlockType,
  CreateBlockRequestSchema,
  DeleteBlockRequestSchema,
  GetBlockRequestSchema,
  ListBlocksRequestSchema,
  UpdateBlockRequestSchema,
} from "@/types/proto/api/v1/ai_service_pb";

// ============================================================================
// Query Configuration
// ============================================================================

/**
 * Cache data structure for block list queries
 * @internal
 */
interface BlockListCacheData {
  blocks: Block[];
  totalCount: number;
}

// Query keys factory for consistent cache management

/** Cache time configuration for different query types */
const CACHE_TIMES = {
  /** Block lists - cache for 1 minute */
  BLOCK_LIST: 1000 * 60,
  /** Single block - cache for 30 seconds */
  BLOCK_DETAIL: 1000 * 30,
  /** Active conversation blocks - cache for 10 seconds */
  ACTIVE_CONVERSATION: 1000 * 10,
} as const;

/** Stale time configuration - data is considered fresh for this duration */
const STALE_TIMES = {
  /** Block lists - fresh for 30 seconds */
  BLOCK_LIST: 1000 * 30,
  /** Single block - fresh for 10 seconds */
  BLOCK_DETAIL: 1000 * 10,
  /** Active conversation - fresh for 5 seconds */
  ACTIVE_CONVERSATION: 1000 * 5,
} as const;

/** Retry configuration for different error types */
const RETRY_CONFIG = {
  /** Maximum retry attempts for network errors */
  MAX_RETRIES: 3,
  /** Base retry delay in milliseconds */
  BASE_RETRY_DELAY: 1000,
  /** Maximum retry delay in milliseconds */
  MAX_RETRY_DELAY: 30000,
} as const;

// Query keys factory for consistent cache management
export const blockKeys = {
  all: ["blocks"] as const,
  lists: () => [...blockKeys.all, "list"] as const,
  list: (conversationId: number, filters?: Partial<ListBlocksRequest>) => [...blockKeys.lists(), conversationId, filters] as const,
  details: () => [...blockKeys.all, "detail"] as const,
  detail: (id: number) => [...blockKeys.details(), id] as const,
};

/**
 * Hook to fetch blocks for a conversation
 *
 * @param conversationId - The conversation ID to fetch blocks for
 * @param filters - Optional filters for the block list
 * @param options - Additional options like isActive (for active conversations)
 */
export function useBlocks(conversationId: number, filters?: Partial<ListBlocksRequest>, options?: { isActive?: boolean }) {
  const is_active = options?.isActive ?? false;

  return useQuery({
    queryKey: blockKeys.list(conversationId, filters),
    queryFn: async () => {
      const request = create(ListBlocksRequestSchema, {
        conversationId,
        ...filters,
      } as Record<string, unknown>);
      const response = await aiServiceClient.listBlocks(request);
      return response;
    },
    enabled: conversationId > 0,
    staleTime: is_active ? STALE_TIMES.ACTIVE_CONVERSATION : STALE_TIMES.BLOCK_LIST,
    gcTime: is_active ? CACHE_TIMES.ACTIVE_CONVERSATION : CACHE_TIMES.BLOCK_LIST,
    retry: (failureCount, error) => {
      // Use type-safe error handling instead of any
      const err = error as ApiError;
      return shouldRetryError(err, failureCount, RETRY_CONFIG.MAX_RETRIES);
    },
    retryDelay: createRetryDelay(RETRY_CONFIG.BASE_RETRY_DELAY, RETRY_CONFIG.MAX_RETRY_DELAY),
    refetchOnWindowFocus: is_active,
    refetchOnReconnect: true,
  });
}

/**
 * Hook to fetch a single block by ID
 *
 * @param id - The block ID to fetch
 * @param options - Additional options
 */
export function useBlock(id: number, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: blockKeys.detail(id),
    queryFn: async () => {
      const request = create(GetBlockRequestSchema, { id: BigInt(id) });
      const response = await aiServiceClient.getBlock(request);
      return response;
    },
    enabled: (options?.enabled ?? true) && id > 0,
    staleTime: STALE_TIMES.BLOCK_DETAIL,
    gcTime: CACHE_TIMES.BLOCK_DETAIL,
    retry: (failureCount, error) => {
      const err = error as ApiError;
      return shouldRetryError(err, failureCount, RETRY_CONFIG.MAX_RETRIES);
    },
    retryDelay: createRetryDelay(RETRY_CONFIG.BASE_RETRY_DELAY, RETRY_CONFIG.MAX_RETRY_DELAY),
  });
}

/**
 * Hook to create a new block with optimistic update
 *
 * Optimistically adds the block to the cache before the server responds,
 * rolling back on error.
 */
export function useCreateBlock() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: CreateBlockRequest) => {
      const req = create(CreateBlockRequestSchema, request as Record<string, unknown>);
      const response = await aiServiceClient.createBlock(req);
      return response;
    },
    onMutate: async (variables) => {
      // Cancel outgoing refetches
      const conversationId = Number(variables.conversationId || 0);
      await queryClient.cancelQueries({ queryKey: blockKeys.list(conversationId) });

      // Snapshot previous value
      const previousBlocks = queryClient.getQueryData(blockKeys.list(conversationId));

      // Generate temp ID for optimistic update
      const tempId = BigInt(-Date.now());

      // Create optimistic block with proper type safety
      const now = BigInt(Date.now());
      const optimisticBlock: Block = {
        $typeName: "memos.api.v1.Block" as const,
        id: tempId,
        uid: `temp-${tempId}`,
        conversationId: conversationId,
        roundNumber: 0,
        mode: variables.mode || BlockMode.NORMAL,
        blockType: variables.blockType || BlockType.MESSAGE,
        userInputs: variables.userInputs || [],
        assistantContent: "",
        eventStream: [],
        status: BlockStatus.PENDING,
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
      };

      // Optimistically update cache
      queryClient.setQueryData(blockKeys.list(conversationId), (old: BlockListCacheData | undefined) => {
        if (!old) return { blocks: [optimisticBlock], totalCount: 1 };
        return {
          ...old,
          blocks: [...old.blocks, optimisticBlock],
          totalCount: (old.totalCount || 0) + 1,
        };
      });

      // Return context with rollback function
      return { previousBlocks, conversationId, tempId };
    },
    onError: (_error, _variables, context) => {
      // Rollback on error
      if (context?.previousBlocks) {
        queryClient.setQueryData(blockKeys.list(context.conversationId), context.previousBlocks);
      }
    },
    onSuccess: (newBlock, variables, context) => {
      // Update cache with actual server response
      const conversationId = Number(variables.conversationId);
      queryClient.setQueryData(blockKeys.detail(Number(newBlock.id)), newBlock);

      // Replace optimistic block in list with actual block
      queryClient.setQueryData(blockKeys.list(conversationId), (old: BlockListCacheData | undefined) => {
        if (!old) return { blocks: [newBlock], totalCount: 1 };
        return {
          ...old,
          blocks: old.blocks.map((b: Block) => (b.id === context?.tempId ? newBlock : b)),
        };
      });
    },
    onSettled: (_data, _error, variables) => {
      // Refetch to ensure consistency
      queryClient.invalidateQueries({
        queryKey: blockKeys.list(Number(variables.conversationId)),
      });
    },
    // Mutation operations should not auto-retry (may cause duplicate creation)
    retry: 0,
  });
}

/**
 * Hook to update a block with optimistic update
 *
 * Supports streaming updates where the block status changes frequently.
 */
export function useUpdateBlock() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: UpdateBlockRequest) => {
      const req = create(UpdateBlockRequestSchema, request as Record<string, unknown>);
      const response = await aiServiceClient.updateBlock(req);
      return response;
    },
    onMutate: async (variables) => {
      const blockId = Number(variables.id);

      // Snapshot previous value
      const previousBlock = queryClient.getQueryData(blockKeys.detail(blockId));

      // Optimistically update cache
      queryClient.setQueryData(blockKeys.detail(blockId), (old: Block | undefined) => {
        if (!old) return old;
        return {
          ...old,
          ...variables,
          updatedTs: BigInt(Date.now()),
        };
      });

      return { previousBlock, blockId };
    },
    onError: (_error, _variables, context) => {
      // Rollback on error
      if (context?.previousBlock) {
        queryClient.setQueryData(blockKeys.detail(context.blockId), context.previousBlock);
      }
    },
    onSuccess: (updatedBlock) => {
      // Update block cache
      queryClient.setQueryData(blockKeys.detail(Number(updatedBlock.id)), updatedBlock);
    },
    onSettled: () => {
      // Invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: blockKeys.details() });
      queryClient.invalidateQueries({ queryKey: blockKeys.lists() });
    },
  });
}

/**
 * Hook to delete a block
 */
export function useDeleteBlock() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: DeleteBlockRequest) => {
      const req = create(DeleteBlockRequestSchema, request as Record<string, unknown>);
      const response = await aiServiceClient.deleteBlock(req);
      return response;
    },
    onSuccess: (_, variables) => {
      // Remove block from cache
      queryClient.removeQueries({ queryKey: blockKeys.detail(Number(variables.id)) });
      // Invalidate block list (conversationId unknown, invalidate all)
      queryClient.invalidateQueries({ queryKey: blockKeys.lists() });
    },
  });
}

/**
 * Hook to append user input to a block
 */
export function useAppendUserInput() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: AppendUserInputRequest) => {
      const req = create(AppendUserInputRequestSchema, request as Record<string, unknown>);
      const response = await aiServiceClient.appendUserInput(req);
      return response;
    },
    onSuccess: (_, variables) => {
      // Invalidate block cache
      queryClient.invalidateQueries({ queryKey: blockKeys.detail(Number(variables.id)) });
      // Invalidate block list
      queryClient.invalidateQueries({ queryKey: blockKeys.lists() });
    },
  });
}

/**
 * Hook to append event to a block (optimized for streaming)
 *
 * This hook is optimized for high-frequency calls during streaming.
 * It uses direct cache manipulation instead of invalidation for better performance.
 */
export function useAppendEvent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: AppendEventRequest) => {
      const req = create(AppendEventRequestSchema, request as Record<string, unknown>);
      const response = await aiServiceClient.appendEvent(req);
      return response;
    },
    onSuccess: (_, variables) => {
      const blockId = Number(variables.id);

      // Direct cache update instead of invalidation (faster)
      queryClient.setQueryData(blockKeys.detail(blockId), (old: Block | undefined) => {
        if (!old) return old;

        // Append event to existing event stream
        const existingStream = old.eventStream || [];
        const newEvent = variables.event;

        return {
          ...old,
          eventStream: [...existingStream, newEvent],
          updatedTs: BigInt(Date.now()),
        };
      });
    },
  });
}

/**
 * Hook to append multiple events at once (batch append)
 *
 * More efficient than multiple useAppendEvent calls.
 */
export function useAppendEventsBatch() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (events: Array<{ blockId: number; event: AppendEventRequest }>) => {
      // Process all events
      const results = await Promise.allSettled(
        events.map(({ event }) => aiServiceClient.appendEvent(create(AppendEventRequestSchema, event as Record<string, unknown>))),
      );
      return results;
    },
    onSuccess: (_, variables) => {
      // Group by blockId for cache updates
      // biome-ignore lint/suspicious/noExplicitAny: Event array for batch processing
      const updatesByBlock = new Map<number, any[]>();

      for (const { event } of variables) {
        const blockId = Number(event.id);
        if (!updatesByBlock.has(blockId)) {
          updatesByBlock.set(blockId, []);
        }
        updatesByBlock.get(blockId)!.push(event.event);
      }

      // Update caches for all affected blocks
      for (const [blockId, events] of updatesByBlock) {
        queryClient.setQueryData(blockKeys.detail(blockId), (old: Block | undefined) => {
          if (!old) return old;
          return {
            ...old,
            eventStream: [...(old.eventStream || []), ...events],
            updatedTs: BigInt(Date.now()),
          };
        });
      }
    },
  });
}

// ============================================================================
// Streaming Hooks
// ============================================================================

/**
 * Hook for managing streaming block updates
 *
 * Handles optimistic updates during AI streaming without
 * waiting for server confirmations.
 */
export function useStreamingBlock(blockId: number) {
  const queryClient = useQueryClient();

  const updateStreamingContent = (content: string) => {
    queryClient.setQueryData(blockKeys.detail(blockId), (old: Block | undefined) => {
      if (!old) return old;
      return {
        ...old,
        assistantContent: content,
        status: BlockStatus.STREAMING,
        updatedTs: BigInt(Date.now()),
      };
    });
  };

  // Event type from streaming is JSON string
  const appendStreamingEvent = (event: string) => {
    queryClient.setQueryData(blockKeys.detail(blockId), (old: Block | undefined) => {
      if (!old) return old;
      return {
        ...old,
        eventStream: [...(old.eventStream || []), event],
        updatedTs: BigInt(Date.now()),
      };
    });
  };

  // SessionStats is optional JSON string
  const completeStreaming = (finalContent: string, sessionStats?: string) => {
    queryClient.setQueryData(blockKeys.detail(blockId), (old: Block | undefined) => {
      if (!old) return old;
      return {
        ...old,
        assistantContent: finalContent,
        status: BlockStatus.COMPLETED,
        sessionStats: sessionStats || old.sessionStats,
        updatedTs: BigInt(Date.now()),
      };
    });
  };

  const markStreamingError = (errorMessage: string) => {
    queryClient.setQueryData(blockKeys.detail(blockId), (old: Block | undefined) => {
      if (!old) return old;
      return {
        ...old,
        status: BlockStatus.ERROR,
        errorMessage,
        updatedTs: BigInt(Date.now()),
      };
    });
  };

  return {
    updateStreamingContent,
    appendStreamingEvent,
    completeStreaming,
    markStreamingError,
  };
}

// ============================================================================
// Prefetch Hooks
// ============================================================================

/**
 * Hook to prefetch a single block
 *
 * Use this to preload block data when hovering over a block reference.
 */
export function usePrefetchBlock() {
  const queryClient = useQueryClient();

  const prefetchBlock = async (id: number) => {
    await queryClient.prefetchQuery({
      queryKey: blockKeys.detail(id),
      queryFn: async () => {
        const request = create(GetBlockRequestSchema, { id: BigInt(id) });
        return aiServiceClient.getBlock(request);
      },
      staleTime: STALE_TIMES.BLOCK_DETAIL,
    });
  };

  return { prefetchBlock };
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Convert frontend BlockMode to proto BlockMode
 */
export function toProtoBlockMode(mode: "normal" | "geek" | "evolution"): BlockMode {
  switch (mode) {
    case "normal":
      return BlockMode.NORMAL;
    case "geek":
      return BlockMode.GEEK;
    case "evolution":
      return BlockMode.EVOLUTION;
    default:
      return BlockMode.UNSPECIFIED;
  }
}

/**
 * Convert frontend BlockType to proto BlockType
 */
export function toProtoBlockType(type: "message" | "context_separator"): BlockType {
  switch (type) {
    case "message":
      return BlockType.MESSAGE;
    case "context_separator":
      return BlockType.CONTEXT_SEPARATOR;
    default:
      return BlockType.UNSPECIFIED;
  }
}

/**
 * Convert proto BlockMode to frontend
 */
export function fromProtoBlockMode(mode: BlockMode): "normal" | "geek" | "evolution" {
  switch (mode) {
    case BlockMode.NORMAL:
      return "normal";
    case BlockMode.GEEK:
      return "geek";
    case BlockMode.EVOLUTION:
      return "evolution";
    default:
      return "normal";
  }
}

/**
 * Convert proto BlockStatus to frontend
 */
export function fromProtoBlockStatus(status: BlockStatus): "pending" | "streaming" | "completed" | "error" {
  switch (status) {
    case BlockStatus.PENDING:
      return "pending";
    case BlockStatus.STREAMING:
      return "streaming";
    case BlockStatus.COMPLETED:
      return "completed";
    case BlockStatus.ERROR:
      return "error";
    default:
      return "pending";
  }
}

// ============================================================================
// ERROR HANDLING & FALLBACK (Phase 4)
// ============================================================================

/**
 * Result type for useBlocksWithFallback - includes error state for fallback
 */
interface BlocksWithFallbackResult {
  blocks: Block[];
  isLoading: boolean;
  error: Error | null;
  shouldFallback: boolean;
  refetch: () => void; // Manual refetch function to refresh blocks
}

/**
 * Hook to fetch blocks with automatic fallback on error
 *
 * When Block API fails (network error, 404, etc.), this hook returns
 * an error state that signals the UI to fall back to ChatItem[].
 *
 * @param conversationId - The conversation ID to fetch blocks for
 * @param filters - Optional filters for the block list
 * @param options - Additional options like isActive (for active conversations)
 * @returns BlocksWithFallbackResult with blocks, loading state, and error info
 */
export function useBlocksWithFallback(
  conversationId: number,
  filters?: Partial<ListBlocksRequest>,
  options?: { isActive?: boolean },
): BlocksWithFallbackResult {
  const is_active = options?.isActive ?? false;

  const query = useQuery({
    queryKey: blockKeys.list(conversationId, filters),
    queryFn: async () => {
      const request = create(ListBlocksRequestSchema, {
        conversationId,
        ...filters,
      } as Record<string, unknown>);
      const response = await aiServiceClient.listBlocks(request);
      return response;
    },
    enabled: conversationId > 0,
    staleTime: is_active ? STALE_TIMES.ACTIVE_CONVERSATION : STALE_TIMES.BLOCK_LIST,
    gcTime: is_active ? CACHE_TIMES.ACTIVE_CONVERSATION : CACHE_TIMES.BLOCK_LIST,
    retry: (failureCount, error) => {
      const err = error as ApiError;
      return shouldRetryError(err, failureCount, RETRY_CONFIG.MAX_RETRIES);
    },
    retryDelay: createRetryDelay(RETRY_CONFIG.BASE_RETRY_DELAY, RETRY_CONFIG.MAX_RETRY_DELAY),
    refetchOnWindowFocus: is_active,
    refetchOnReconnect: true,
  });

  const blocks = query.data?.blocks || [];

  // Determine if we should fallback based on error state
  // Fallback conditions:
  // 1. Query failed and has error
  // 2. No blocks returned (might indicate API not available)
  const shouldFallback = query.isError || (query.isSuccess && blocks.length === 0 && conversationId > 0);

  return {
    blocks,
    isLoading: query.isLoading,
    error: query.error ?? null,
    shouldFallback,
    refetch: query.refetch, // Expose refetch function for manual refresh after streaming
  };
}

// ============================================================================
// FORK BLOCK (Branching Support)
// ============================================================================

/**
 * Hook to fork a block, creating a new branch
 *
 * @param blockId - The parent block ID to fork from
 * @param reason - Optional reason for forking
 * @param replaceUserInputs - Optional new user inputs to replace inherited ones (for message editing)
 * @returns Mutation with forked block data
 */
export function useForkBlock() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ blockId, reason, replaceUserInputs }: { blockId: bigint; reason?: string; replaceUserInputs?: UserInput[] }) => {
      const response = await aiServiceClient.forkBlock({
        id: blockId,
        reason,
        replaceUserInputs,
      });
      return response;
    },
    onSuccess: (newBlock, _variables) => {
      // Add the new forked block to cache
      const conversationId = Number(newBlock.conversationId);
      queryClient.setQueryData(blockKeys.detail(Number(newBlock.id)), newBlock);

      // Invalidate the block list to show the new branch
      queryClient.invalidateQueries({
        queryKey: blockKeys.list(conversationId),
      });
    },
  });
}

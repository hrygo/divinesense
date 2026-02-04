/**
 * Block API Hooks for Unified Block Model
 *
 * Provides React Query hooks for interacting with the Block API.
 *
 * Phase 4: Frontend Block hooks
 * @see docs/specs/unified-block-model.md
 */

import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { aiServiceClient } from "@/connect";
import {
  Block,
  BlockMode,
  BlockStatus,
  BlockType,
  ListBlocksRequestSchema,
  GetBlockRequestSchema,
  CreateBlockRequestSchema,
  UpdateBlockRequestSchema,
  DeleteBlockRequestSchema,
  AppendUserInputRequestSchema,
  AppendEventRequestSchema,
} from "@/types/proto/api/v1/ai_service_pb";
import type {
  ListBlocksRequest,
  GetBlockRequest,
  CreateBlockRequest,
  UpdateBlockRequest,
  DeleteBlockRequest,
  AppendUserInputRequest,
  AppendEventRequest,
} from "@/types/proto/api/v1/ai_service_pb";

// Query keys factory for consistent cache management
export const blockKeys = {
  all: ["blocks"] as const,
  lists: () => [...blockKeys.all, "list"] as const,
  list: (conversationId: number, filters?: Partial<ListBlocksRequest>) =>
    [...blockKeys.lists(), conversationId, filters] as const,
  details: () => [...blockKeys.all, "detail"] as const,
  detail: (id: number) => [...blockKeys.details(), id] as const,
};

/**
 * Hook to fetch blocks for a conversation
 */
export function useBlocks(conversationId: number, filters?: Partial<ListBlocksRequest>) {
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
    staleTime: 1000 * 30, // 30 seconds
  });
}

/**
 * Hook to fetch a single block by ID
 */
export function useBlock(id: number, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: blockKeys.detail(id),
    queryFn: async () => {
      const request = create(GetBlockRequestSchema, { id });
      const response = await aiServiceClient.getBlock(request);
      return response;
    },
    enabled: options?.enabled ?? id > 0,
    staleTime: 1000 * 10, // 10 seconds
  });
}

/**
 * Hook to create a new block
 */
export function useCreateBlock() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: CreateBlockRequest) => {
      const req = create(CreateBlockRequestSchema, request as Record<string, unknown>);
      const response = await aiServiceClient.createBlock(req);
      return response;
    },
    onSuccess: (newBlock, variables) => {
      // Invalidate block list for this conversation
      queryClient.invalidateQueries({
        queryKey: blockKeys.list(variables.conversationId),
      });
      // Add new block to cache
      queryClient.setQueryData(blockKeys.detail(newBlock.id), newBlock);
    },
  });
}

/**
 * Hook to update a block
 */
export function useUpdateBlock() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: UpdateBlockRequest) => {
      const req = create(UpdateBlockRequestSchema, request as Record<string, unknown>);
      const response = await aiServiceClient.updateBlock(req);
      return response;
    },
    onSuccess: (updatedBlock) => {
      // Update block in cache
      queryClient.setQueryData(blockKeys.detail(updatedBlock.id), updatedBlock);
      // Invalidate block list (conversationId unknown from response, invalidate all)
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
      queryClient.removeQueries({ queryKey: blockKeys.detail(variables.id) });
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
      queryClient.invalidateQueries({ queryKey: blockKeys.detail(variables.id) });
      // Invalidate block list
      queryClient.invalidateQueries({ queryKey: blockKeys.lists() });
    },
  });
}

/**
 * Hook to append event to a block
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
      // Invalidate block cache
      queryClient.invalidateQueries({ queryKey: blockKeys.detail(variables.id) });
      // Invalidate block list
      queryClient.invalidateQueries({ queryKey: blockKeys.lists() });
    },
  });
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
      return BlockMode.BLOCK_MODE_NORMAL;
    case "geek":
      return BlockMode.BLOCK_MODE_GEEK;
    case "evolution":
      return BlockMode.BLOCK_MODE_EVOLUTION;
    default:
      return BlockMode.BLOCK_MODE_UNSPECIFIED;
  }
}

/**
 * Convert frontend BlockType to proto BlockType
 */
export function toProtoBlockType(type: "message" | "context_separator"): BlockType {
  switch (type) {
    case "message":
      return BlockType.BLOCK_TYPE_MESSAGE;
    case "context_separator":
      return BlockType.BLOCK_TYPE_CONTEXT_SEPARATOR;
    default:
      return BlockType.BLOCK_TYPE_UNSPECIFIED;
  }
}

/**
 * Convert proto BlockMode to frontend
 */
export function fromProtoBlockMode(mode: BlockMode): "normal" | "geek" | "evolution" {
  switch (mode) {
    case BlockMode.BLOCK_MODE_NORMAL:
      return "normal";
    case BlockMode.BLOCK_MODE_GEEK:
      return "geek";
    case BlockMode.BLOCK_MODE_EVOLUTION:
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
    case BlockStatus.BLOCK_STATUS_PENDING:
      return "pending";
    case BlockStatus.BLOCK_STATUS_STREAMING:
      return "streaming";
    case BlockStatus.BLOCK_STATUS_COMPLETED:
      return "completed";
    case BlockStatus.BLOCK_STATUS_ERROR:
      return "error";
    default:
      return "pending";
  }
}

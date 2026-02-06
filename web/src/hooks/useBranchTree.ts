import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useState } from "react";
import { aiServiceClient } from "@/connect";
import { type BlockBranch } from "@/types/block";
import {
  DeleteBranchRequestSchema,
  ForkBlockRequestSchema,
  ListBlockBranchesRequestSchema,
  SwitchBranchRequestSchema,
} from "@/types/proto/api/v1/ai_service_pb";

interface UseBranchTreeOptions {
  conversationId: number;
  blockId?: number;
}

interface UseBranchTreeResult {
  branches: BlockBranch[];
  currentPath: string;
  isLoading: boolean;
  error: Error | null;
  isBranchSelectorOpen: boolean;
  openBranchSelector: () => void;
  closeBranchSelector: () => void;
  switchBranch: (path: string) => void;
  deleteBranch: (path: string, cascade?: boolean) => void;
  forkBlock: (reason: string) => void;
  refreshBranches: () => void;
  isSwitching: boolean;
  isDeleting: boolean;
  isForking: boolean;
}

/**
 * useBranchTree - Custom hook for managing conversation branching
 *
 * Provides:
 * - List branches for a block or conversation
 * - Switch to a different branch
 * - Delete a branch (with cascade option)
 * - Fork a new branch from a block
 * - UI state for branch selector modal
 */
export function useBranchTree({ conversationId, blockId }: UseBranchTreeOptions): UseBranchTreeResult {
  const queryClient = useQueryClient();
  const [isBranchSelectorOpen, setIsBranchSelectorOpen] = useState(false);
  const [currentPath, setCurrentPath] = useState<string>("");

  // Query for block branches
  const {
    data: branchData,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["ai", "branches", conversationId, blockId],
    queryFn: async () => {
      // If blockId is provided, list branches for that block
      // Otherwise list branches for the conversation
      if (blockId) {
        const request = create(ListBlockBranchesRequestSchema, {
          id: BigInt(blockId),
        });
        return await aiServiceClient.listBlockBranches(request);
      }
      // For conversation-level, we'd need a different API call
      // For now, return empty branches if no blockId
      return { branches: [], activeBranchPath: "" };
    },
    enabled: conversationId > 0,
  });

  const branches = branchData?.branches ?? [];
  const activePath = branchData?.activeBranchPath ?? "";

  // Update current path when data changes
  useEffect(() => {
    if (activePath && activePath !== currentPath) {
      setCurrentPath(activePath);
    }
  }, [activePath, currentPath]);

  // Switch branch mutation
  const switchMutation = useMutation({
    mutationFn: async (path: string) => {
      const request = create(SwitchBranchRequestSchema, {
        conversationId,
        targetBranchPath: path,
      });
      await aiServiceClient.switchBranch(request);
    },
    onSuccess: () => {
      // Invalidate blocks query to get updated branch state
      queryClient.invalidateQueries({
        queryKey: ["ai", "blocks", conversationId],
      });
      queryClient.invalidateQueries({
        queryKey: ["ai", "branches", conversationId, blockId],
      });
    },
  });

  // Delete branch mutation
  const deleteMutation = useMutation({
    mutationFn: async ({ path, cascade }: { path: string; cascade?: boolean }) => {
      const request = create(DeleteBranchRequestSchema, {
        id: branches.find((b) => b.branchPath === path)?.block?.id ?? 0n,
        cascade: cascade ?? false,
      });
      await aiServiceClient.deleteBranch(request);
    },
    onSuccess: () => {
      // Invalidate blocks and branches queries
      queryClient.invalidateQueries({
        queryKey: ["ai", "blocks", conversationId],
      });
      queryClient.invalidateQueries({
        queryKey: ["ai", "branches", conversationId, blockId],
      });
    },
  });

  // Fork block mutation
  const forkMutation = useMutation({
    mutationFn: async (reason: string) => {
      if (!blockId) {
        throw new Error("Cannot fork without a blockId");
      }
      const request = create(ForkBlockRequestSchema, {
        id: BigInt(blockId),
        reason,
      });
      return await aiServiceClient.forkBlock(request);
    },
    onSuccess: () => {
      // Invalidate blocks and branches queries
      queryClient.invalidateQueries({
        queryKey: ["ai", "blocks", conversationId],
      });
      queryClient.invalidateQueries({
        queryKey: ["ai", "branches", conversationId, blockId],
      });
    },
  });

  const switchBranch = useCallback(
    (path: string) => {
      switchMutation.mutate(path);
    },
    [switchMutation],
  );

  const deleteBranch = useCallback(
    (path: string, cascade = false) => {
      deleteMutation.mutate({ path, cascade });
    },
    [deleteMutation],
  );

  const forkBlock = useCallback(
    (reason: string) => {
      forkMutation.mutate(reason);
    },
    [forkMutation],
  );

  const refreshBranches = useCallback(() => {
    refetch();
  }, [refetch]);

  const openBranchSelector = useCallback(() => {
    setIsBranchSelectorOpen(true);
  }, []);

  const closeBranchSelector = useCallback(() => {
    setIsBranchSelectorOpen(false);
  }, []);

  return {
    branches,
    currentPath: activePath,
    isLoading,
    error,
    isBranchSelectorOpen,
    openBranchSelector,
    closeBranchSelector,
    switchBranch,
    deleteBranch,
    forkBlock,
    refreshBranches,
    isSwitching: switchMutation.isPending,
    isDeleting: deleteMutation.isPending,
    isForking: forkMutation.isPending,
  };
}

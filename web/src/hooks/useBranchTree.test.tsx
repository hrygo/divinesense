/**
 * useBranchTree Hook Tests
 *
 * Tests for conversation branching functionality including:
 * - List branches for a block/conversation
 * - Switch to a different branch
 * - Delete a branch (with cascade option)
 * - Fork a new block from a block
 * - UI state management for branch selector
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { BlockBranch } from "@/types/block";
import { useBranchTree } from "./useBranchTree";

// Mock the aiServiceClient
vi.mock("@/connect", () => ({
  aiServiceClient: {
    listBlockBranches: vi.fn(),
    switchBranch: vi.fn(),
    deleteBranch: vi.fn(),
    forkBlock: vi.fn(),
  },
}));

const { aiServiceClient } = await import("@/connect");

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("useBranchTree", () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createWrapper>;

  const mockBranches = [
    {
      $typeName: "memos.api.v1.BlockBranch" as const,
      block: {
        $typeName: "memos.api.v1.Block" as const,
        id: 1n,
        uid: "block-1",
        conversationId: 123,
        roundNumber: 1,
        mode: 0,
        blockType: 0,
        userInputs: [{ $typeName: "memos.api.v1.UserInput" as const, content: "Original", timestamp: 1000n, metadata: "{}" }],
        assistantContent: "Response",
        eventStream: [],
        status: 2,
        metadata: "{}",
        createdTs: 1000n,
        updatedTs: 2000n,
        assistantTimestamp: 1500n,
        ccSessionId: "",
        parentBlockId: 0n,
        branchPath: "0",
        costEstimate: 1000n,
        modelVersion: "deepseek-chat",
        userFeedback: "",
        regenerationCount: 0,
        errorMessage: "",
        archivedAt: 0n,
        sessionStats: undefined,
        tokenUsage: undefined,
      },
      branchPath: "0",
      isActive: true,
      children: [
        {
          $typeName: "memos.api.v1.BlockBranch" as const,
          block: {
            $typeName: "memos.api.v1.Block" as const,
            id: 2n,
            uid: "block-2",
            conversationId: 123,
            roundNumber: 2,
            mode: 0,
            blockType: 0,
            userInputs: [{ $typeName: "memos.api.v1.UserInput" as const, content: "Forked", timestamp: 2000n, metadata: "{}" }],
            assistantContent: "Forked response",
            eventStream: [],
            status: 2,
            metadata: "{}",
            createdTs: 2000n,
            updatedTs: 3000n,
            assistantTimestamp: 2500n,
            ccSessionId: "",
            parentBlockId: 1n,
            branchPath: "0/1",
            costEstimate: 500n,
            modelVersion: "deepseek-chat",
            userFeedback: "",
            regenerationCount: 0,
            errorMessage: "",
            archivedAt: 0n,
            sessionStats: undefined,
            tokenUsage: undefined,
          },
          branchPath: "0/1",
          isActive: false,
          children: [],
        },
      ],
    },
  ] as BlockBranch[];

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createWrapper(queryClient);
    vi.clearAllMocks();
  });

  afterEach(() => {
    queryClient.clear();
  });

  it("should fetch branches for a block", async () => {
    vi.mocked(aiServiceClient.listBlockBranches).mockResolvedValue({
      branches: mockBranches,
      activeBranchPath: "0",
    });

    const { result } = renderHook(
      () => useBranchTree({ conversationId: 123, blockId: 1 }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.branches).toEqual(mockBranches);
    expect(result.current.currentPath).toBe("0");
    expect(aiServiceClient.listBlockBranches).toHaveBeenCalledWith(
      expect.objectContaining({ id: 1n }),
    );
  });

  it("should return empty branches when no blockId provided", async () => {
    const { result } = renderHook(
      () => useBranchTree({ conversationId: 123 }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.branches).toEqual([]);
    expect(result.current.currentPath).toBe("");
  });

  it("should be disabled when conversationId is 0", () => {
    const { result } = renderHook(
      () => useBranchTree({ conversationId: 0, blockId: 1 }),
      { wrapper },
    );

    expect(result.current.isLoading).toBe(false);
    expect(aiServiceClient.listBlockBranches).not.toHaveBeenCalled();
  });

  it("should open and close branch selector", () => {
    vi.mocked(aiServiceClient.listBlockBranches).mockResolvedValue({
      branches: [],
      activeBranchPath: "",
    });

    const { result } = renderHook(
      () => useBranchTree({ conversationId: 123, blockId: 1 }),
      { wrapper },
    );

    expect(result.current.isBranchSelectorOpen).toBe(false);

    result.current.openBranchSelector();
    expect(result.current.isBranchSelectorOpen).toBe(true);

    result.current.closeBranchSelector();
    expect(result.current.isBranchSelectorOpen).toBe(false);
  });

  it("should switch branch", async () => {
    vi.mocked(aiServiceClient.listBlockBranches).mockResolvedValue({
      branches: mockBranches,
      activeBranchPath: "0",
    });

    vi.mocked(aiServiceClient.switchBranch).mockResolvedValue({});

    const { result } = renderHook(
      () => useBranchTree({ conversationId: 123, blockId: 1 }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    result.current.switchBranch("0/1");

    await waitFor(() => expect(result.current.isSwitching).toBe(true));

    expect(aiServiceClient.switchBranch).toHaveBeenCalledWith(
      expect.objectContaining({
        conversationId: 123,
        targetBranchPath: "0/1",
      }),
    );
  });

  it("should delete branch", async () => {
    vi.mocked(aiServiceClient.listBlockBranches).mockResolvedValue({
      branches: mockBranches,
      activeBranchPath: "0",
    });

    vi.mocked(aiServiceClient.deleteBranch).mockResolvedValue({});

    const { result } = renderHook(
      () => useBranchTree({ conversationId: 123, blockId: 1 }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    result.current.deleteBranch("0/1", false);

    await waitFor(() => expect(result.current.isDeleting).toBe(true));

    expect(aiServiceClient.deleteBranch).toHaveBeenCalled();
  });

  it("should fork block with reason", async () => {
    vi.mocked(aiServiceClient.listBlockBranches).mockResolvedValue({
      branches: mockBranches,
      activeBranchPath: "0",
    });

    const forkedBlock = {
      id: 3n,
      uid: "block-3",
      conversationId: 123,
      roundNumber: 2,
      mode: 0,
      blockType: 0,
      userInputs: [{ content: "Original", timestamp: 1000n }],
      assistantContent: "",
      eventStream: [],
      status: 0,
      metadata: "{}",
      createdTs: BigInt(Date.now()),
      updatedTs: BigInt(Date.now()),
      assistantTimestamp: BigInt(Date.now()),
      ccSessionId: "",
      parentBlockId: 1n,
      branchPath: "0/2",
      costEstimate: 0n,
      modelVersion: "",
      userFeedback: "",
      regenerationCount: 0,
      errorMessage: "",
      archivedAt: 0n,
    };

    vi.mocked(aiServiceClient.forkBlock).mockResolvedValue(forkedBlock);

    const { result } = renderHook(
      () => useBranchTree({ conversationId: 123, blockId: 1 }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    result.current.forkBlock("Trying a different approach");

    await waitFor(() => expect(result.current.isForking).toBe(true));

    expect(aiServiceClient.forkBlock).toHaveBeenCalledWith(
      expect.objectContaining({
        id: 1n,
        reason: "Trying a different approach",
      }),
    );
  });

  it("should throw error when forking without blockId", async () => {
    const { result } = renderHook(
      () => useBranchTree({ conversationId: 123 }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(() => result.current.forkBlock("Test")).toThrow("Cannot fork without a blockId");
  });

  it("should refresh branches", async () => {
    vi.mocked(aiServiceClient.listBlockBranches)
      .mockResolvedValueOnce({
        branches: mockBranches,
        activeBranchPath: "0",
      })
      .mockResolvedValueOnce({
        branches: [...mockBranches, mockBranches[0]],
        activeBranchPath: "0",
      });

    const { result } = renderHook(
      () => useBranchTree({ conversationId: 123, blockId: 1 }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(aiServiceClient.listBlockBranches).toHaveBeenCalledTimes(1);

    result.current.refreshBranches();

    await waitFor(() =>
      expect(aiServiceClient.listBlockBranches).toHaveBeenCalledTimes(2),
    );
  });
});

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

import { create } from "@bufbuild/protobuf";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { act } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Block, BlockBranch, ListBlockBranchesResponse } from "@/types/proto/api/v1/ai_service_pb";
import {
  BlockBranchSchema,
  BlockMode,
  BlockSchema,
  BlockStatus,
  BlockType,
  ListBlockBranchesResponseSchema,
  UserInputSchema,
} from "@/types/proto/api/v1/ai_service_pb";
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

/**
 * Helper to create a mock Block with proper protobuf structure
 */
function createMockBlock(overrides: Record<string, unknown> = {}): Block {
  return create(BlockSchema, {
    id: 1n,
    uid: "block-1",
    conversationId: 123,
    roundNumber: 1,
    mode: BlockMode.NORMAL,
    blockType: BlockType.MESSAGE,
    userInputs: [],
    assistantContent: "",
    eventStream: [],
    status: BlockStatus.COMPLETED,
    metadata: "{}",
    createdTs: 1000n,
    updatedTs: 2000n,
    assistantTimestamp: 1500n,
    ccSessionId: "",
    parentBlockId: 0n,
    branchPath: "",
    costEstimate: 1000n,
    modelVersion: "deepseek-chat",
    userFeedback: "",
    regenerationCount: 0,
    errorMessage: "",
    archivedAt: 0n,
    ...overrides,
  });
}

/**
 * Helper to create a mock BlockBranch
 */
function createMockBranch(block: Block, branchPath: string, isActive: boolean, children: BlockBranch[] = []): BlockBranch {
  return create(BlockBranchSchema, {
    block,
    branchPath,
    isActive,
    children,
  });
}

/**
 * Helper to create a mock ListBlockBranchesResponse
 */
function createMockListBranchesResponse(branches: BlockBranch[], activeBranchPath: string): ListBlockBranchesResponse {
  return create(ListBlockBranchesResponseSchema, {
    branches,
    activeBranchPath,
  });
}

describe("useBranchTree", () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createWrapper>;

  const mockBranches = [
    createMockBranch(
      createMockBlock({
        id: 1n,
        userInputs: [create(UserInputSchema, { content: "Original", timestamp: 1000n, metadata: "{}" })],
        assistantContent: "Response",
        branchPath: "0",
      }),
      "0",
      true,
      [
        createMockBranch(
          createMockBlock({
            id: 2n,
            userInputs: [create(UserInputSchema, { content: "Forked", timestamp: 2000n, metadata: "{}" })],
            assistantContent: "Forked response",
            branchPath: "0/1",
            parentBlockId: 1n,
            costEstimate: 500n,
          }),
          "0/1",
          false,
        ),
      ],
    ),
  ];

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createWrapper(queryClient);
    vi.clearAllMocks();
  });

  afterEach(() => {
    queryClient.clear();
  });

  it("should fetch branches for a block", async () => {
    vi.mocked(aiServiceClient.listBlockBranches).mockResolvedValue(createMockListBranchesResponse(mockBranches, "0"));

    const { result } = renderHook(() => useBranchTree({ conversationId: 123, blockId: 1 }), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.branches).toEqual(mockBranches);
    expect(result.current.currentPath).toBe("0");
    expect(aiServiceClient.listBlockBranches).toHaveBeenCalledWith(expect.objectContaining({ id: 1n }));
  });

  it("should return empty branches when no blockId provided", async () => {
    const { result } = renderHook(() => useBranchTree({ conversationId: 123 }), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.branches).toEqual([]);
    expect(result.current.currentPath).toBe("");
  });

  it("should be disabled when conversationId is 0", () => {
    const { result } = renderHook(() => useBranchTree({ conversationId: 0, blockId: 1 }), { wrapper });

    expect(result.current.isLoading).toBe(false);
    expect(aiServiceClient.listBlockBranches).not.toHaveBeenCalled();
  });

  it("should open and close branch selector", async () => {
    vi.mocked(aiServiceClient.listBlockBranches).mockResolvedValue(createMockListBranchesResponse([], ""));

    const { result } = renderHook(() => useBranchTree({ conversationId: 123, blockId: 1 }), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.isBranchSelectorOpen).toBe(false);

    act(() => {
      result.current.openBranchSelector();
    });
    expect(result.current.isBranchSelectorOpen).toBe(true);

    act(() => {
      result.current.closeBranchSelector();
    });
    expect(result.current.isBranchSelectorOpen).toBe(false);
  });

  it("should switch branch", async () => {
    vi.mocked(aiServiceClient.listBlockBranches).mockResolvedValue(createMockListBranchesResponse(mockBranches, "0"));

    // Use a delayed promise to ensure isPending stays true long enough to test
    let resolveSwitch: () => void;
    const switchPromise = new Promise<void>((resolve) => {
      resolveSwitch = resolve;
    });
    // biome-ignore lint/suspicious/noExplicitAny: Test mock for complex Promise type
    vi.mocked(aiServiceClient.switchBranch).mockReturnValue(switchPromise as any);

    const { result } = renderHook(() => useBranchTree({ conversationId: 123, blockId: 1 }), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => {
      result.current.switchBranch("0/1");
    });

    // isSwitching should be true while mutation is in progress
    await waitFor(() => expect(result.current.isSwitching).toBe(true));

    // Resolve the mutation
    act(() => {
      resolveSwitch!();
    });

    // Then wait for mutation to complete
    await waitFor(() => expect(result.current.isSwitching).toBe(false));
  });

  it("should delete branch", async () => {
    vi.mocked(aiServiceClient.listBlockBranches).mockResolvedValue(createMockListBranchesResponse(mockBranches, "0"));

    // Use a delayed promise to ensure isPending stays true long enough to test
    let resolveDelete: () => void;
    const deletePromise = new Promise<void>((resolve) => {
      resolveDelete = resolve;
    });
    // biome-ignore lint/suspicious/noExplicitAny: Test mock for complex Promise type
    vi.mocked(aiServiceClient.deleteBranch).mockReturnValue(deletePromise as any);

    const { result } = renderHook(() => useBranchTree({ conversationId: 123, blockId: 1 }), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => {
      result.current.deleteBranch("0/1", false);
    });

    // isDeleting should be true while mutation is in progress
    await waitFor(() => expect(result.current.isDeleting).toBe(true));

    // Resolve the mutation
    act(() => {
      resolveDelete!();
    });

    // Then wait for mutation to complete
    await waitFor(() => expect(result.current.isDeleting).toBe(false));
  });

  it("should fork block with reason", async () => {
    vi.mocked(aiServiceClient.listBlockBranches).mockResolvedValue(createMockListBranchesResponse(mockBranches, "0"));

    const forkedBlock = createMockBlock({
      id: 3n,
      uid: "block-3",
      userInputs: [create(UserInputSchema, { content: "Original", timestamp: 1000n, metadata: "{}" })],
      branchPath: "0/2",
      parentBlockId: 1n,
    });

    // Use a delayed promise to ensure isPending stays true long enough to test
    let resolveFork: () => void;
    const forkPromise = new Promise<typeof forkedBlock>((resolve) => {
      resolveFork = () => resolve(forkedBlock);
    });
    vi.mocked(aiServiceClient.forkBlock).mockReturnValue(forkPromise);

    const { result } = renderHook(() => useBranchTree({ conversationId: 123, blockId: 1 }), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => {
      result.current.forkBlock("Trying a different approach");
    });

    // isForking should be true while mutation is in progress
    await waitFor(() => expect(result.current.isForking).toBe(true));

    // Resolve the mutation
    act(() => {
      resolveFork!();
    });

    // Then wait for mutation to complete
    await waitFor(() => expect(result.current.isForking).toBe(false));
  });

  it("should handle error when forking without blockId", async () => {
    // The hook's forkBlock mutation handles the error internally
    // When blockId is undefined, the mutationFn throws and sets error state
    const { result } = renderHook(() => useBranchTree({ conversationId: 123 }), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    // Call forkBlock - mutation will fail but won't throw synchronously
    act(() => {
      result.current.forkBlock("Test");
    });

    // The mutation should error (mutationFn throws in the hook)
    await waitFor(() => expect(result.current.isForking).toBe(false));
  });

  it("should refresh branches", async () => {
    vi.mocked(aiServiceClient.listBlockBranches)
      .mockResolvedValueOnce(createMockListBranchesResponse(mockBranches, "0"))
      .mockResolvedValueOnce(createMockListBranchesResponse([...mockBranches, mockBranches[0]], "0"));

    const { result } = renderHook(() => useBranchTree({ conversationId: 123, blockId: 1 }), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(aiServiceClient.listBlockBranches).toHaveBeenCalledTimes(1);

    result.current.refreshBranches();

    await waitFor(() => expect(aiServiceClient.listBlockBranches).toHaveBeenCalledTimes(2));
  });
});

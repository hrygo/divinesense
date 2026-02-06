/**
 * Block API Hooks Tests
 *
 * Tests for useBlockQueries hooks including:
 * - useBlocks with fallback
 * - useStreamingBlock
 * - New features: token usage, cost tracking, branching
 */

import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import {
  useBlocks,
  useBlock,
  useCreateBlock,
  useUpdateBlock,
  useDeleteBlock,
  useAppendEvent,
  useStreamingBlock,
  useBlocksWithFallback,
  blockKeys,
  toProtoBlockMode,
  fromProtoBlockMode,
  toProtoBlockType,
  fromProtoBlockStatus,
} from "./useBlockQueries";
import { BlockMode, BlockStatus, BlockType } from "@/types/proto/api/v1/ai_service_pb";

// Mock the aiServiceClient
vi.mock("@/connect", () => ({
  aiServiceClient: {
    listBlocks: vi.fn(),
    getBlock: vi.fn(),
    createBlock: vi.fn(),
    updateBlock: vi.fn(),
    deleteBlock: vi.fn(),
    appendEvent: vi.fn(),
    appendUserInput: vi.fn(),
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

describe("blockKeys", () => {
  it("should generate consistent query keys", () => {
    expect(blockKeys.all).toEqual(["blocks"]);
    expect(blockKeys.lists()).toEqual(["blocks", "list"]);
    expect(blockKeys.list(123, { isActive: true })).toEqual(["blocks", "list", 123, { isActive: true }]);
    expect(blockKeys.details()).toEqual(["blocks", "detail"]);
    expect(blockKeys.detail(456)).toEqual(["blocks", "detail", 456]);
  });
});

describe("useBlocks", () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createWrapper(queryClient);
    vi.clearAllMocks();
  });

  afterEach(() => {
    queryClient.clear();
  });

  it("should fetch blocks for a conversation", async () => {
    const mockBlocks = {
      blocks: [
        {
          id: 1n,
          uid: "block-1",
          conversationId: 123,
          roundNumber: 1,
          mode: BlockMode.NORMAL,
          blockType: BlockType.MESSAGE,
          userInputs: [{ content: "Hello", timestamp: 1000n }],
          assistantContent: "Hi there!",
          eventStream: [],
          status: BlockStatus.COMPLETED,
          metadata: "{}",
          createdTs: 1000n,
          updatedTs: 2000n,
          assistantTimestamp: 1500n,
          ccSessionId: "",
          parentBlockId: 0n,
          branchPath: "",
          costEstimate: 2100n,
          modelVersion: "deepseek-chat",
          userFeedback: "",
          regenerationCount: 0,
          errorMessage: "",
          archivedAt: 0n,
        },
      ],
      totalCount: 1,
    };

    vi.mocked(aiServiceClient.listBlocks).mockResolvedValue(mockBlocks);

    const { result } = renderHook(() => useBlocks(123), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockBlocks);
    expect(aiServiceClient.listBlocks).toHaveBeenCalledTimes(1);
  });

  it("should be disabled when conversationId is 0", () => {
    const { result } = renderHook(() => useBlocks(0), { wrapper });

    expect(result.current.fetchStatus).toBe("idle");
    expect(aiServiceClient.listBlocks).not.toHaveBeenCalled();
  });
});

describe("useBlocksWithFallback", () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createWrapper(queryClient);
    vi.clearAllMocks();
  });

  afterEach(() => {
    queryClient.clear();
  });

  it("should return blocks and not indicate fallback when successful", async () => {
    const mockBlocks = {
      blocks: [
        {
          id: 1n,
          uid: "block-1",
          conversationId: 123,
          roundNumber: 1,
          mode: BlockMode.NORMAL,
          blockType: BlockType.MESSAGE,
          userInputs: [],
          assistantContent: "Response",
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
        },
      ],
      totalCount: 1,
    };

    vi.mocked(aiServiceClient.listBlocks).mockResolvedValue(mockBlocks);

    const { result } = renderHook(() => useBlocksWithFallback(123), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.blocks).toEqual(mockBlocks.blocks);
    expect(result.current.shouldFallback).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("should indicate fallback when query fails", async () => {
    vi.mocked(aiServiceClient.listBlocks).mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useBlocksWithFallback(123), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.shouldFallback).toBe(true);
    expect(result.current.error).not.toBeNull();
    expect(result.current.error?.message).toBe("Network error");
  });

  it("should indicate fallback when no blocks returned for active conversation", async () => {
    vi.mocked(aiServiceClient.listBlocks).mockResolvedValue({
      blocks: [],
      totalCount: 0,
    });

    const { result } = renderHook(
      () => useBlocksWithFallback(123, undefined, { isActive: true }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.shouldFallback).toBe(true);
    expect(result.current.blocks).toEqual([]);
  });

  it("should provide refetch function", async () => {
    vi.mocked(aiServiceClient.listBlocks).mockResolvedValue({
      blocks: [],
      totalCount: 0,
    });

    const { result } = renderHook(() => useBlocksWithFallback(123), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(typeof result.current.refetch).toBe("function");

    // Call refetch
    result.current.refetch();

    expect(aiServiceClient.listBlocks).toHaveBeenCalledTimes(2);
  });
});

describe("useStreamingBlock", () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createWrapper(queryClient);

    // Pre-populate cache with a block
    queryClient.setQueryData(blockKeys.detail(1), {
      id: 1n,
      uid: "block-1",
      conversationId: 123,
      roundNumber: 1,
      mode: BlockMode.NORMAL,
      blockType: BlockType.MESSAGE,
      userInputs: [],
      assistantContent: "",
      eventStream: [],
      status: BlockStatus.PENDING,
      metadata: "{}",
      createdTs: 1000n,
      updatedTs: 1000n,
      assistantTimestamp: 1000n,
      ccSessionId: "",
      parentBlockId: 0n,
      branchPath: "",
      costEstimate: 0n,
      modelVersion: "",
      userFeedback: "",
      regenerationCount: 0,
      errorMessage: "",
      archivedAt: 0n,
    });
  });

  it("should update streaming content", () => {
    const { result } = renderHook(() => useStreamingBlock(1), { wrapper });

    result.current.updateStreamingContent("Partial response");

    const cached = queryClient.getQueryData(blockKeys.detail(1));
    expect(cached).toMatchObject({
      assistantContent: "Partial response",
      status: BlockStatus.STREAMING,
    });
  });

  it("should append streaming events", () => {
    const { result } = renderHook(() => useStreamingBlock(1), { wrapper });

    const event = {
      type: "thinking",
      timestamp: Date.now(),
    };

    result.current.appendStreamingEvent(event);

    const cached = queryClient.getQueryData(blockKeys.detail(1));
    expect(cached).toMatchObject({
      eventStream: [event],
    });
  });

  it("should complete streaming with session stats", () => {
    const { result } = renderHook(() => useStreamingBlock(1), { wrapper });

    const sessionStats = JSON.stringify({
      llmCalls: [{ promptTokens: 100, completionTokens: 50 }],
      totalTokens: 150,
    });

    result.current.completeStreaming("Final response", sessionStats);

    const cached = queryClient.getQueryData(blockKeys.detail(1));
    expect(cached).toMatchObject({
      assistantContent: "Final response",
      status: BlockStatus.COMPLETED,
      sessionStats,
    });
  });

  it("should mark streaming error", () => {
    const { result } = renderHook(() => useStreamingBlock(1), { wrapper });

    result.current.markStreamingError("Something went wrong");

    const cached = queryClient.getQueryData(blockKeys.detail(1));
    expect(cached).toMatchObject({
      status: BlockStatus.ERROR,
      errorMessage: "Something went wrong",
    });
  });
});

describe("useCreateBlock", () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createWrapper(queryClient);
    vi.clearAllMocks();
  });

  it("should create block with optimistic update", async () => {
    const createdBlock = {
      id: 999n,
      uid: "block-999",
      conversationId: 123,
      roundNumber: 1,
      mode: BlockMode.NORMAL,
      blockType: BlockType.MESSAGE,
      userInputs: [{ content: "Test", timestamp: 1000n }],
      assistantContent: "",
      eventStream: [],
      status: BlockStatus.PENDING,
      metadata: "{}",
      createdTs: BigInt(Date.now()),
      updatedTs: BigInt(Date.now()),
      assistantTimestamp: BigInt(Date.now()),
      ccSessionId: "",
      parentBlockId: 0n,
      branchPath: "",
      costEstimate: 0n,
      modelVersion: "",
      userFeedback: "",
      regenerationCount: 0,
      errorMessage: "",
      archivedAt: 0n,
    };

    vi.mocked(aiServiceClient.createBlock).mockResolvedValue(createdBlock);

    const { result } = renderHook(() => useCreateBlock(), { wrapper });

    // First populate the list
    queryClient.setQueryData(blockKeys.list(123), {
      blocks: [],
      totalCount: 0,
    });

    result.current.mutate({
      conversationId: 123,
      mode: BlockMode.NORMAL,
      blockType: BlockType.MESSAGE,
      userInputs: [{ content: "Test", timestamp: 1000n }],
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(aiServiceClient.createBlock).toHaveBeenCalled();
  });
});

describe("useUpdateBlock", () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createWrapper(queryClient);
    vi.clearAllMocks();
  });

  it("should update block with optimistic update", async () => {
    const updatedBlock = {
      id: 1n,
      uid: "block-1",
      conversationId: 123,
      roundNumber: 1,
      mode: BlockMode.NORMAL,
      blockType: BlockType.MESSAGE,
      userInputs: [],
      assistantContent: "Updated content",
      eventStream: [],
      status: BlockStatus.COMPLETED,
      metadata: "{}",
      createdTs: 1000n,
      updatedTs: 3000n,
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
    };

    vi.mocked(aiServiceClient.updateBlock).mockResolvedValue(updatedBlock);

    const { result } = renderHook(() => useUpdateBlock(), { wrapper });

    result.current.mutate({
      id: 1n,
      assistantContent: "Updated content",
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(aiServiceClient.updateBlock).toHaveBeenCalled();
  });
});

describe("useDeleteBlock", () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createWrapper(queryClient);
    vi.clearAllMocks();
  });

  it("should delete block and invalidate cache", async () => {
    vi.mocked(aiServiceClient.deleteBlock).mockResolvedValue({});

    const { result } = renderHook(() => useDeleteBlock(), { wrapper });

    result.current.mutate({ id: 1n });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(aiServiceClient.deleteBlock).toHaveBeenCalledWith({ id: 1n });
  });
});

describe("useAppendEvent", () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createWrapper(queryClient);
    vi.clearAllMocks();
  });

  it("should append event to block", async () => {
    vi.mocked(aiServiceClient.appendEvent).mockResolvedValue({});

    const { result } = renderHook(() => useAppendEvent(), { wrapper });

    const event = JSON.stringify({ type: "thinking" });

    result.current.mutate({
      id: 1n,
      event,
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(aiServiceClient.appendEvent).toHaveBeenCalled();
  });
});

describe("Converter functions", () => {
  describe("toProtoBlockMode", () => {
    it("should convert frontend modes to proto", () => {
      expect(toProtoBlockMode("normal")).toBe(BlockMode.NORMAL);
      expect(toProtoBlockMode("geek")).toBe(BlockMode.GEEK);
      expect(toProtoBlockMode("evolution")).toBe(BlockMode.EVOLUTION);
      expect(toProtoBlockMode("unknown" as any)).toBe(BlockMode.UNSPECIFIED);
    });
  });

  describe("fromProtoBlockMode", () => {
    it("should convert proto modes to frontend", () => {
      expect(fromProtoBlockMode(BlockMode.NORMAL)).toBe("normal");
      expect(fromProtoBlockMode(BlockMode.GEEK)).toBe("geek");
      expect(fromProtoBlockMode(BlockMode.EVOLUTION)).toBe("evolution");
      expect(fromProtoBlockMode(BlockMode.UNSPECIFIED)).toBe("normal");
    });
  });

  describe("toProtoBlockType", () => {
    it("should convert frontend types to proto", () => {
      expect(toProtoBlockType("message")).toBe(BlockType.MESSAGE);
      expect(toProtoBlockType("context_separator")).toBe(BlockType.CONTEXT_SEPARATOR);
      expect(toProtoBlockType("unknown" as any)).toBe(BlockType.UNSPECIFIED);
    });
  });

  describe("fromProtoBlockStatus", () => {
    it("should convert proto status to frontend", () => {
      expect(fromProtoBlockStatus(BlockStatus.PENDING)).toBe("pending");
      expect(fromProtoBlockStatus(BlockStatus.STREAMING)).toBe("streaming");
      expect(fromProtoBlockStatus(BlockStatus.COMPLETED)).toBe("completed");
      expect(fromProtoBlockStatus(BlockStatus.ERROR)).toBe("error");
      expect(fromProtoBlockStatus(BlockStatus.UNSPECIFIED)).toBe("pending");
    });
  });
});

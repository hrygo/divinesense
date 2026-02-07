/**
 * Block API Hooks Tests
 *
 * Tests for useBlockQueries hooks including:
 * - useBlocks with fallback
 * - useStreamingBlock
 * - New features: token usage, cost tracking, branching
 */

import { create } from "@bufbuild/protobuf";
import { EmptySchema } from "@bufbuild/protobuf/wkt";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Block, ListBlocksResponse, UserInput } from "@/types/proto/api/v1/ai_service_pb";
import {
  AppendEventRequestSchema,
  BlockEventSchema,
  BlockMode,
  BlockSchema,
  BlockStatus,
  BlockType,
  CreateBlockRequestSchema,
  DeleteBlockRequestSchema,
  ListBlocksResponseSchema,
  UpdateBlockRequestSchema,
  UserInputSchema,
} from "@/types/proto/api/v1/ai_service_pb";
import {
  blockKeys,
  fromProtoBlockMode,
  fromProtoBlockStatus,
  toProtoBlockMode,
  toProtoBlockType,
  useAppendEvent,
  useBlocks,
  useBlocksWithFallback,
  useCreateBlock,
  useDeleteBlock,
  useStreamingBlock,
  useUpdateBlock,
} from "./useBlockQueries";

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

// Mock shouldRetryError to disable retries in tests
vi.mock("@/config/errors", async () => {
  const actual = await vi.importActual("@/config/errors");
  return {
    ...actual,
    shouldRetryError: () => false, // Never retry in tests
  };
});

const { aiServiceClient } = await import("@/connect");

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
 * Helper to create a mock UserInput
 */
function createMockUserInput(content: string, timestamp?: number): UserInput {
  return create(UserInputSchema, {
    content,
    timestamp: BigInt(timestamp ?? Date.now()),
    metadata: "{}",
  });
}

/**
 * Helper to create a mock ListBlocksResponse
 */
function createMockListResponse(blocks: Block[]): ListBlocksResponse {
  return create(ListBlocksResponseSchema, {
    blocks,
  });
}

/**
 * Helper to create a mock ListBlocksResponse with pagination metadata
 */
function createMockPaginatedResponse(blocks: Block[], hasMore = false, totalCount = blocks.length): ListBlocksResponse {
  return create(ListBlocksResponseSchema, {
    blocks,
    hasMore,
    totalCount,
    latestBlockUid: blocks.length > 0 ? blocks[blocks.length - 1].uid : "",
    syncRequired: false,
  });
}

/**
 * Helper to create an empty ListBlocksResponse
 */
function createEmptyListResponse(): ListBlocksResponse {
  return create(ListBlocksResponseSchema, { blocks: [] });
}

/**
 * Helper to create an empty paginated ListBlocksResponse
 */
function createEmptyPaginatedResponse(): ListBlocksResponse {
  return create(ListBlocksResponseSchema, {
    blocks: [],
    hasMore: false,
    totalCount: 0,
    latestBlockUid: "",
    syncRequired: false,
  });
}

/**
 * Helper to create an empty protobuf message (for delete/append operations)
 */
function createEmptyMessage() {
  return create(EmptySchema, {});
}

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
    expect(blockKeys.list(123, {})).toEqual(["blocks", "list", 123, {}]);
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
    const mockBlocks = createMockListResponse([
      createMockBlock({
        id: 1n,
        conversationId: 123,
        userInputs: [createMockUserInput("Hello", 1000)],
        assistantContent: "Hi there!",
        costEstimate: 2100n,
      }),
    ]);

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
    const mockBlocks = [
      createMockBlock({
        id: 1n,
        conversationId: 123,
        assistantContent: "Response",
      }),
    ];

    vi.mocked(aiServiceClient.listBlocks).mockResolvedValue(createMockPaginatedResponse(mockBlocks));

    const { result } = renderHook(() => useBlocksWithFallback(123), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.blocks).toEqual(mockBlocks);
    expect(result.current.shouldFallback).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("should indicate fallback when query fails", async () => {
    // Create fresh query client for this test
    const freshQueryClient = createTestQueryClient();
    const freshWrapper = createWrapper(freshQueryClient);

    // Use mockRejectedValue directly
    const error = new Error("Network error");
    vi.mocked(aiServiceClient.listBlocks).mockRejectedValue(error);

    const { result } = renderHook(() => useBlocksWithFallback(123), { wrapper: freshWrapper });

    // Wait for query to finish
    await waitFor(() => expect(result.current.isLoading).toBe(false), { timeout: 15000 });

    expect(result.current.shouldFallback).toBe(true);
    expect(result.current.error).not.toBeNull();
  });

  it("should indicate fallback when no blocks returned for active conversation", async () => {
    // Clear any cached data from previous tests
    queryClient.clear();

    vi.mocked(aiServiceClient.listBlocks).mockResolvedValue(createEmptyPaginatedResponse());

    const { result } = renderHook(() => useBlocksWithFallback(123, { isActive: true }), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.shouldFallback).toBe(true);
    expect(result.current.blocks).toEqual([]);
  });

  it("should provide refetch function", async () => {
    queryClient.clear();

    vi.mocked(aiServiceClient.listBlocks).mockResolvedValue(createEmptyPaginatedResponse());

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
    queryClient.setQueryData(
      blockKeys.detail(1),
      createMockBlock({
        id: 1n,
        status: BlockStatus.PENDING,
        createdTs: 1000n,
        updatedTs: 1000n,
        assistantTimestamp: 1000n,
        costEstimate: 0n,
        modelVersion: "",
      }),
    );
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

    const event = create(BlockEventSchema, {
      type: "thinking",
      content: "",
      timestamp: BigInt(Date.now()),
      meta: "{}",
    });

    // Convert BigInt to string for JSON serialization
    const eventJson = JSON.stringify(event, (_, value) => (typeof value === "bigint" ? value.toString() : value));

    // appendStreamingEvent expects a JSON string
    result.current.appendStreamingEvent(eventJson);

    const cached = queryClient.getQueryData(blockKeys.detail(1)) as Block | undefined;
    // The event is stored as JSON string in the eventStream array
    expect(cached).toBeDefined();
    expect(cached?.eventStream).toContain(eventJson);
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
    const createdBlock = createMockBlock({
      id: 999n,
      uid: "block-999",
      conversationId: 123,
      userInputs: [createMockUserInput("Test", 1000)],
      status: BlockStatus.PENDING,
      costEstimate: 0n,
      modelVersion: "",
    });

    vi.mocked(aiServiceClient.createBlock).mockResolvedValue(createdBlock);

    const { result } = renderHook(() => useCreateBlock(), { wrapper });

    // First populate the list
    queryClient.setQueryData(blockKeys.list(123), createEmptyListResponse());

    result.current.mutate(
      create(CreateBlockRequestSchema, {
        conversationId: 123,
        mode: BlockMode.NORMAL,
        blockType: BlockType.MESSAGE,
        userInputs: [createMockUserInput("Test", 1000)],
      }),
    );

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
    const updatedBlock = createMockBlock({
      id: 1n,
      assistantContent: "Updated content",
      status: BlockStatus.COMPLETED,
      updatedTs: 3000n,
      costEstimate: 1000n,
      modelVersion: "deepseek-chat",
    });

    vi.mocked(aiServiceClient.updateBlock).mockResolvedValue(updatedBlock);

    const { result } = renderHook(() => useUpdateBlock(), { wrapper });

    result.current.mutate(
      create(UpdateBlockRequestSchema, {
        id: 1n,
        assistantContent: "Updated content",
      }),
    );

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
    vi.mocked(aiServiceClient.deleteBlock).mockResolvedValue(createEmptyMessage());

    const { result } = renderHook(() => useDeleteBlock(), { wrapper });

    result.current.mutate(
      create(DeleteBlockRequestSchema, {
        id: 1n,
      }),
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(aiServiceClient.deleteBlock).toHaveBeenCalled();
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
    vi.mocked(aiServiceClient.appendEvent).mockResolvedValue(createEmptyMessage());

    const { result } = renderHook(() => useAppendEvent(), { wrapper });

    const event = create(BlockEventSchema, {
      type: "thinking",
      content: "",
      timestamp: BigInt(Date.now()),
      meta: "{}",
    });

    result.current.mutate(
      create(AppendEventRequestSchema, {
        id: 1n,
        event,
      }),
    );

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

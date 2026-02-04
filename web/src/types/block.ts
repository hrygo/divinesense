/**
 * Unified Block Model - Frontend Type Definitions
 *
 * This file exports Block-related types from the generated proto definitions
 * and provides additional convenience types for the frontend.
 *
 * Phase 3: Frontend type definitions for Unified Block Model
 * @see docs/specs/unified-block-model.md
 */

// Re-export proto types for convenience
export type {
  Block,
  BlockType,
  BlockMode,
  BlockStatus,
  UserInput,
  BlockEvent,
  ListBlocksRequest,
  ListBlocksResponse,
  GetBlockRequest,
  CreateBlockRequest,
  UpdateBlockRequest,
  DeleteBlockRequest,
  AppendUserInputRequest,
  AppendEventRequest,
} from "./proto/api/v1/ai_service_pb";

// Re-export SessionStats since it's used by Block
export type { SessionStats } from "./proto/api/v1/ai_service_pb";

/**
 * Block type constants (for type guards and comparisons)
 */
export const BLOCK_TYPE = {
  UNSPECIFIED: "BLOCK_TYPE_UNSPECIFIED",
  MESSAGE: "BLOCK_TYPE_MESSAGE",
  CONTEXT_SEPARATOR: "BLOCK_TYPE_CONTEXT_SEPARATOR",
} as const;

/**
 * Block mode constants (for type guards and comparisons)
 */
export const BLOCK_MODE = {
  UNSPECIFIED: "BLOCK_MODE_UNSPECIFIED",
  NORMAL: "BLOCK_MODE_NORMAL",
  GEEK: "BLOCK_MODE_GEEK",
  EVOLUTION: "BLOCK_MODE_EVOLUTION",
} as const;

/**
 * Block status constants (for type guards and comparisons)
 */
export const BLOCK_STATUS = {
  UNSPECIFIED: "BLOCK_STATUS_UNSPECIFIED",
  PENDING: "BLOCK_STATUS_PENDING",
  STREAMING: "BLOCK_STATUS_STREAMING",
  COMPLETED: "BLOCK_STATUS_COMPLETED",
  ERROR: "BLOCK_STATUS_ERROR",
} as const;

/**
 * Event type constants (for type guards and comparisons)
 */
export const EVENT_TYPE = {
  THINKING: "thinking",
  TOOL_USE: "tool_use",
  TOOL_RESULT: "tool_result",
  ANSWER: "answer",
  ERROR: "error",
} as const;

/**
 * Type guard for checking if a status is terminal (completed or error)
 */
export function isTerminalStatus(status: string): boolean {
  return status === BLOCK_STATUS.COMPLETED || status === BLOCK_STATUS.ERROR;
}

/**
 * Type guard for checking if a status is active (pending or streaming)
 */
export function isActiveStatus(status: string): boolean {
  return status === BLOCK_STATUS.PENDING || status === BLOCK_STATUS.STREAMING;
}

/**
 * Get display name for block type
 */
export function getBlockTypeName(type: string): string {
  switch (type) {
    case BLOCK_TYPE.MESSAGE:
      return "message";
    case BLOCK_TYPE.CONTEXT_SEPARATOR:
      return "context_separator";
    default:
      return "unspecified";
  }
}

/**
 * Get display name for block mode
 */
export function getBlockModeName(mode: string): string {
  switch (mode) {
    case BLOCK_MODE.NORMAL:
      return "normal";
    case BLOCK_MODE.GEEK:
      return "geek";
    case BLOCK_MODE.EVOLUTION:
      return "evolution";
    default:
      return "unspecified";
  }
}

/**
 * Get display name for block status
 */
export function getBlockStatusName(status: string): string {
  switch (status) {
    case BLOCK_STATUS.PENDING:
      return "pending";
    case BLOCK_STATUS.STREAMING:
      return "streaming";
    case BLOCK_STATUS.COMPLETED:
      return "completed";
    case BLOCK_STATUS.ERROR:
      return "error";
    default:
      return "unspecified";
  }
}

/**
 * Frontend-specific Block type with additional computed properties
 */
export interface BlockWithMetadata {
  // Original block data
  block: import("./proto/api/v1/ai_service_pb").Block;
  // Computed properties
  isActive: boolean;
  isTerminal: boolean;
  modeName: string;
  statusName: string;
  eventCount: number;
  userInputsCount: number;
}

/**
 * Create a BlockWithMetadata from a Block
 */
export function createBlockWithMetadata(
  block: import("./proto/api/v1/ai_service_pb").Block
): BlockWithMetadata {
  const status = block.status;
  return {
    block,
    isActive: isActiveStatus(status),
    isTerminal: isTerminalStatus(status),
    modeName: getBlockModeName(block.mode),
    statusName: getBlockStatusName(status),
    eventCount: block.eventStream?.length ?? 0,
    userInputsCount: block.userInputs?.length ?? 0,
  };
}

/**
 * Block list filter options (for UI filtering)
 */
export interface BlockListFilters {
  status?: string;
  mode?: string;
  ccSessionId?: string;
}

/**
 * Create a ListBlocksRequest from filters
 */
export function createListBlocksRequest(
  conversationId: number,
  filters?: BlockListFilters
): import("./proto/api/v1/ai_service_pb").ListBlocksRequest {
  return {
    conversationId,
    status: filters?.status ?? ("" as any),
    mode: filters?.mode ?? ("" as any),
    ccSessionId: filters?.ccSessionId ?? "",
  };
}

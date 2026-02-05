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
// Re-export SessionStats since it's used by Block
export type {
  AppendEventRequest,
  AppendUserInputRequest,
  Block,
  BlockEvent,
  BlockMode,
  BlockStatus,
  BlockType,
  CreateBlockRequest,
  DeleteBlockRequest,
  GetBlockRequest,
  ListBlocksRequest,
  ListBlocksResponse,
  SessionStats,
  UpdateBlockRequest,
  UserInput,
} from "./proto/api/v1/ai_service_pb";

// Import enum types for type guards
import { BlockMode as BlockModeEnum, BlockStatus as BlockStatusEnum, BlockType as BlockTypeEnum } from "./proto/api/v1/ai_service_pb";

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
export function isTerminalStatus(status: BlockStatusEnum | string): boolean {
  const statusStr = typeof status === "string" ? status : String(status);
  return statusStr === String(BLOCK_STATUS.COMPLETED) || statusStr === String(BLOCK_STATUS.ERROR);
}

/**
 * Type guard for checking if a status is active (pending or streaming)
 * BlockStatus enum values: UNSPECIFIED=0, PENDING=1, STREAMING=2, COMPLETED=3, ERROR=4
 */
export function isActiveStatus(status: BlockStatusEnum | string | number): boolean {
  const statusNum = typeof status === "number" ? status : parseInt(String(status), 10) || 0;
  // PENDING=1 or STREAMING=2 are active states
  return statusNum === 1 || statusNum === 2;
}

/**
 * Get display name for block type
 */
export function getBlockTypeName(type: BlockTypeEnum | string): string {
  const typeStr = typeof type === "string" ? type : String(type);
  switch (typeStr) {
    case String(BLOCK_TYPE.MESSAGE):
      return "message";
    case String(BLOCK_TYPE.CONTEXT_SEPARATOR):
      return "context_separator";
    default:
      return "unspecified";
  }
}

/**
 * Get display name for block mode
 * BlockMode is numeric: 0=UNSPECIFIED, 1=NORMAL, 2=GEEK, 3=EVOLUTION
 */
export function getBlockModeName(mode: BlockModeEnum | string): string {
  const modeNum = typeof mode === "number" ? mode : parseInt(String(mode), 10) || 0;
  switch (modeNum) {
    case 1: // BlockMode.NORMAL
      return "normal";
    case 2: // BlockMode.GEEK
      return "geek";
    case 3: // BlockMode.EVOLUTION
      return "evolution";
    case 0: // BlockMode.UNSPECIFIED
    default:
      return "unspecified";
  }
}

/**
 * Get display name for block status
 */
export function getBlockStatusName(status: BlockStatusEnum | string): string {
  const statusStr = typeof status === "string" ? status : String(status);
  switch (statusStr) {
    case String(BLOCK_STATUS.PENDING):
      return "pending";
    case String(BLOCK_STATUS.STREAMING):
      return "streaming";
    case String(BLOCK_STATUS.COMPLETED):
      return "completed";
    case String(BLOCK_STATUS.ERROR):
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
export function createBlockWithMetadata(block: import("./proto/api/v1/ai_service_pb").Block): BlockWithMetadata {
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
 * Import ParrotAgentType for mode conversion
 */
import { ParrotAgentType } from "./parrot";

/**
 * Convert BlockMode to ParrotAgentType
 * Maps the unified Block model modes to Parrot agent types
 *
 * Note: BlockMode and ParrotAgentType are independent concepts.
 * - BlockMode: User-selected mode in UI (NORMAL/GEKE/EVOLUTION)
 * - ParrotAgentType: Actual parrot agent handling the request
 *
 * @param mode - The BlockMode enum value
 * @returns The corresponding ParrotAgentType (for display purposes only)
 */
export function blockModeToParrotAgentType(mode: BlockModeEnum | string): ParrotAgentType {
  // BlockMode is a numeric enum from proto:
  // UNSPECIFIED = 0, NORMAL = 1, GEEK = 2, EVOLUTION = 3
  const modeNum = typeof mode === "number" ? mode : parseInt(String(mode), 10) || 0;
  switch (modeNum) {
    case 2: // BlockMode.GEEK → GEEK parrot
      return ParrotAgentType.GEEK;
    case 3: // BlockMode.EVOLUTION → EVOLUTION parrot
      return ParrotAgentType.EVOLUTION;
    case 1: // BlockMode.NORMAL → AUTO (routed by backend)
    case 0: // BlockMode.UNSPECIFIED
    default:
      return ParrotAgentType.AUTO;
  }
}

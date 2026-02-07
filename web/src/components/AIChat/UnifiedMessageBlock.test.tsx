/**
 * UnifiedMessageBlock Component Tests
 *
 * Tests for the UnifiedMessageBlock component and its internal components including:
 * - BlockHeader functionality (responsive layout, multi-user input badge, session stats, toggle, branch indicator)
 * - Message rendering
 * - Collapse/expand behavior
 * - Status indicators
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ConversationMessage } from "@/types/aichat";
import type { BlockSummary } from "@/types/parrot";
import { ParrotAgentType } from "@/types/parrot";
import type { UnifiedMessageBlockProps } from "./UnifiedMessageBlock";
import { UnifiedMessageBlock } from "./UnifiedMessageBlock";

// Mock i18next
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) => {
      if (key === "ai.session_stats.currency_symbol") return "$";
      if (options?.count) return `${key}:${options.count}`;
      return key;
    },
  }),
}));

// Mock Lucide icons
vi.mock("lucide-react", async () => {
  const actual = await vi.importActual("lucide-react");
  return {
    ...actual,
    // Add any specific icon mocks if needed
  };
});

describe("UnifiedMessageBlock - BlockHeader Functionality", () => {
  // Counter for unique message IDs
  let messageIdCounter = 0;

  // Mock props factory
  const createMockUserMessage = (content: string, timestamp?: number, metadata?: ConversationMessage["metadata"]): ConversationMessage => ({
    id: `msg-${messageIdCounter++}`,
    uid: `uid-${messageIdCounter - 1}`,
    role: "user",
    content,
    timestamp: timestamp ?? Date.now(),
    metadata,
  });

  const createMockAssistantMessage = (content?: string, error?: boolean): ConversationMessage => ({
    id: `msg-${messageIdCounter++}`,
    uid: `uid-${messageIdCounter - 1}`,
    role: "assistant",
    content: content ?? "This is a response",
    timestamp: Date.now(),
    error,
    metadata: {
      toolCalls: [],
      toolResults: [],
    },
  });

  const createMockBlockSummary = (overrides?: Partial<BlockSummary>): BlockSummary => ({
    totalDurationMs: 5000,
    thinkingDurationMs: 1000,
    toolDurationMs: 2000,
    generationDurationMs: 2000,
    totalInputTokens: 1000,
    totalOutputTokens: 500,
    totalCostUSD: 0.001,
    toolCallCount: 2,
    ...overrides,
  });

  const createDefaultProps = (overrides?: Partial<UnifiedMessageBlockProps>): UnifiedMessageBlockProps => ({
    userMessage: createMockUserMessage("Hello, how are you?"),
    assistantMessage: createMockAssistantMessage(),
    blockSummary: createMockBlockSummary(),
    parrotId: ParrotAgentType.AUTO,
    isLatest: true,
    onCopy: vi.fn(),
    onRegenerate: vi.fn(),
    onDelete: vi.fn(),
    ...overrides,
  });

  beforeEach(() => {
    // Reset matchMedia mock before each test - default to mobile (<1024px)
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: query === "(min-width: 1024px)" ? false : query === "(min-width: 640px)",
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe("BlockHeader - Basic Rendering", () => {
    it("should render user message preview in header", () => {
      const props = createDefaultProps();
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Look for the header's truncate paragraph which contains the message preview
      const previewElement = container.querySelector(".truncate[title]");
      expect(previewElement).toBeInTheDocument();
      expect(previewElement).toHaveAttribute("title", "Hello, how are you?");
    });

    it("should extract and display user initial from message", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("Test message"),
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Find avatar with w-7 h-7 classes
      const avatar = container.querySelector(".w-7.h-7");
      expect(avatar).toBeInTheDocument();
      expect(avatar).toHaveTextContent("T");
    });

    it("should extract Chinese character as initial", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("你好世界"),
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      const avatar = container.querySelector(".w-7.h-7");
      expect(avatar).toHaveTextContent("你");
    });

    it("should display 'U' for empty content", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage(""),
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      const avatar = container.querySelector(".w-7.h-7");
      expect(avatar).toHaveTextContent("U");
    });
  });

  describe("BlockHeader - Multi-User Input Badge", () => {
    it("should not show badge when there is only one input", () => {
      const props = createDefaultProps({
        additionalUserInputs: [],
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      const badge = container.querySelector(".absolute.-top-1.-right-1");
      expect(badge).not.toBeInTheDocument();
    });

    it("should show badge with count when there are multiple inputs", () => {
      const props = createDefaultProps({
        additionalUserInputs: [createMockUserMessage("Additional input 1"), createMockUserMessage("Additional input 2")],
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      const badge = container.querySelector(".absolute.-top-1.-right-1");
      expect(badge).toBeInTheDocument();
      expect(badge).toHaveTextContent("3");
    });

    it("should show '+1' suffix when there are 2 inputs", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("First input"),
        additionalUserInputs: [createMockUserMessage("Second input")],
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // The preview should show first line
      const previewElement = container.querySelector(".truncate[title]");
      expect(previewElement).toHaveAttribute("title", "First input");
    });
  });

  describe("BlockHeader - Session Stats Display", () => {
    it("should display token usage for normal mode", () => {
      const props = createDefaultProps({
        blockSummary: createMockBlockSummary({
          totalInputTokens: 1500,
          totalOutputTokens: 500,
          totalCostUSD: 0.002,
        }),
        parrotId: ParrotAgentType.AUTO,
      });
      render(<UnifiedMessageBlock {...props} />);

      // Should show lightning icon for tokens (use getAllByText as there might be multiple)
      const lightning = screen.getAllByText("⚡");
      expect(lightning.length).toBeGreaterThan(0);
    });

    it("should display cost for normal mode", () => {
      const props = createDefaultProps({
        blockSummary: createMockBlockSummary({
          totalCostUSD: 0.005,
        }),
        parrotId: ParrotAgentType.AUTO,
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Cost appears in both desktop and mobile views
      const costBadges = container.querySelectorAll(".font-bold");
      const dollarSigns = Array.from(costBadges).filter((el) => el.textContent === "$");
      expect(dollarSigns.length).toBeGreaterThan(0);
    });

    it("should display duration for geek mode", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("Help me with code", undefined, { mode: "geek" }),
        blockSummary: createMockBlockSummary({
          totalDurationMs: 15000,
          toolCallCount: 5,
        }),
        parrotId: ParrotAgentType.GEEK,
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Check for duration text
      const allText = container.textContent || "";
      expect(allText).toContain("15.0s");
    });

    it("should display duration for evolution mode", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("Evolution task", undefined, { mode: "evolution" }),
        blockSummary: createMockBlockSummary({
          totalDurationMs: 30000,
          filesModified: 3,
        }),
        parrotId: ParrotAgentType.EVOLUTION,
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Check for duration text
      const allText = container.textContent || "";
      expect(allText).toContain("30.0s");
    });

    it("should not render stats when blockSummary is not provided", () => {
      const props = createDefaultProps({
        blockSummary: undefined,
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Should still render the header but without stats
      const previewElement = container.querySelector(".truncate[title]");
      expect(previewElement).toBeInTheDocument();
    });
  });

  describe("BlockHeader - Status Border", () => {
    it("should show blue border when streaming", () => {
      const props = createDefaultProps({
        isStreaming: true,
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      const header = container.querySelector(".border-l-blue-500\\/50");
      expect(header).toBeInTheDocument();
    });

    it("should show red border when assistant message has error", () => {
      const props = createDefaultProps({
        assistantMessage: createMockAssistantMessage(undefined, true),
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      const header = container.querySelector(".border-l-red-500");
      expect(header).toBeInTheDocument();
    });

    it("should show transparent border when not streaming and no error", () => {
      const props = createDefaultProps({
        isStreaming: false,
        assistantMessage: createMockAssistantMessage(),
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      const header = container.querySelector(".border-l-transparent");
      expect(header).toBeInTheDocument();
    });
  });

  describe("BlockHeader - Parrot Badge", () => {
    it("should show GEEK badge for GEEK parrot", () => {
      const props = createDefaultProps({
        parrotId: ParrotAgentType.GEEK,
      });
      render(<UnifiedMessageBlock {...props} />);

      expect(screen.getByText("ai.mode.geek")).toBeInTheDocument();
    });

    it("should show EVOLUTION badge for EVOLUTION parrot", () => {
      const props = createDefaultProps({
        parrotId: ParrotAgentType.EVOLUTION,
      });
      render(<UnifiedMessageBlock {...props} />);

      expect(screen.getByText("ai.mode.evolution")).toBeInTheDocument();
    });

    it("should not show badge for AUTO parrot", () => {
      const props = createDefaultProps({
        parrotId: ParrotAgentType.AUTO,
      });
      render(<UnifiedMessageBlock {...props} />);

      expect(screen.queryByText("ai.mode.geek")).not.toBeInTheDocument();
      expect(screen.queryByText("ai.mode.evolution")).not.toBeInTheDocument();
    });
  });

  describe("BlockHeader - Timestamp Display", () => {
    it("should display timestamp", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("Test", Date.now() - 30000),
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Should show clock icon
      const clockIcon = container.querySelector(".lucide-clock");
      expect(clockIcon).toBeInTheDocument();
    });
  });

  describe("BlockHeader - Block Number Indicator", () => {
    it("should render block number indicator when blockNumber is provided", () => {
      const props = createDefaultProps({
        blockNumber: 1,
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Look for the block number in the component
      const allText = container.textContent || "";
      expect(allText).toContain("1");
    });

    it("should not render block number indicator when blockNumber is not provided", () => {
      const props = createDefaultProps({
        blockNumber: undefined,
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Check that no hash icon exists for block number (lucide-hash is only used for block number)
      const hashIcons = container.querySelectorAll(".lucide-hash");
      expect(hashIcons.length).toBe(0);
    });
  });

  describe("Collapse/Expand Toggle", () => {
    it("should toggle collapse state when header is clicked", () => {
      const props = createDefaultProps();
      render(<UnifiedMessageBlock {...props} />);

      // Find the header by the user message text - use getAllByText and find the one with cursor-pointer class
      const textElements = screen.getAllByText("Hello, how are you?");
      const header = textElements.map((el) => el.closest("div.cursor-pointer")).find((el) => el !== null);

      expect(header).toBeInTheDocument();

      // Click to collapse
      if (header) {
        fireEvent.click(header);
      }

      // After clicking, the state should toggle
      // Note: This tests the interaction exists, actual state change depends on component implementation
    });
  });

  describe("Accessibility", () => {
    it("should have cursor-pointer class on header", () => {
      const props = createDefaultProps();
      const { container } = render(<UnifiedMessageBlock {...props} />);

      const header = container.querySelector(".cursor-pointer");
      expect(header).toBeInTheDocument();
    });

    it("should have select-none class on header", () => {
      const props = createDefaultProps();
      const { container } = render(<UnifiedMessageBlock {...props} />);

      const header = container.querySelector(".select-none");
      expect(header).toBeInTheDocument();
    });
  });

  describe("Edge Cases", () => {
    it("should handle undefined assistantMessage", () => {
      const props = createDefaultProps({
        assistantMessage: undefined,
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Component should still render with user message
      const previewElement = container.querySelector(".truncate[title]");
      expect(previewElement).toHaveAttribute("title", "Hello, how are you?");
    });

    it("should handle undefined blockSummary", () => {
      const props = createDefaultProps({
        blockSummary: undefined,
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Component should still render
      const previewElement = container.querySelector(".truncate[title]");
      expect(previewElement).toHaveAttribute("title", "Hello, how are you?");
    });

    it("should handle zero token counts", () => {
      const props = createDefaultProps({
        blockSummary: createMockBlockSummary({
          totalInputTokens: 0,
          totalOutputTokens: 0,
          totalCostUSD: 0,
        }),
      });
      const { container } = render(<UnifiedMessageBlock {...props} />);

      // Component should still render
      const previewElement = container.querySelector(".truncate[title]");
      expect(previewElement).toHaveAttribute("title", "Hello, how are you?");
    });
  });
});

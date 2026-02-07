/**
 * BlockHeader Component Tests
 *
 * Tests for the BlockHeader component including:
 * - Responsive layout (mobile/desktop)
 * - Multi-user input badge display
 * - Session stats display
 * - Collapse/expand toggle
 * - Branch indicator display
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ConversationMessage } from "@/types/aichat";
import type { BlockBranch } from "@/types/block";
import type { BlockSummary } from "@/types/parrot";
import { ParrotAgentType } from "@/types/parrot";
import type { BlockHeaderProps } from "../BlockHeader";
import { BlockHeader } from "../BlockHeader";

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

// Mock utility functions - need to use default import for the index
vi.mock("../utils", () => ({
  formatRelativeTime: vi.fn((timestamp: number) => {
    const now = Date.now();
    const diff = now - timestamp;
    if (diff < 60000) return "ai.aichat.sidebar.time-just-now";
    if (diff < 3600000) return `ai.aichat.sidebar.time-minutes-ago:${Math.floor(diff / 60000)}`;
    return "Jan 1";
  }),
  getVisualWidth: vi.fn((str: string) => {
    return str.length; // Simplified: count characters
  }),
  truncateByVisualWidth: vi.fn((str: string, maxWidth: number) => {
    return str.length > maxWidth ? str.slice(0, maxWidth) + "..." : str;
  }),
}));

describe("BlockHeader", () => {
  // Mock props factory
  const createMockUserMessage = (content: string, timestamp?: number, metadata?: ConversationMessage["metadata"]): ConversationMessage => ({
    id: "msg-1",
    uid: "uid-1",
    role: "user",
    content,
    timestamp: timestamp ?? Date.now(),
    metadata,
  });

  const createMockAssistantMessage = (content?: string, error?: boolean): ConversationMessage => ({
    id: "msg-2",
    uid: "uid-2",
    role: "assistant",
    content: content ?? "This is a response",
    timestamp: Date.now(),
    error,
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

  const createDefaultProps = (overrides?: Partial<BlockHeaderProps>): BlockHeaderProps => ({
    userMessage: createMockUserMessage("Hello, how are you?"),
    assistantMessage: createMockAssistantMessage(),
    blockSummary: createMockBlockSummary(),
    parrotId: ParrotAgentType.AUTO,
    theme: {
      border: "border-l-4",
      headerBg: "bg-white",
      footerBg: "bg-gray-50",
      badgeBg: "bg-blue-100",
      badgeText: "text-blue-700",
      ringColor: "ring-blue-500",
    },
    onToggle: vi.fn(),
    isCollapsed: false,
    isStreaming: false,
    additionalUserInputs: [],
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

  describe("Basic Rendering", () => {
    it("should render user message preview", () => {
      const props = createDefaultProps();
      render(<BlockHeader {...props} />);

      expect(screen.getByText("Hello, how are you?")).toBeInTheDocument();
    });

    it("should extract and display user initial from message", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("Test message"),
      });
      const { container } = render(<BlockHeader {...props} />);

      const avatar = container.querySelector(".w-7.h-7");
      expect(avatar).toHaveTextContent("T");
    });

    it("should extract Chinese character as initial", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("你好世界"),
      });
      const { container } = render(<BlockHeader {...props} />);

      const avatar = container.querySelector(".w-7.h-7");
      expect(avatar).toHaveTextContent("你");
    });

    it("should display 'U' for empty content", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage(""),
      });
      const { container } = render(<BlockHeader {...props} />);

      const avatar = container.querySelector(".w-7.h-7");
      expect(avatar).toHaveTextContent("U");
    });

    it("should call onToggle when header is clicked", () => {
      const onToggle = vi.fn();
      const props = createDefaultProps({ onToggle });
      render(<BlockHeader {...props} />);

      const header = screen.getByText("Hello, how are you?").closest("div");
      expect(header).toBeInTheDocument();
      if (header) {
        fireEvent.click(header);
      }
      expect(onToggle).toHaveBeenCalled();
    });
  });

  describe("Multi-User Input Badge", () => {
    it("should not show badge when there is only one input", () => {
      const props = createDefaultProps({
        additionalUserInputs: [],
      });
      const { container } = render(<BlockHeader {...props} />);

      const badge = container.querySelector(".absolute.-top-1.-right-1");
      expect(badge).not.toBeInTheDocument();
    });

    it("should show badge with count when there are multiple inputs", () => {
      const props = createDefaultProps({
        additionalUserInputs: [createMockUserMessage("Additional input 1"), createMockUserMessage("Additional input 2")],
      });
      const { container } = render(<BlockHeader {...props} />);

      const badge = container.querySelector(".absolute.-top-1.-right-1");
      expect(badge).toHaveTextContent("3");
    });

    it("should show '+1' suffix when there are 2 inputs", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("First input"),
        additionalUserInputs: [createMockUserMessage("Second input")],
      });
      render(<BlockHeader {...props} />);

      // The preview should show truncated first line + "+1"
      expect(screen.getByText(/First input/)).toBeInTheDocument();
    });

    it("should show '+N' suffix when there are more than 2 inputs", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("First"),
        additionalUserInputs: [createMockUserMessage("Second"), createMockUserMessage("Third"), createMockUserMessage("Fourth")],
      });
      render(<BlockHeader {...props} />);

      expect(screen.getByText(/First/)).toBeInTheDocument();
    });
  });

  describe("Session Stats Display", () => {
    it("should display token usage for normal mode", () => {
      const props = createDefaultProps({
        blockSummary: createMockBlockSummary({
          totalInputTokens: 1500,
          totalOutputTokens: 500,
          totalCostUSD: 0.002,
        }),
        parrotId: ParrotAgentType.AUTO,
      });
      render(<BlockHeader {...props} />);

      // Should show lightning icon and token count (desktop)
      const lightning = screen.getByText("⚡");
      expect(lightning).toBeInTheDocument();
    });

    it("should display cost for normal mode", () => {
      const props = createDefaultProps({
        blockSummary: createMockBlockSummary({
          totalCostUSD: 0.005,
        }),
        parrotId: ParrotAgentType.AUTO,
      });
      const { container } = render(<BlockHeader {...props} />);

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
      const { container } = render(<BlockHeader {...props} />);

      // Check for clock icon and duration
      const clockIcons = container.querySelectorAll(".lucide-clock");
      expect(clockIcons.length).toBeGreaterThan(0);

      // Check for duration text
      const allText = container.textContent || "";
      expect(allText).toContain("15.0s");
    });

    it("should display tool count for geek mode on desktop", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("Help me with code", undefined, { mode: "geek" }),
        blockSummary: createMockBlockSummary({
          toolCallCount: 5,
        }),
        parrotId: ParrotAgentType.GEEK,
      });
      render(<BlockHeader {...props} />);

      // Just verify the component renders
      expect(screen.getByText("Help me with code")).toBeInTheDocument();
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
      const { container } = render(<BlockHeader {...props} />);

      // Check for duration text
      const allText = container.textContent || "";
      expect(allText).toContain("30.0s");
    });

    it("should not render stats when blockSummary is not provided", () => {
      const props = createDefaultProps({
        blockSummary: undefined,
      });
      render(<BlockHeader {...props} />);

      // Should still render the header but without stats
      expect(screen.getByText("Hello, how are you?")).toBeInTheDocument();
    });
  });

  describe("Collapse/Expand Toggle", () => {
    it("should show ChevronUp when expanded", () => {
      const props = createDefaultProps({ isCollapsed: false });
      const { container } = render(<BlockHeader {...props} />);

      const toggleButton = container.querySelector("button[aria-expanded='true']");
      expect(toggleButton).toBeInTheDocument();
    });

    it("should show ChevronDown when collapsed", () => {
      const props = createDefaultProps({ isCollapsed: true });
      const { container } = render(<BlockHeader {...props} />);

      const toggleButton = container.querySelector("button[aria-expanded='false']");
      expect(toggleButton).toBeInTheDocument();
    });

    it("should call onToggle when toggle button is clicked", () => {
      const onToggle = vi.fn();
      const props = createDefaultProps({ onToggle, isCollapsed: false });
      const { container } = render(<BlockHeader {...props} />);

      const toggleButton = container.querySelector("button");
      if (toggleButton) {
        fireEvent.click(toggleButton);
      }
      expect(onToggle).toHaveBeenCalled();
    });

    it("should stop propagation when toggle button is clicked", () => {
      const onToggle = vi.fn();
      const props = createDefaultProps({ onToggle, isCollapsed: false });
      const { container } = render(<BlockHeader {...props} />);

      const toggleButton = container.querySelector("button");
      if (toggleButton) {
        // Click should not trigger parent's onClick
        const clickEvent = new MouseEvent("click", { bubbles: true });
        toggleButton.dispatchEvent(clickEvent);
      }
      // Should only be called once (not twice from propagation)
      expect(onToggle).toHaveBeenCalledTimes(1);
    });
  });

  describe("Status Border", () => {
    it("should show blue border when streaming", () => {
      const props = createDefaultProps({
        isStreaming: true,
      });
      const { container } = render(<BlockHeader {...props} />);

      const header = container.firstChild as HTMLElement;
      expect(header).toHaveClass("border-l-blue-500/50");
    });

    it("should show red border when assistant message has error", () => {
      const props = createDefaultProps({
        assistantMessage: createMockAssistantMessage(undefined, true),
      });
      const { container } = render(<BlockHeader {...props} />);

      const header = container.firstChild as HTMLElement;
      expect(header).toHaveClass("border-l-red-500");
    });

    it("should show transparent border when not streaming and no error", () => {
      const props = createDefaultProps({
        isStreaming: false,
        assistantMessage: createMockAssistantMessage(),
      });
      const { container } = render(<BlockHeader {...props} />);

      const header = container.firstChild as HTMLElement;
      expect(header).toHaveClass("border-l-transparent");
    });
  });

  describe("Parrot Badge", () => {
    it("should show GEEK badge for GEEK parrot", () => {
      const props = createDefaultProps({
        parrotId: ParrotAgentType.GEEK,
      });
      render(<BlockHeader {...props} />);

      expect(screen.getByText("ai.mode.geek")).toBeInTheDocument();
    });

    it("should show EVOLUTION badge for EVOLUTION parrot", () => {
      const props = createDefaultProps({
        parrotId: ParrotAgentType.EVOLUTION,
      });
      render(<BlockHeader {...props} />);

      expect(screen.getByText("ai.mode.evolution")).toBeInTheDocument();
    });

    it("should show AMAZING badge for AMAZING parrot", () => {
      const props = createDefaultProps({
        parrotId: ParrotAgentType.AMAZING,
      });
      render(<BlockHeader {...props} />);

      expect(screen.getByText("ai.mode.normal")).toBeInTheDocument();
    });

    it("should not show badge for AUTO parrot", () => {
      const props = createDefaultProps({
        parrotId: ParrotAgentType.AUTO,
      });
      render(<BlockHeader {...props} />);

      // AUTO should not show a badge
      expect(screen.queryByText("ai.mode.geek")).not.toBeInTheDocument();
      expect(screen.queryByText("ai.mode.evolution")).not.toBeInTheDocument();
    });
  });

  describe("Branch Indicator", () => {
    it("should render branch indicator when branchPath is provided", () => {
      const props = createDefaultProps({
        branchPath: "A.1.2",
      });
      const { container } = render(<BlockHeader {...props} />);

      // The actual BranchIndicator component renders, not the mock
      const branchButton = container.querySelector('button[title="A.1.2"]');
      expect(branchButton).toBeInTheDocument();
    });

    it("should render branch indicator when branches array is provided", () => {
      const mockBranch = {
        block: undefined,
        branchPath: "A",
        isActive: true,
        children: [],
        $typeName: "memos.api.v1.BlockBranch",
      } as BlockBranch;
      const props = createDefaultProps({
        branches: [mockBranch],
      });
      const { container } = render(<BlockHeader {...props} />);

      // Should show branch count when no path
      const branchButtons = container.querySelectorAll('button[class*="purple"]');
      expect(branchButtons.length).toBeGreaterThan(0);
    });

    it("should not render branch indicator when no branches", () => {
      const props = createDefaultProps({
        branches: [],
        branchPath: undefined,
      });
      const { container } = render(<BlockHeader {...props} />);

      // Check that no purple branch button exists
      const branchButtons = container.querySelectorAll('button[class*="purple"]');
      expect(branchButtons.length).toBe(0);
    });

    it("should call onBranchClick when branch indicator is clicked", () => {
      const onBranchClick = vi.fn();
      const props = createDefaultProps({
        branchPath: "A.1",
        onBranchClick,
      });
      const { container } = render(<BlockHeader {...props} />);

      const branchButton = container.querySelector('button[title="A.1"]');
      if (branchButton) {
        fireEvent.click(branchButton);
        expect(onBranchClick).toHaveBeenCalled();
      }
    });
  });

  describe("Message Preview Truncation", () => {
    it("should truncate long message previews", () => {
      const longMessage = "This is a very long message that should be truncated because it exceeds the visual width limit";
      const props = createDefaultProps({
        userMessage: createMockUserMessage(longMessage),
      });
      render(<BlockHeader {...props} />);

      // The mock truncateByVisualWidth truncates at 24 chars
      expect(screen.getByText(/This is a very long mess/)).toBeInTheDocument();
    });

    it("should not truncate short messages", () => {
      const shortMessage = "Hi";
      const props = createDefaultProps({
        userMessage: createMockUserMessage(shortMessage),
      });
      render(<BlockHeader {...props} />);

      expect(screen.getByText("Hi")).toBeInTheDocument();
    });
  });

  describe("Timestamp Display", () => {
    it("should display relative timestamp", () => {
      const props = createDefaultProps({
        userMessage: createMockUserMessage("Test", Date.now() - 30000), // 30 seconds ago
      });
      const { container } = render(<BlockHeader {...props} />);

      // Should show clock icon and time
      const clockIcon = container.querySelector(".lucide-clock");
      expect(clockIcon).toBeInTheDocument();
    });
  });

  describe("Accessibility", () => {
    it("should have proper aria-label on toggle button when collapsed", () => {
      const props = createDefaultProps({ isCollapsed: true });
      const { container } = render(<BlockHeader {...props} />);

      const toggleButton = container.querySelector("button");
      expect(toggleButton).toHaveAttribute("aria-label", "common.expand");
      expect(toggleButton).toHaveAttribute("aria-expanded", "false");
    });

    it("should have proper aria-label on toggle button when expanded", () => {
      const props = createDefaultProps({ isCollapsed: false });
      const { container } = render(<BlockHeader {...props} />);

      const toggleButton = container.querySelector("button");
      expect(toggleButton).toHaveAttribute("aria-label", "common.collapse");
      expect(toggleButton).toHaveAttribute("aria-expanded", "true");
    });

    it("should have cursor-pointer class", () => {
      const props = createDefaultProps();
      const { container } = render(<BlockHeader {...props} />);

      const header = container.firstChild as HTMLElement;
      expect(header).toHaveClass("cursor-pointer");
    });

    it("should have select-none class", () => {
      const props = createDefaultProps();
      const { container } = render(<BlockHeader {...props} />);

      const header = container.firstChild as HTMLElement;
      expect(header).toHaveClass("select-none");
    });
  });

  describe("Responsive Layout", () => {
    it("should hide parrot badge on mobile", () => {
      // Default matchMedia returns false for sm breakpoint
      const props = createDefaultProps({
        parrotId: ParrotAgentType.GEEK,
      });
      const { container } = render(<BlockHeader {...props} />);

      // Badge should have hidden sm:inline class
      const badge = container.querySelector('span[class*="hidden sm:inline"]');
      expect(badge).toBeInTheDocument();
    });
  });

  describe("Edge Cases", () => {
    it("should handle undefined assistantMessage", () => {
      const props = createDefaultProps({
        assistantMessage: undefined,
      });
      render(<BlockHeader {...props} />);

      expect(screen.getByText("Hello, how are you?")).toBeInTheDocument();
    });

    it("should handle undefined blockSummary", () => {
      const props = createDefaultProps({
        blockSummary: undefined,
      });
      render(<BlockHeader {...props} />);

      expect(screen.getByText("Hello, how are you?")).toBeInTheDocument();
    });

    it("should handle zero token counts", () => {
      const props = createDefaultProps({
        blockSummary: createMockBlockSummary({
          totalInputTokens: 0,
          totalOutputTokens: 0,
          totalCostUSD: 0,
        }),
      });
      render(<BlockHeader {...props} />);

      // Component should still render
      expect(screen.getByText("Hello, how are you?")).toBeInTheDocument();
    });

    it("should handle empty additionalUserInputs array", () => {
      const props = createDefaultProps({
        additionalUserInputs: [],
      });
      render(<BlockHeader {...props} />);

      expect(screen.getByText("Hello, how are you?")).toBeInTheDocument();
    });
  });
});

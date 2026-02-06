/**
 * TokenUsageBadge Component Tests
 *
 * Tests for the token usage display badge including:
 * - Collapsed/expanded states
 * - Token number formatting
 * - Cache token display
 * - i18n integration
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { TokenUsage } from "@/types/block";
import { TokenUsageBadge } from "./TokenUsageBadge";

// Mock i18next
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

describe("TokenUsageBadge", () => {
  const mockTokenUsage: TokenUsage = {
    $typeName: "memos.api.v1.TokenUsage" as const,
    promptTokens: 1000,
    completionTokens: 500,
    totalTokens: 1500,
    cacheReadTokens: 200,
    cacheWriteTokens: 50,
  };

  it("should render nothing when tokenUsage is not provided", () => {
    const { container } = render(<TokenUsageBadge />);
    expect(container.firstChild).toBeNull();
  });

  it("should render collapsed badge with total tokens", () => {
    render(<TokenUsageBadge tokenUsage={mockTokenUsage} />);

    expect(screen.getByText("1.5K")).toBeInTheDocument();
  });

  it("should expand on click", () => {
    render(<TokenUsageBadge tokenUsage={mockTokenUsage} />);

    const button = screen.getByRole("button");
    fireEvent.click(button);

    // Check expanded content
    expect(screen.getByText("chat.block-summary.total-tokens")).toBeInTheDocument();
    expect(screen.getByText("chat.block-summary.input-tokens")).toBeInTheDocument();
    expect(screen.getByText("chat.block-summary.output-tokens")).toBeInTheDocument();
  });

  it("should collapse on second click", () => {
    render(<TokenUsageBadge tokenUsage={mockTokenUsage} />);

    const button = screen.getByRole("button");

    // First click expands
    fireEvent.click(button);
    expect(screen.queryByText("chat.block-summary.total-tokens")).toBeInTheDocument();

    // Second click collapses
    fireEvent.click(button);
    expect(screen.queryByText("chat.block-summary.total-tokens")).not.toBeInTheDocument();
  });

  it("should format large numbers correctly", () => {
    const largeTokenUsage: TokenUsage = {
      $typeName: "memos.api.v1.TokenUsage" as const,
      promptTokens: 1000000,
      completionTokens: 2000000,
      totalTokens: 3000000,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
    };

    render(<TokenUsageBadge tokenUsage={largeTokenUsage} />);

    expect(screen.getByText("3.0M")).toBeInTheDocument();

    // Expand to see breakdown
    fireEvent.click(screen.getByRole("button"));
    expect(screen.getAllByText("1.0M").length).toBeGreaterThan(0);
    expect(screen.getByText("2.0M")).toBeInTheDocument();
  });

  it("should show cache tokens when present", () => {
    render(<TokenUsageBadge tokenUsage={mockTokenUsage} />);

    fireEvent.click(screen.getByRole("button"));

    expect(screen.getByText("chat.block-summary.cache-read")).toBeInTheDocument();
    expect(screen.getByText("chat.block-summary.cache-write")).toBeInTheDocument();
  });

  it("should not show cache section when no cache tokens", () => {
    const noCacheUsage: TokenUsage = {
      $typeName: "memos.api.v1.TokenUsage" as const,
      promptTokens: 1000,
      completionTokens: 500,
      totalTokens: 1500,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
    };

    render(<TokenUsageBadge tokenUsage={noCacheUsage} />);

    fireEvent.click(screen.getByRole("button"));

    expect(screen.queryByText("chat.block-summary.cache-read")).not.toBeInTheDocument();
    expect(screen.queryByText("chat.block-summary.cache-write")).not.toBeInTheDocument();
  });

  it("should calculate total from prompt+completion when total is 0", () => {
    const noTotalUsage: TokenUsage = {
      $typeName: "memos.api.v1.TokenUsage" as const,
      promptTokens: 1000,
      completionTokens: 500,
      totalTokens: 0,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
    };

    render(<TokenUsageBadge tokenUsage={noTotalUsage} />);

    expect(screen.getByText("1.5K")).toBeInTheDocument();
  });

  it("should apply custom className", () => {
    const { container } = render(<TokenUsageBadge tokenUsage={mockTokenUsage} className="custom-class" />);

    expect(container.firstChild).toHaveClass("custom-class");
  });
});

/**
 * BlockCostBadge Component Tests
 *
 * Tests for the cost estimate display badge including:
 * - Cost formatting (milli-cents, cents, dollars)
 * - Color coding based on cost tier
 * - Zero cost handling
 */

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BlockCostBadge } from "./BlockCostBadge";

// Mock i18next
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

describe("BlockCostBadge", () => {
  it("should render nothing when costEstimate is not provided", () => {
    const { container } = render(<BlockCostBadge />);
    expect(container.firstChild).toBeNull();
  });

  it("should render nothing when costEstimate is 0", () => {
    const { container } = render(<BlockCostBadge costEstimate={0n} />);
    expect(container.firstChild).toBeNull();
  });

  it("should display cost in milli-cents for very small amounts", () => {
    render(<BlockCostBadge costEstimate={500n} />); // 0.5 cents = 500 milli-cents

    expect(screen.getByText("500m¢")).toBeInTheDocument();
  });

  it("should display cost in cents for small amounts", () => {
    render(<BlockCostBadge costEstimate={10000n} />); // 10 cents = 10000 milli-cents

    expect(screen.getByText("10.00¢")).toBeInTheDocument();
  });

  it("should display cost in dollars for larger amounts", () => {
    render(<BlockCostBadge costEstimate={150000n} />); // $1.50 = 150000 milli-cents

    expect(screen.getByText("$1.5000")).toBeInTheDocument();
  });

  it("should use emerald color for very low cost (< $0.01)", () => {
    const { container } = render(<BlockCostBadge costEstimate={500n} />);

    expect(container.firstChild).toHaveClass("bg-emerald-100");
    expect(container.firstChild).toHaveClass("text-emerald-700");
  });

  it("should use green color for low cost (< $0.1)", () => {
    const { container } = render(<BlockCostBadge costEstimate={5000n} />); // $0.05

    expect(container.firstChild).toHaveClass("bg-green-100");
    expect(container.firstChild).toHaveClass("text-green-700");
  });

  it("should use amber color for medium cost (< $1)", () => {
    const { container } = render(<BlockCostBadge costEstimate={50000n} />); // $0.50

    expect(container.firstChild).toHaveClass("bg-amber-100");
    expect(container.firstChild).toHaveClass("text-amber-700");
  });

  it("should use orange color for high cost (>= $1)", () => {
    const { container } = render(<BlockCostBadge costEstimate={150000n} />); // $1.50

    expect(container.firstChild).toHaveClass("bg-orange-100");
    expect(container.firstChild).toHaveClass("text-orange-700");
  });

  it("should have title with full cost precision", () => {
    render(<BlockCostBadge costEstimate={12345n} />); // $0.12345

    const badge = screen.getByTitle("chat.block-summary.estimated-cost: $0.123450");
    expect(badge).toBeInTheDocument();
  });

  it("should apply custom className", () => {
    const { container } = render(<BlockCostBadge costEstimate={1000n} className="custom-class" />);

    expect(container.firstChild).toHaveClass("custom-class");
  });
});

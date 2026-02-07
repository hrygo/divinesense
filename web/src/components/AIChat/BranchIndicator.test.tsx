/**
 * BranchIndicator Component Tests
 *
 * Tests for the BlockNumberIndicator and related components.
 * Tests cover:
 * - BlockNumberIndicator rendering
 * - CompactBlockNumberIndicator rendering
 * - SimpleBlockNumberIndicator rendering
 * - Legacy BranchIndicator backward compatibility
 */

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
  BlockNumberIndicator,
  BranchIndicator,
  CompactBlockNumberIndicator,
  CompactBranchIndicator,
  SimpleBlockNumberIndicator,
  SimplePathIndicator,
} from "./BranchIndicator";

describe("BlockNumberIndicator", () => {
  it("should render for positive block numbers", () => {
    const { container } = render(<BlockNumberIndicator blockNumber={1} />);

    expect(container.textContent).toContain("1");
    expect(container.querySelector(".lucide-hash")).toBeInTheDocument();
  });

  it("should not render when blockNumber is 0", () => {
    const { container } = render(<BlockNumberIndicator blockNumber={0} />);

    expect(container.firstChild).toBe(null);
  });

  it("should not render when blockNumber is negative", () => {
    const { container } = render(<BlockNumberIndicator blockNumber={-1} />);

    expect(container.firstChild).toBe(null);
  });

  it("should show active dot when isActive is true", () => {
    const { container } = render(<BlockNumberIndicator blockNumber={5} isActive={true} />);

    // Should have the active dot
    const activeDot = container.querySelector(".rounded-full");
    expect(activeDot).toBeInTheDocument();
    expect(activeDot?.className).toContain("bg-slate-500");
  });

  it("should not show active dot when isActive is false", () => {
    const { container } = render(<BlockNumberIndicator blockNumber={5} isActive={false} />);

    const activeDot = container.querySelector(".rounded-full");
    expect(activeDot).not.toBeInTheDocument();
  });

  it("should display custom label when provided", () => {
    const { container } = render(<BlockNumberIndicator blockNumber={3} label="Custom Label" />);

    // The title is on the div, not the container
    const div = container.querySelector("div");
    expect(div?.getAttribute("title")).toBe("Custom Label");
  });

  it("should handle large block numbers", () => {
    const { container } = render(<BlockNumberIndicator blockNumber={999} />);

    expect(container.textContent).toContain("999");
  });
});

describe("CompactBlockNumberIndicator", () => {
  it("should render without Hash icon", () => {
    const { container } = render(<CompactBlockNumberIndicator blockNumber={2} />);

    expect(container.textContent).toContain("2");
    expect(container.querySelector(".lucide-hash")).not.toBeInTheDocument();
  });

  it("should not render when blockNumber is 0", () => {
    const { container } = render(<CompactBlockNumberIndicator blockNumber={0} />);

    expect(container.firstChild).toBe(null);
  });
});

describe("SimpleBlockNumberIndicator", () => {
  it("should render as circular badge", () => {
    const { container } = render(<SimpleBlockNumberIndicator blockNumber={1} />);

    expect(container.textContent).toContain("1");
    const firstChild = container.firstChild as HTMLElement | null;
    expect(firstChild?.className).toContain("rounded-full");
  });

  it("should not render when blockNumber is 0", () => {
    const { container } = render(<SimpleBlockNumberIndicator blockNumber={0} />);

    expect(container.firstChild).toBe(null);
  });
});

describe("BranchIndicator (legacy compatibility)", () => {
  it("should prefer blockNumber over branchPath", () => {
    const { container } = render(<BranchIndicator blockNumber={5} branchPath="0/1/2" isActive={false} className="test-class" />);

    expect(container.textContent).toContain("5");
  });

  it("should convert branchPath to blockNumber", () => {
    const { container } = render(<BranchIndicator branchPath="0/1/2" />);

    // Last part "2" + 1 = 3 (1-based)
    expect(container.textContent).toContain("3");
  });

  it("should return null for invalid branchPath", () => {
    const { container } = render(<BranchIndicator branchPath="invalid" />);

    expect(container.firstChild).toBe(null);
  });

  it("should return null when no valid input provided", () => {
    const { container } = render(<BranchIndicator />);

    expect(container.firstChild).toBe(null);
  });
});

describe("CompactBranchIndicator (legacy)", () => {
  it("should prefer blockNumber over branchPath", () => {
    const { container } = render(<CompactBranchIndicator blockNumber={3} branchPath="0/1/2" />);

    expect(container.textContent).toContain("3");
  });

  it("should convert branchPath to blockNumber", () => {
    const { container } = render(<CompactBranchIndicator branchPath="0/4" />);

    // "4" + 1 = 5 (1-based)
    expect(container.textContent).toContain("5");
  });
});

describe("SimplePathIndicator (legacy)", () => {
  it("should prefer blockNumber over branchPath", () => {
    const { container } = render(<SimplePathIndicator blockNumber={7} branchPath="0/1/2" />);

    expect(container.textContent).toContain("7");
  });
});

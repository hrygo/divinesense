/**
 * TimelineNode Component Tests
 *
 * Tests for the TimelineNode component including:
 * - All node types: user, thinking, tool, answer, error
 * - Custom className prop
 * - onClick callback
 * - Custom icon rendering
 * - Responsive styles
 * - Snapshot testing for visual consistency
 */

import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { TimelineNodeType } from "../../types";
import { TimelineNode } from "../TimelineNode";

// Mock i18next
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

describe("TimelineNode", () => {
  describe("Node Types", () => {
    it("should render user node with correct styling", () => {
      const { container } = render(<TimelineNode type="user" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("w-6", "h-6");
      expect(node).toHaveClass("border-2", "rounded-full");
      expect(node).toHaveClass("bg-blue-100");
      expect(node).toHaveClass("border-blue-500");
      expect(node).toHaveClass("text-blue-600");
    });

    it("should render thinking node with correct styling", () => {
      const { container } = render(<TimelineNode type="thinking" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("bg-purple-100");
      expect(node).toHaveClass("border-purple-500");
      expect(node).toHaveClass("text-purple-600");
    });

    it("should render tool node with correct styling", () => {
      const { container } = render(<TimelineNode type="tool" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("bg-card");
      expect(node).toHaveClass("border-border");
    });

    it("should render answer node with correct styling", () => {
      const { container } = render(<TimelineNode type="answer" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("bg-zinc-50");
      expect(node).toHaveClass("border-zinc-500");
      expect(node).toHaveClass("text-zinc-600");
    });

    it("should render error node with correct styling", () => {
      const { container } = render(<TimelineNode type="error" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("bg-red-100");
      expect(node).toHaveClass("border-red-500");
      expect(node).toHaveClass("text-red-600");
    });
  });

  describe("Icons", () => {
    const nodeTypes: TimelineNodeType[] = ["user", "thinking", "tool", "answer", "error"];

    it("should render default icon for each node type", () => {
      nodeTypes.forEach((type) => {
        const { container } = render(<TimelineNode type={type} />);
        const node = container.firstChild as HTMLElement;

        // Should contain an SVG icon
        const svg = node.querySelector("svg");
        expect(svg).toBeInTheDocument();
        expect(svg).toHaveClass("w-3.5", "h-3.5");
      });
    });

    it("should render custom icon when provided", () => {
      const customIcon = (
        <svg data-testid="custom-icon" className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
          <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
        </svg>
      );

      const { container } = render(<TimelineNode type="user" icon={customIcon} />);
      const node = container.firstChild as HTMLElement;

      expect(node.querySelector('[data-testid="custom-icon"]')).toBeInTheDocument();
    });

    it("should prioritize custom icon over default icon", () => {
      const customIcon = <span data-testid="custom-icon">X</span>;

      const { container } = render(<TimelineNode type="user" icon={customIcon} />);
      const node = container.firstChild as HTMLElement;

      // Custom icon should be present
      expect(node.querySelector('[data-testid="custom-icon"]')).toBeInTheDocument();
      // Default SVG should not be present (or at least custom takes precedence)
      expect(node.textContent).toBe("X");
    });
  });

  describe("Interactions", () => {
    it("should call onClick when clicked", () => {
      const handleClick = vi.fn();

      const { container } = render(<TimelineNode type="user" onClick={handleClick} />);
      const node = container.firstChild as HTMLElement;

      node.click();

      expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it("should add cursor-pointer and hover classes when onClick is provided", () => {
      const { container } = render(<TimelineNode type="user" onClick={() => {}} />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("cursor-pointer");
      expect(node).toHaveClass("hover:scale-110");
      expect(node).toHaveClass("transition-transform");
    });

    it("should not add cursor-pointer when onClick is not provided", () => {
      const { container } = render(<TimelineNode type="user" />);

      const node = container.firstChild as HTMLElement;
      expect(node).not.toHaveClass("cursor-pointer");
      expect(node).not.toHaveClass("hover:scale-110");
    });
  });

  describe("Custom Styling", () => {
    it("should apply custom className", () => {
      const { container } = render(<TimelineNode type="user" className="custom-test-class" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("custom-test-class");
    });

    it("should merge custom className with base classes", () => {
      const { container } = render(
        <TimelineNode type="user" className="custom-class another-class" />,
      );

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("w-6"); // base class
      expect(node).toHaveClass("custom-class"); // custom class
      expect(node).toHaveClass("another-class"); // custom class
    });

    it("should maintain type-specific colors with custom className", () => {
      const { container } = render(<TimelineNode type="thinking" className="custom-class" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("bg-purple-100"); // type-specific
      expect(node).toHaveClass("custom-class"); // custom
    });
  });

  describe("Accessibility", () => {
    it("should have aria-label for each node type", () => {
      const nodeTypes: TimelineNodeType[] = ["user", "thinking", "tool", "answer", "error"];

      nodeTypes.forEach((type) => {
        const { container } = render(<TimelineNode type={type} />);
        const node = container.firstChild as HTMLElement;

        expect(node).toHaveAttribute("aria-label", `${type} node`);
      });
    });
  });

  describe("Layout Classes", () => {
    it("should have flex centering classes", () => {
      const { container } = render(<TimelineNode type="user" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("flex");
      expect(node).toHaveClass("items-center");
      expect(node).toHaveClass("justify-center");
    });

    it("should have shrink-0 to prevent flex shrinking", () => {
      const { container } = render(<TimelineNode type="user" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("shrink-0");
    });

    it("should use consistent size from TIMELINE_NODE_CONFIG", () => {
      const { container } = render(<TimelineNode type="user" />);

      const node = container.firstChild as HTMLElement;
      // w-6 h-6 from TIMELINE_NODE_CONFIG.size
      expect(node).toHaveClass("w-6", "h-6");
    });
  });

  describe("Dark Mode Support", () => {
    it("should include dark mode classes for user node", () => {
      const { container } = render(<TimelineNode type="user" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("dark:bg-blue-900/40");
      expect(node).toHaveClass("dark:text-blue-400");
    });

    it("should include dark mode classes for thinking node", () => {
      const { container } = render(<TimelineNode type="thinking" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("dark:bg-purple-900/40");
      expect(node).toHaveClass("dark:text-purple-400");
    });

    it("should include dark mode classes for answer node", () => {
      const { container } = render(<TimelineNode type="answer" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("dark:bg-zinc-800/40");
      expect(node).toHaveClass("dark:text-zinc-400");
    });

    it("should include dark mode classes for error node", () => {
      const { container } = render(<TimelineNode type="error" />);

      const node = container.firstChild as HTMLElement;
      expect(node).toHaveClass("dark:bg-red-900/30");
      expect(node).toHaveClass("dark:text-red-400");
    });
  });

  describe("Snapshot Tests", () => {
    it("should match snapshot for user node", () => {
      const { container } = render(<TimelineNode type="user" />);
      expect(container.firstChild).toMatchSnapshot();
    });

    it("should match snapshot for thinking node", () => {
      const { container } = render(<TimelineNode type="thinking" />);
      expect(container.firstChild).toMatchSnapshot();
    });

    it("should match snapshot for tool node", () => {
      const { container } = render(<TimelineNode type="tool" />);
      expect(container.firstChild).toMatchSnapshot();
    });

    it("should match snapshot for answer node", () => {
      const { container } = render(<TimelineNode type="answer" />);
      expect(container.firstChild).toMatchSnapshot();
    });

    it("should match snapshot for error node", () => {
      const { container } = render(<TimelineNode type="error" />);
      expect(container.firstChild).toMatchSnapshot();
    });

    it("should match snapshot with onClick handler", () => {
      const { container } = render(<TimelineNode type="user" onClick={() => {}} />);
      expect(container.firstChild).toMatchSnapshot();
    });

    it("should match snapshot with custom className", () => {
      const { container } = render(<TimelineNode type="user" className="custom-class" />);
      expect(container.firstChild).toMatchSnapshot();
    });

    it("should match snapshot with custom icon", () => {
      const customIcon = (
        <svg data-testid="custom-icon" className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
          <path d="M12 2L2 7l10 5 10-5-10-5z" />
        </svg>
      );
      const { container } = render(<TimelineNode type="user" icon={customIcon} />);
      expect(container.firstChild).toMatchSnapshot();
    });
  });

  describe("Edge Cases", () => {
    it("should render without crashing when all props are provided", () => {
      const customIcon = <span data-testid="icon">X</span>;
      const handleClick = vi.fn();

      expect(() =>
        render(
          <TimelineNode
            type="user"
            icon={customIcon}
            className="test-class"
            onClick={handleClick}
          />,
        ),
      ).not.toThrow();
    });

    it("should handle rapid onClick calls", () => {
      const handleClick = vi.fn();
      const { container } = render(<TimelineNode type="user" onClick={handleClick} />);
      const node = container.firstChild as HTMLElement;

      // Simulate rapid clicks
      node.click();
      node.click();
      node.click();

      expect(handleClick).toHaveBeenCalledTimes(3);
    });
  });

  describe("Responsive Behavior", () => {
    it("should maintain consistent sizing across breakpoints", () => {
      // TimelineNode uses fixed w-6 h-6 classes which are breakpoint-agnostic
      const { container } = render(<TimelineNode type="user" />);
      const node = container.firstChild as HTMLElement;

      // Fixed size classes should be present
      expect(node).toHaveClass("w-6", "h-6");

      // No responsive breakpoint variants should be present
      expect(node.className).not.toMatch(/(sm:|md:|lg:|xl:)/);
    });

    it("should support hover effects only when interactive", () => {
      // Without onClick - no hover classes
      const { container: noClick } = render(<TimelineNode type="user" />);
      const nodeNoClick = noClick.firstChild as HTMLElement;
      expect(nodeNoClick.className).not.toContain("hover:");

      // With onClick - hover classes present
      const { container: withClick } = render(<TimelineNode type="user" onClick={() => {}} />);
      const nodeWithClick = withClick.firstChild as HTMLElement;
      expect(nodeWithClick.className).toContain("hover:scale-110");
    });
  });
});

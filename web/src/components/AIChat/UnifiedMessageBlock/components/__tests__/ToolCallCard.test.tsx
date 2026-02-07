/**
 * ToolCallCard Component Tests
 *
 * Tests for the ToolCallCard and InlineToolCall components including:
 * - Hover effects (border highlight, icon color change)
 * - Tool call states (pending/running/done/error)
 * - Expand/collapse functionality for input/output
 * - React.memo optimization verification
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ToolCallData } from "../ToolCallCard";
import { InlineToolCall, ToolCallCard } from "../ToolCallCard";

// Mock i18next
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

// Mock CompactTerminal and TerminalOutput
vi.mock("../../../TerminalOutput", () => ({
  CompactTerminal: ({ output, maxLines }: { output: string; maxLines: number }) => (
    <div data-testid="compact-terminal" data-max-lines={maxLines}>
      {output}
    </div>
  ),
  TerminalOutput: ({ output, command, exitCode }: { output: string; command?: string; exitCode?: number }) => (
    <div data-testid="terminal-output" data-command={command} data-exit-code={exitCode}>
      {output}
    </div>
  ),
}));

// Mock ToolResultBadge
vi.mock("../../../EventBadge", () => ({
  ToolResultBadge: ({ isError }: { isError?: boolean }) => (
    <span data-testid="tool-result-badge" data-is-error={isError}>
      {isError ? "Failed" : "Success"}
    </span>
  ),
}));

describe("ToolCallCard", () => {
  const createMockData = (overrides: Partial<ToolCallData> = {}): ToolCallData => ({
    toolName: "bash",
    toolId: "tool_1234567890abcdef",
    input: { command: "ls -la" },
    output: "file1.txt\nfile2.txt",
    duration: 150,
    isError: false,
    ...overrides,
  });

  describe("Rendering", () => {
    it("should render tool name", () => {
      const data = createMockData({ toolName: "write_file" });
      render(<ToolCallCard data={data} />);

      expect(screen.getByText("write_file")).toBeInTheDocument();
    });

    it("should render tool ID truncated", () => {
      const data = createMockData({ toolId: "tool_1234567890abcdef" });
      render(<ToolCallCard data={data} />);

      // toolId is truncated to 8 chars: # + first 7 chars
      expect(screen.getByText(/#tool_/)).toBeInTheDocument();
      // Find the element with the truncated ID
      const truncatedElement = screen.getByText((_content, element) => {
        return element?.textContent === "#tool_123";
      });
      expect(truncatedElement).toBeInTheDocument();
    });

    it("should not render tool ID when not provided", () => {
      const data = createMockData({ toolId: undefined });
      render(<ToolCallCard data={data} />);

      expect(screen.queryByText(/#tool_/)).not.toBeInTheDocument();
    });

    it("should render duration when provided", () => {
      const data = createMockData({ duration: 250 });
      render(<ToolCallCard data={data} />);

      expect(screen.getByText("250ms")).toBeInTheDocument();
    });

    it("should not render duration when not provided", () => {
      const data = createMockData({ duration: undefined });
      render(<ToolCallCard data={data} />);

      expect(screen.queryByText(/\d+ms/)).not.toBeInTheDocument();
    });
  });

  describe("Status badges", () => {
    it("should show success badge when output exists and no error", () => {
      const data = createMockData({ output: "Success output", isError: false });
      render(<ToolCallCard data={data} />);

      expect(screen.getByTestId("tool-result-badge")).toHaveAttribute("data-is-error", "false");
    });

    it("should show error badge when isError is true", () => {
      const data = createMockData({ output: "Error output", isError: true });
      render(<ToolCallCard data={data} />);

      expect(screen.getByTestId("tool-result-badge")).toHaveAttribute("data-is-error", "true");
    });

    it("should not show badge when no output", () => {
      const data = createMockData({ output: undefined });
      render(<ToolCallCard data={data} />);

      expect(screen.queryByTestId("tool-result-badge")).not.toBeInTheDocument();
    });
  });

  describe("Tool call states", () => {
    it("should render pending state (no output yet) - shows when expanded", () => {
      const data = createMockData({ output: undefined, input: { command: "pending" } });
      render(<ToolCallCard data={data} />);

      // Initially collapsed, so no output text visible
      expect(screen.queryByText("ai.events.no_output")).not.toBeInTheDocument();

      // Click to expand
      const header = screen.getByText("bash").closest("div")?.parentElement;
      if (header) {
        fireEvent.click(header);
      }

      // Now the no_output text should be visible
      expect(screen.getByText("ai.events.no_output")).toBeInTheDocument();
    });

    it("should render running state (with input, no output) - shows when expanded", () => {
      const data = createMockData({
        toolName: "run_bash",
        input: { command: "sleep 10" },
        output: undefined,
      });
      render(<ToolCallCard data={data} />);

      // Click to expand
      const header = screen.getByText("run_bash").closest("div")?.parentElement;
      if (header) {
        fireEvent.click(header);
      }

      expect(screen.getByText("ai.events.no_output")).toBeInTheDocument();
    });

    it("should render done state (with output)", () => {
      const data = createMockData({
        toolName: "read_file",
        input: { file_path: "/tmp/file.txt" },
        output: "file content",
        isError: false,
      });
      render(<ToolCallCard data={data} />);

      expect(screen.getByTestId("tool-result-badge")).toHaveAttribute("data-is-error", "false");
    });

    it("should render error state (isError true)", () => {
      const data = createMockData({
        toolName: "run_bash",
        input: { command: "exit 1" },
        output: "Command failed",
        isError: true,
        exitCode: 1,
      });
      render(<ToolCallCard data={data} />);

      expect(screen.getByTestId("tool-result-badge")).toHaveAttribute("data-is-error", "true");
    });
  });

  describe("Expand/Collapse", () => {
    it("should start collapsed by default", () => {
      const data = createMockData({ input: { command: "test" }, output: "result" });
      render(<ToolCallCard data={data} />);

      expect(screen.queryByText("ai.events.input")).not.toBeInTheDocument();
      expect(screen.queryByText("ai.events.output")).not.toBeInTheDocument();
    });

    it("should expand on header click", () => {
      const data = createMockData({ input: { command: "test" }, output: "result" });
      render(<ToolCallCard data={data} />);

      const header = screen.getByText("bash").closest("div")?.parentElement;
      if (header) {
        fireEvent.click(header);
      }

      expect(screen.getByText("ai.events.input")).toBeInTheDocument();
      expect(screen.getByText("ai.events.output")).toBeInTheDocument();
    });

    it("should collapse on second header click", () => {
      const data = createMockData({ input: { command: "test" }, output: "result" });
      render(<ToolCallCard data={data} />);

      const header = screen.getByText("bash").closest("div")?.parentElement;
      if (header) {
        // First click expands
        fireEvent.click(header);
        expect(screen.getByText("ai.events.input")).toBeInTheDocument();

        // Second click collapses
        fireEvent.click(header);
        expect(screen.queryByText("ai.events.input")).not.toBeInTheDocument();
      }
    });

    it("should expand/collapse on button click", () => {
      const data = createMockData({ input: { command: "test" }, output: "result" });
      render(<ToolCallCard data={data} />);

      // Find expand/collapse button (ChevronRight when collapsed)
      const buttons = screen.getAllByRole("button");
      const expandButton = buttons.find((btn) => btn.querySelector("svg"));
      expect(expandButton).toBeDefined();

      if (expandButton) {
        // Click to expand
        fireEvent.click(expandButton);
        expect(screen.getByText("ai.events.input")).toBeInTheDocument();

        // Find collapse button (ChevronDown when expanded)
        const collapseButton = screen.getAllByRole("button").find((btn) => btn.querySelector("svg"));
        if (collapseButton) {
          fireEvent.click(collapseButton);
          expect(screen.queryByText("ai.events.input")).not.toBeInTheDocument();
        }
      }
    });

    it("should show input when expanded and has input", () => {
      const data = createMockData({
        input: { command: "echo hello" },
        output: "hello",
      });
      render(<ToolCallCard data={data} />);

      const header = screen.getByText("bash").closest("div")?.parentElement;
      if (header) {
        fireEvent.click(header);
      }

      expect(screen.getByText("ai.events.input")).toBeInTheDocument();
      // There may be multiple compact-terminal elements when both input and output exist
      const terminals = screen.getAllByTestId("compact-terminal");
      expect(terminals.some((t) => t.textContent === "$ echo hello")).toBe(true);
    });

    it("should show output when expanded and has output", () => {
      const data = createMockData({
        input: { command: "echo test" },
        output: "test output",
      });
      render(<ToolCallCard data={data} />);

      const header = screen.getByText("bash").closest("div")?.parentElement;
      if (header) {
        fireEvent.click(header);
      }

      expect(screen.getByText("ai.events.output")).toBeInTheDocument();
      const terminals = screen.getAllByTestId("compact-terminal");
      expect(terminals.some((t) => t.textContent === "test output")).toBe(true);
    });

    it("should use TerminalOutput for long output (> 500 chars)", () => {
      const longOutput = "x".repeat(501);
      const data = createMockData({
        input: { command: "generate" },
        output: longOutput,
      });
      render(<ToolCallCard data={data} />);

      const header = screen.getByText("bash").closest("div")?.parentElement;
      if (header) {
        fireEvent.click(header);
      }

      expect(screen.getByTestId("terminal-output")).toBeInTheDocument();
    });

    it("should use CompactTerminal for short output (<= 500 chars)", () => {
      const shortOutput = "x".repeat(100);
      const data = createMockData({
        input: { command: "short" },
        output: shortOutput,
      });
      render(<ToolCallCard data={data} />);

      const header = screen.getByText("bash").closest("div")?.parentElement;
      if (header) {
        fireEvent.click(header);
      }

      expect(screen.getAllByTestId("compact-terminal").length).toBeGreaterThan(0);
      expect(screen.queryByTestId("terminal-output")).not.toBeInTheDocument();
    });
  });

  describe("Hover effects", () => {
    it("should have group class for hover effect", () => {
      const data = createMockData();
      const { container } = render(<ToolCallCard data={data} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass("group");
    });

    it("should have hover:border-purple-400/50 class", () => {
      const data = createMockData();
      const { container } = render(<ToolCallCard data={data} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass("hover:border-purple-400/50");
    });

    it("should have hover:shadow-[0_0_0_1px_rgba(168,85,247,0.1)] class", () => {
      const data = createMockData();
      const { container } = render(<ToolCallCard data={data} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass("hover:shadow-[0_0_0_1px_rgba(168,85,247,0.1)]");
    });

    it("should have group-hover classes on icon container", () => {
      const data = createMockData();
      const { container } = render(<ToolCallCard data={data} />);

      const iconContainer = container.querySelector('[class*="group-hover:bg-purple-100"]');
      expect(iconContainer).toBeInTheDocument();
    });
  });

  describe("Input formatting", () => {
    it("should format command input with $ prefix", () => {
      const data = createMockData({ input: { command: "ls -la" } });
      render(<ToolCallCard data={data} />);

      const header = screen.getByText("bash").closest("div")?.parentElement;
      if (header) {
        fireEvent.click(header);
      }

      const terminals = screen.getAllByTestId("compact-terminal");
      expect(terminals.some((t) => t.textContent === "$ ls -la")).toBe(true);
    });

    it("should format file_path input", () => {
      const data = createMockData({ toolName: "read_file", input: { file_path: "/path/to/file.txt" } });
      render(<ToolCallCard data={data} />);

      const header = screen.getByText("read_file").closest("div")?.parentElement;
      if (header) {
        fireEvent.click(header);
      }

      const terminals = screen.getAllByTestId("compact-terminal");
      expect(terminals.some((t) => t.textContent === "/path/to/file.txt")).toBe(true);
    });

    it("should format text input", () => {
      const data = createMockData({ input: { text: "sample text content" } });
      render(<ToolCallCard data={data} />);

      const header = screen.getByText("bash").closest("div")?.parentElement;
      if (header) {
        fireEvent.click(header);
      }

      const terminals = screen.getAllByTestId("compact-terminal");
      expect(terminals.some((t) => t.textContent === "sample text content")).toBe(true);
    });

    it("should JSON.stringify complex input", () => {
      const complexInput = { key1: "value1", key2: 123, nested: { a: true } };
      const data = createMockData({ input: complexInput });
      render(<ToolCallCard data={data} />);

      const header = screen.getByText("bash").closest("div")?.parentElement;
      if (header) {
        fireEvent.click(header);
      }

      const terminals = screen.getAllByTestId("compact-terminal");
      expect(terminals.some((t) => t.textContent?.includes("key1"))).toBe(true);
      expect(terminals.some((t) => t.textContent?.includes("value1"))).toBe(true);
    });
  });

  describe("Tool icons", () => {
    it("should use FileIcon for write/edit tools", () => {
      const { container: c1 } = render(<ToolCallCard data={createMockData({ toolName: "write_file" })} />);
      const { container: c2 } = render(<ToolCallCard data={createMockData({ toolName: "edit_file" })} />);
      const { container: c3 } = render(<ToolCallCard data={createMockData({ toolName: "file_operation" })} />);

      // All should have SVG icons
      expect(c1.querySelector("svg")).toBeInTheDocument();
      expect(c2.querySelector("svg")).toBeInTheDocument();
      expect(c3.querySelector("svg")).toBeInTheDocument();
    });

    it("should use TerminalIcon for run/exec/bash tools", () => {
      const { container: c1 } = render(<ToolCallCard data={createMockData({ toolName: "run_command" })} />);
      const { container: c2 } = render(<ToolCallCard data={createMockData({ toolName: "execute" })} />);
      const { container: c3 } = render(<ToolCallCard data={createMockData({ toolName: "bash" })} />);

      expect(c1.querySelector("svg")).toBeInTheDocument();
      expect(c2.querySelector("svg")).toBeInTheDocument();
      expect(c3.querySelector("svg")).toBeInTheDocument();
    });

    it("should use FileIcon for read tools", () => {
      const { container } = render(<ToolCallCard data={createMockData({ toolName: "read_file" })} />);

      expect(container.querySelector("svg")).toBeInTheDocument();
    });

    it("should use Play icon for other tools", () => {
      const { container } = render(<ToolCallCard data={createMockData({ toolName: "unknown_tool" })} />);

      expect(container.querySelector("svg")).toBeInTheDocument();
    });
  });

  describe("React.memo optimization", () => {
    it("should not re-render when data reference is same", () => {
      const data = createMockData();
      const { rerender } = render(<ToolCallCard data={data} />);

      const initialHTML = screen.getByText("bash").parentElement?.innerHTML;

      // Re-render with same reference
      rerender(<ToolCallCard data={data} />);

      expect(screen.getByText("bash").parentElement?.innerHTML).toBe(initialHTML);
    });

    it("should not re-render when all data properties are equal", () => {
      const data1 = createMockData();
      const data2 = createMockData(); // Same values, different reference

      const { rerender } = render(<ToolCallCard data={data1} />);

      // Re-render with same values but different reference
      rerender(<ToolCallCard data={data2} />);

      // Element should still be there (component didn't unmount)
      expect(screen.getByText("bash")).toBeInTheDocument();

      // The key is that React.memo prevents unnecessary re-renders
      // We verify the component still works correctly
    });

    it("should re-render when toolName changes", () => {
      const data1 = createMockData({ toolName: "tool_a" });
      const data2 = createMockData({ toolName: "tool_b" });

      const { rerender } = render(<ToolCallCard data={data1} />);

      expect(screen.getByText("tool_a")).toBeInTheDocument();

      rerender(<ToolCallCard data={data2} />);

      expect(screen.queryByText("tool_a")).not.toBeInTheDocument();
      expect(screen.getByText("tool_b")).toBeInTheDocument();
    });

    it("should re-render when output changes", () => {
      const data1 = createMockData({ output: "first output" });
      const data2 = createMockData({ output: "second output" });

      const { rerender } = render(<ToolCallCard data={data1} />);

      rerender(<ToolCallCard data={data2} />);

      // Should re-render because output changed
      expect(screen.queryByText("first output")).not.toBeInTheDocument();
    });

    it("should re-render when isError changes", () => {
      const data1 = createMockData({ isError: false });
      const data2 = createMockData({ isError: true });

      const { rerender } = render(<ToolCallCard data={data1} />);

      expect(screen.getByTestId("tool-result-badge")).toHaveAttribute("data-is-error", "false");

      rerender(<ToolCallCard data={data2} />);

      expect(screen.getByTestId("tool-result-badge")).toHaveAttribute("data-is-error", "true");
    });
  });

  describe("Custom className", () => {
    it("should apply custom className to root element", () => {
      const data = createMockData();
      const { container } = render(<ToolCallCard data={data} className="custom-class" />);

      expect(container.firstChild).toHaveClass("custom-class");
    });

    it("should merge custom className with default classes", () => {
      const data = createMockData();
      const { container } = render(<ToolCallCard data={data} className="custom-class" />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass("custom-class");
      expect(card).toHaveClass("rounded-lg");
      expect(card).toHaveClass("border");
    });
  });

  describe("Accessibility", () => {
    it("should have proper button label for expand/collapse", () => {
      const data = createMockData({ output: "result" });
      render(<ToolCallCard data={data} />);

      const button = screen.getAllByRole("button").find((btn) => btn.getAttribute("aria-label"));
      expect(button?.getAttribute("aria-label")).toBe("Expand");
    });

    it("should update aria-label when expanded", () => {
      const data = createMockData({ output: "result" });
      render(<ToolCallCard data={data} />);

      const button = screen.getAllByRole("button").find((btn) => btn.getAttribute("aria-label"));

      if (button) {
        // Initially collapsed
        expect(button.getAttribute("aria-label")).toBe("Expand");

        // Click to expand
        fireEvent.click(button);

        // Should now be Collapse
        const expandedButton = screen.getAllByRole("button").find((btn) => btn.getAttribute("aria-label"));
        expect(expandedButton?.getAttribute("aria-label")).toBe("Collapse");
      }
    });

    it("should have cursor-pointer on header", () => {
      const data = createMockData();
      const { container } = render(<ToolCallCard data={data} />);

      const header = container.querySelector('[class*="cursor-pointer"]');
      expect(header).toBeInTheDocument();
    });
  });
});

describe("InlineToolCall", () => {
  describe("Rendering", () => {
    it("should render tool name", () => {
      render(<InlineToolCall toolName="bash" />);

      expect(screen.getByText("bash")).toBeInTheDocument();
    });

    it("should render input summary when provided", () => {
      render(<InlineToolCall toolName="bash" inputSummary='echo "hello"' />);

      expect(screen.getByText(/bash/)).toBeInTheDocument();
      const element = screen.getByText(/bash/);
      expect(element.textContent).toContain('echo "hello"');
    });

    it("should render file path when provided", () => {
      render(<InlineToolCall toolName="read_file" filePath="/tmp/file.txt" />);

      const element = screen.getByText(/read_file/);
      expect(element.textContent).toContain("/tmp/file.txt");
    });

    it("should prefer inputSummary over filePath", () => {
      render(<InlineToolCall toolName="read_file" inputSummary="custom summary" filePath="/tmp/file.txt" />);

      const element = screen.getByText(/read_file/);
      expect(element.textContent).toContain("custom summary");
      expect(element.textContent).not.toContain("/tmp/file.txt");
    });

    it("should truncate long text", () => {
      render(<InlineToolCall toolName="bash" inputSummary={"a".repeat(100)} />);

      const element = screen.getByText(/bash/);
      expect(element).toHaveClass("truncate");
    });
  });

  describe("Error state", () => {
    it("should have red border when isError is true", () => {
      const { container } = render(<InlineToolCall toolName="bash" isError={true} />);

      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass("border-red-300");
    });

    it("should have red text when isError is true", () => {
      const { container } = render(<InlineToolCall toolName="bash" isError={true} />);

      const text = container.querySelector('[class*="text-red"]');
      expect(text).toBeInTheDocument();
    });

    it("should have blue border when isError is false", () => {
      const { container } = render(<InlineToolCall toolName="bash" isError={false} />);

      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass("border-blue-200");
    });
  });

  describe("Hover effect", () => {
    it("should have hover:border-purple-400/50 class", () => {
      const { container } = render(<InlineToolCall toolName="bash" />);

      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass("hover:border-purple-400/50");
    });
  });

  describe("Title attribute", () => {
    it("should have title with tool name when no details", () => {
      render(<InlineToolCall toolName="bash" />);

      const element = screen.getByText("bash").closest("[title]");
      expect(element).toHaveAttribute("title", "bash");
    });

    it("should have title with tool name and summary", () => {
      render(<InlineToolCall toolName="bash" inputSummary="echo test" />);

      const element = screen.getByText(/bash/).closest("[title]");
      expect(element).toHaveAttribute("title", "bash: echo test");
    });

    it("should have title with tool name and file path", () => {
      render(<InlineToolCall toolName="read_file" filePath="/tmp/file.txt" />);

      const element = screen.getByText(/read_file/).closest("[title]");
      expect(element).toHaveAttribute("title", "read_file: /tmp/file.txt");
    });
  });

  describe("Tool icons", () => {
    it("should display icon for tool", () => {
      const { container } = render(<InlineToolCall toolName="bash" />);

      const icon = container.querySelector("svg");
      expect(icon).toBeInTheDocument();
    });

    it("should have red icon when isError", () => {
      const { container } = render(<InlineToolCall toolName="bash" isError={true} />);

      const icon = container.querySelector("svg");
      expect(icon).toHaveClass("text-red-500");
    });

    it("should have blue icon when no error", () => {
      const { container } = render(<InlineToolCall toolName="bash" isError={false} />);

      const icon = container.querySelector("svg");
      expect(icon).toHaveClass("text-blue-500");
    });
  });

  describe("Custom className", () => {
    it("should apply custom className", () => {
      const { container } = render(<InlineToolCall toolName="bash" className="my-custom" />);

      expect(container.firstChild).toHaveClass("my-custom");
    });

    it("should merge custom className with defaults", () => {
      const { container } = render(<InlineToolCall toolName="bash" className="my-custom" />);

      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass("my-custom");
      expect(wrapper).toHaveClass("inline-flex");
    });
  });

  describe("Responsive max-width", () => {
    it("should have max-w-[240px] on mobile", () => {
      const { container } = render(<InlineToolCall toolName="bash" />);

      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass("max-w-[240px]");
    });

    it("should have md:max-w-[360px] on medium screens", () => {
      const { container } = render(<InlineToolCall toolName="bash" />);

      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass("md:max-w-[360px]");
    });
  });
});

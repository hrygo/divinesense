import { memo, useMemo, useRef } from "react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { CodeBlock } from "@/components/MemoContent/CodeBlock";
import { cn } from "@/lib/utils";
import { ParrotAgentType } from "@/types/parrot";
import TypingCursor from "./TypingCursor";

type CodeComponentProps = React.ComponentProps<"code"> & { inline?: boolean };

interface StreamingMarkdownProps {
  content: string;
  isStreaming?: boolean;
  parrotId?: ParrotAgentType;
  className?: string;
  enableTypingCursor?: boolean;
  onContentChange?: (complete: string, streaming: string) => void;
}

/**
 * StreamingMarkdown - 增量流式 Markdown 渲染器
 *
 * Phase 2 优化:
 * - 检测句子边界，仅对完整句子进行完整渲染
 * - 正在输入的部分使用打字光标动画
 * - 减少流式更新时的 DOM 重排
 */
const StreamingMarkdown = memo(function StreamingMarkdown({
  content,
  isStreaming = false,
  parrotId,
  className,
  enableTypingCursor = true,
  onContentChange,
}: StreamingMarkdownProps) {
  // 缓存上次完整渲染的内容，避免重复解析
  const lastRenderedRef = useRef<string>("");

  // 检测句子边界 - 支持中英文标点
  const { completePart, streamingPart } = useMemo(() => {
    if (!isStreaming) {
      return { completePart: content, streamingPart: "" };
    }

    // 句子结束标点符号
    const sentenceEnders = /[。！？.!?。！？\n]/;
    const lines = content.split("\n");

    let completeEnd = 0;
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      // 如果是最后一行，检查是否有完整句子
      if (i === lines.length - 1) {
        const lastSentenceEnd = line.lastIndexOf(line.match(sentenceEnders)?.[0] || "");
        if (lastSentenceEnd !== -1) {
          completeEnd += lastSentenceEnd + 1;
        }
      } else {
        // 非最后一行，整行都是完整的
        completeEnd += line.length + 1; // +1 for newline
      }
    }

    return {
      completePart: content.slice(0, completeEnd),
      streamingPart: content.slice(completeEnd),
    };
  }, [content, isStreaming]);

  // 触发内容变化回调
  if (onContentChange && (completePart !== lastRenderedRef.current || !isStreaming)) {
    lastRenderedRef.current = completePart;
    onContentChange(completePart, streamingPart);
  }

  // 代码块检测 - 如果正在流式输出代码块，不进行增量渲染
  const isInCodeBlock = useMemo(() => {
    const codeBlockCount = (content.match(/```/g) || []).length;
    return codeBlockCount % 2 !== 0;
  }, [content]);

  // 如果在代码块中，直接渲染全部内容
  if (isInCodeBlock) {
    return (
      <div className={cn("prose prose-sm dark:prose-invert max-w-none", className)}>
        <ReactMarkdown
          remarkPlugins={[remarkGfm, remarkBreaks]}
          components={{
            a: ({ node, ...props }) => <a {...props} className="text-blue-500 hover:underline" target="_blank" rel="noopener noreferrer" />,
            p: ({ node, ...props }) => <p {...props} className="mb-1 last:mb-0" />,
            pre: ({ node, ...props }) => <CodeBlock {...props} hideCopy={true} />,
            code: ({ className, children, inline, ...props }: CodeComponentProps) => {
              return inline ? (
                <code className={cn("px-1.5 py-0.5 rounded-md bg-muted text-xs", className)} {...props}>
                  {children}
                </code>
              ) : (
                <code className={className} {...props}>
                  {children}
                </code>
              );
            },
          }}
        >
          {content}
        </ReactMarkdown>
      </div>
    );
  }

  return (
    <div className={cn("prose prose-sm dark:prose-invert max-w-none", className)}>
      {/* 完整部分 - 静态渲染 */}
      {completePart && (
        <ReactMarkdown
          remarkPlugins={[remarkGfm, remarkBreaks]}
          components={{
            a: ({ node, ...props }) => <a {...props} className="text-blue-500 hover:underline" target="_blank" rel="noopener noreferrer" />,
            p: ({ node, ...props }) => <p {...props} className="mb-1 last:mb-0" />,
            pre: ({ node, ...props }) => <CodeBlock {...props} hideCopy={true} />,
            code: ({ className, children, inline, ...props }: CodeComponentProps) => {
              return inline ? (
                <code className={cn("px-1.5 py-0.5 rounded-md bg-muted text-xs", className)} {...props}>
                  {children}
                </code>
              ) : (
                <code className={className} {...props}>
                  {children}
                </code>
              );
            },
          }}
        >
          {completePart}
        </ReactMarkdown>
      )}

      {/* 流式部分 - 带光标动画 */}
      {isStreaming && streamingPart && (
        <span className="inline-flex items-center">
          <span>{streamingPart}</span>
          {enableTypingCursor && (
            <span className="ml-0.5">
              <TypingCursor active={true} parrotId={parrotId} variant="cursor" />
            </span>
          )}
        </span>
      )}

      {/* 流式输出中但无新内容时显示光标 */}
      {isStreaming && !streamingPart && enableTypingCursor && (
        <span className="ml-1">
          <TypingCursor active={true} parrotId={parrotId} variant="cursor" />
        </span>
      )}
    </div>
  );
});

export default StreamingMarkdown;

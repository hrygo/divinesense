/**
 * StreamingText - 流式文本组件
 *
 * 实现类似 ChatGPT 的逐字打字效果，支持 Markdown 渲染。
 *
 * ## 特性
 * - 平滑的打字动画（可配置速度）
 * - 支持 Markdown 渲染（流式解析）
 * - 流式光标效果
 * - 自动滚动到最新内容
 */

import { useEffect, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { CodeBlock } from "@/components/MemoContent/CodeBlock";
import { cn } from "@/lib/utils";

type CodeComponentProps = React.ComponentProps<"code"> & { inline?: boolean };

export interface StreamingTextProps {
  /** 完整的文本内容 */
  content: string;
  /** 是否正在流式传输 */
  isStreaming?: boolean;
  /** 每帧显示的字符数（控制打字速度） */
  charsPerFrame?: number;
  /** 流式更新间隔（毫秒） */
  updateInterval?: number;
  /** 是否启用打字效果 */
  enableTypingEffect?: boolean;
  /** 自定义类名 */
  className?: string;
}

/**
 * 流式文本组件
 *
 * 当 isStreaming=true 时，显示打字效果和光标
 * 当 isStreaming=false 时，显示完整内容
 */
export function StreamingText({
  content,
  isStreaming = false,
  charsPerFrame = 2,
  updateInterval = 30,
  enableTypingEffect = true,
  className,
}: StreamingTextProps) {
  const [displayText, setDisplayText] = useState("");
  const [isComplete, setIsComplete] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const animationFrameRef = useRef<number>();
  const lastUpdateRef = useRef<number>(0);
  const currentIndexRef = useRef(0);

  // 重置状态当 content 变化时
  useEffect(() => {
    if (!enableTypingEffect) {
      setDisplayText(content);
      setIsComplete(true);
      return;
    }

    // 如果不是流式状态，直接显示完整内容
    if (!isStreaming) {
      setDisplayText(content);
      setIsComplete(true);
      currentIndexRef.current = content.length;
      return;
    }

    // 流式状态：重置并开始动画
    setDisplayText("");
    setIsComplete(false);
    currentIndexRef.current = 0;
    lastUpdateRef.current = performance.now();

    const animate = (timestamp: number) => {
      const elapsed = timestamp - lastUpdateRef.current;

      if (elapsed >= updateInterval && currentIndexRef.current < content.length) {
        // 计算这次要显示的字符数
        const charsToShow = Math.min(charsPerFrame, content.length - currentIndexRef.current);
        const newIndex = currentIndexRef.current + charsToShow;
        currentIndexRef.current = newIndex;
        lastUpdateRef.current = timestamp;

        // 截取到当前的索引
        setDisplayText(content.slice(0, newIndex));

        // 自动滚动到底部
        if (containerRef.current) {
          containerRef.current.scrollTop = containerRef.current.scrollHeight;
        }
      }

      // 继续动画直到完成
      if (currentIndexRef.current < content.length) {
        animationFrameRef.current = requestAnimationFrame(animate);
      } else {
        setIsComplete(true);
      }
    };

    animationFrameRef.current = requestAnimationFrame(animate);

    return () => {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
    };
  }, [content, isStreaming, enableTypingEffect, charsPerFrame, updateInterval]);

  // 当 content 更新时（流式期间），同步更新显示
  useEffect(() => {
    if (isStreaming && enableTypingEffect) {
      // 流式期间：让动画循环处理
      // 但如果 content 被外部更新（如 SSE 推送），需要同步
      const targetLength = content.length;
      if (currentIndexRef.current > targetLength) {
        // 内容被截断了（重置情况）
        currentIndexRef.current = targetLength;
        setDisplayText(content.slice(0, targetLength));
      }
      // 如果 content 增加了，动画循环会自动处理
    }
  }, [content, isStreaming, enableTypingEffect]);

  // 清理动画帧
  useEffect(() => {
    return () => {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
    };
  }, []);

  return (
    <div ref={containerRef} className={cn("prose prose-sm dark:prose-invert max-w-none", className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        components={{
          a: ({ node, ...props }) => <a {...props} className="text-blue-500 hover:underline" target="_blank" rel="noopener noreferrer" />,
          p: ({ node, ...props }) => <p {...props} className="mb-1 last:mb-0 text-sm leading-relaxed" />,
          ul: ({ node, ...props }) => <ul {...props} className="list-disc pl-5 mb-2 space-y-1" />,
          ol: ({ node, ...props }) => <ol {...props} className="list-decimal pl-5 mb-2 space-y-1" />,
          li: ({ node, ...props }) => <li {...props} className="pl-1" />,
          h1: ({ node, ...props }) => <h1 {...props} className="text-xl font-bold mb-2 mt-4 first:mt-0" />,
          h2: ({ node, ...props }) => <h2 {...props} className="text-lg font-bold mb-2 mt-3" />,
          h3: ({ node, ...props }) => <h3 {...props} className="text-base font-bold mb-1 mt-2" />,
          blockquote: ({ node, ...props }) => (
            <blockquote {...props} className="border-l-4 border-primary/30 pl-4 py-1 my-2 bg-muted/30 italic rounded-r-lg" />
          ),
          table: ({ node, ...props }) => (
            <div className="my-4 w-full overflow-x-auto rounded-lg border border-border shadow-sm">
              <table className="w-full text-sm" {...props} />
            </div>
          ),
          thead: ({ node, ...props }) => <thead className="bg-muted/50 text-xs uppercase" {...props} />,
          tbody: ({ node, ...props }) => <tbody className="divide-y divide-border" {...props} />,
          tr: ({ node, ...props }) => <tr className="hover:bg-muted/50 transition-colors" {...props} />,
          th: ({ node, ...props }) => <th className="px-4 py-2.5 text-left font-medium text-muted-foreground tracking-wider" {...props} />,
          td: ({ node, ...props }) => <td className="px-4 py-2.5 whitespace-pre-wrap" {...props} />,
          pre: ({ node, ...props }) => <CodeBlock {...props} hideCopy={true} />,
          code: ({ className, children, inline, ...props }: CodeComponentProps) => {
            return inline ? (
              <code
                className={cn("px-1.5 py-0.5 rounded-md bg-muted/80 font-mono text-xs text-secondary-foreground", className)}
                {...props}
              >
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
        {displayText}
      </ReactMarkdown>

      {/* 流式光标效果 */}
      {isStreaming && !isComplete && (
        <span className={cn("inline-block w-0.5 h-4 ml-0.5 bg-current align-middle animate-pulse", "transition-opacity duration-200")} />
      )}
    </div>
  );
}

/**
 * 简化版流式文本组件
 *
 * 直接显示内容，仅添加光标效果，不做打字动画
 * 适用于高性能需求场景
 */
export function StreamingTextSimple({
  content,
  isStreaming = false,
  className,
}: {
  content: string;
  isStreaming?: boolean;
  className?: string;
}) {
  return (
    <div className={cn("relative", className)}>
      <span>{content}</span>
      {isStreaming && (
        <span className={cn("inline-block w-0.5 h-4 ml-0.5 bg-current align-middle animate-pulse", "transition-opacity duration-200")} />
      )}
    </div>
  );
}

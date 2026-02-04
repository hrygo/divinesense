import { ReactNode } from "react";
import { cn } from "@/lib/utils";

export interface MobileTableCardColumn<T = unknown> {
  key: string;
  label?: string;
  render?: (value: unknown, row: T) => ReactNode;
}

interface MobileTableCardProps<T = unknown> {
  title?: string;
  subtitle?: string;
  columns: MobileTableCardColumn<T>[];
  data: T[];
  emptyMessage?: string;
  className?: string;
  getRowKey?: (row: T, index: number) => string;
  renderActions?: (row: T) => ReactNode;
}

/**
 * Mobile-optimized table card that displays tabular data as a list of cards.
 * Each card represents one row from the original table.
 * Suitable for displaying access tokens, chat apps credentials, etc.
 */
function MobileTableCard<T>({
  title,
  subtitle,
  columns,
  data,
  emptyMessage = "暂无数据",
  className,
  getRowKey,
  renderActions,
}: MobileTableCardProps<T>) {
  if (data.length === 0) {
    return <div className={cn("px-4 py-8 text-center text-sm text-muted-foreground", className)}>{emptyMessage}</div>;
  }

  return (
    <div className={cn("flex flex-col", className)}>
      {title && (
        <div className="px-4 pt-4 pb-2">
          <div className="text-sm font-medium text-foreground">{title}</div>
          {subtitle && <div className="text-xs text-muted-foreground mt-0.5">{subtitle}</div>}
        </div>
      )}

      {data.map((row, rowIndex) => {
        const rowKey = getRowKey ? getRowKey(row, rowIndex) : rowIndex.toString();

        return (
          <div key={rowKey} className="mx-4 my-2 p-4 bg-background border border-border rounded-lg">
            {columns.map((column) => {
              const value = (row as Record<string, unknown>)[column.key];
              const content = column.render ? column.render(value, row) : (value as ReactNode);

              // Skip if content is null/undefined
              if (content === null || content === undefined) {
                return null;
              }

              return (
                <div key={column.key} className="flex items-start justify-between py-2 border-b border-border last:border-0 last:pb-0">
                  <div className="text-sm text-muted-foreground">{column.label || column.key}</div>
                  <div className="text-sm text-foreground text-right ml-4 flex-1">{content}</div>
                </div>
              );
            })}

            {renderActions && (
              <div className="flex items-center justify-end gap-2 mt-3 pt-3 border-t border-border">{renderActions(row)}</div>
            )}
          </div>
        );
      })}
    </div>
  );
}

export default MobileTableCard;

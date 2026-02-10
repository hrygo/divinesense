/**
 * FixedEditor - Responsive Bottom Memo Editor
 *
 * 响应式底部编辑器：
 * - 移动端：渐进式展开（QuickEditor → Expanded）
 * - PC 端：完整工具栏
 *
 * ## 架构
 * ```
 * FixedEditor
 * ├── MobileEditor (sm 断点以下)
 * │   ├── QuickInput (默认)
 * │   └── BottomSheet (展开后)
 * └── PCEditor (md 断点及以上)
 *     └── FullToolbar
 * ```
 */

import { memo, useMemo } from "react";
import useMediaQuery from "@/hooks/useMediaQuery";
import { useTranslate } from "@/utils/i18n";
import { MobileEditor } from "./MobileEditor";
import { PCEditor } from "./PCEditor";

export interface FixedEditorProps {
  placeholder?: string;
  className?: string;
  onConfirm?: () => void;
}

/**
 * Custom comparison function for FixedEditor memo.
 * Only re-renders when placeholder or className changes, ignoring onConfirm.
 * This is safe because onConfirm is an event callback that doesn't affect rendering.
 */
function arePropsEqual(prevProps: Readonly<FixedEditorProps>, nextProps: Readonly<FixedEditorProps>): boolean {
  return (
    prevProps.placeholder === nextProps.placeholder && prevProps.className === nextProps.className
    // Intentionally ignore onConfirm - it's an event handler that doesn't affect output
  );
}

export const FixedEditor = memo(function FixedEditor({ placeholder, className, onConfirm }: FixedEditorProps) {
  const t = useTranslate();
  // md 断点 (768px) 及以上使用 PC 编辑器
  const isPC = useMediaQuery("md");

  const finalPlaceholder = useMemo(() => {
    return placeholder || t("editor.any-thoughts");
  }, [placeholder, t]);

  if (isPC) {
    return <PCEditor placeholder={finalPlaceholder} className={className} onConfirm={onConfirm} />;
  }

  return <MobileEditor placeholder={finalPlaceholder} className={className} onConfirm={onConfirm} />;
}, arePropsEqual);

FixedEditor.displayName = "FixedEditor";

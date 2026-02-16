# AI Internal Utilities (`ai/internal`)

`internal` 包包含 AI 模块内部使用的通用工具函数和辅助代码。这些代码不作为公共 API 暴露给外部模块。

## Subpackages

### `strutil`
提供字符串处理工具，特别是针对多字节字符（如中文）的安全处理。
*   **`Truncate(s string, maxLen int)`**: 安全地按字符数（Rune）截断字符串，防止切断多字节字符导致乱码或 panic。

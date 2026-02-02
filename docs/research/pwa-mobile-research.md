# PWA 移动端增强 - 调研报告

> **调研日期**: 2025-02-02  
> **版本**: v1.0  
> **相关 Issue**: [#45](https://github.com/hrygo/divinesense/issues/45)

---

## 执行摘要

DivineSense 已具备 PWA 基础（manifest、Service Worker、离线页面），但配置不完整且仅在生产环境启用。本调研报告提出一套完整的 PWA 增强方案，使 DivineSense 可安装到移动端主屏幕，提供类原生 App 体验。

**工作量**: 1 人周  
**风险等级**: 低-中  
**预期收益**: 用户可一键安装 DivineSense 到移动设备主屏幕，离线时可浏览内容

---

## 1. 现状分析

### 1.1 已有 PWA 组件

| 组件 | 状态 | 位置 | 评估 |
|:-----|:-----|:-----|:-----|
| Manifest | ✅ 存在 | `web/public/site.webmanifest` | 🟡 不完整 |
| Service Worker | ✅ 存在 | `web/public/sw.js` | 🟡 仅生产环境，沿 memos 配置 |
| 离线页面 | ✅ 存在 | `web/public/offline.html` | ✅ 可用 |
| 图标资源 | ✅ 存在 | `web/public/*.png` | ✅ 192x192, 512x512, apple-touch-icon |
| 注册逻辑 | ✅ 存在 | `web/src/utils/serviceWorker.ts` | 🟡 仅生产环境 |
| 响应式设计 | ✅ 存在 | `web/src/components/MobileHeader.tsx` | ✅ 已适配 |

### 1.2 当前问题

1. **manifest 不完整** - 缺少 `theme_color`、`background_color`、`description`、`categories` 等字段
2. **开发环境无法测试** - Service Worker 仅在生产注册
3. **无安装提示** - 用户不知道可以安装
4. **缓存策略过时** - 沿用 memos 的缓存配置（缓存名、API 路由）

---

## 2. 技术方案

### 2.1 Manifest 优化

**当前配置** (`web/public/site.webmanifest`):
```json
{
  "name": "DivineSense",
  "short_name": "DivineSense",
  "icons": [
    { "src": "/android-chrome-192x192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/android-chrome-512x512.png", "sizes": "512x512", "type": "image/png" }
  ],
  "display": "standalone",
  "start_url": "/"
}
```

**优化后**:
```json
{
  "name": "DivineSense",
  "short_name": "DivineSense",
  "description": "AI 驱动的个人数字化第二大脑",
  "theme_color": "#3b82f6",
  "background_color": "#ffffff",
  "display": "standalone",
  "orientation": "portrait-primary",
  "start_url": "/",
  "scope": "/",
  "icons": [
    { "src": "/android-chrome-192x192.png", "sizes": "192x192", "type": "image/png", "purpose": "any maskable" },
    { "src": "/android-chrome-512x512.png", "sizes": "512x512", "type": "image/png", "purpose": "any maskable" },
    { "src": "/apple-touch-icon.png", "sizes": "180x180", "type": "image/png" }
  ],
  "categories": ["productivity", "notes", "education"],
  "screenshots": []
}
```

### 2.2 Service Worker 增强

**修改点**:

1. **更新缓存名称** (避免与 memos 混淆):
   - `CACHE_NAME`: `memos-v1` → `divinesense-v1`
   - `STATIC_CACHE`: `memos-static-v1` → `divinesense-static-v1`
   - `API_CACHE`: `memos-api-v1` → `divinesense-api-v1`

2. **API 路由调整**:
   - `/api` → 保留（DivineSense API）
   - 添加 `/memos.api.v1` 缓存支持

3. **开发环境支持**:
   ```typescript
   // web/src/utils/serviceWorker.ts
   // 移除生产环境限制
   if (import.meta.env.DEV) {
     return; // ❌ 删除这行
   }
   ```

### 2.3 安装提示 UI

**新建文件**: `web/src/hooks/usePWAInstall.ts`

```typescript
import { useState, useEffect } from "react";

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

export const usePWAInstall = () => {
  const [deferredPrompt, setDeferredPrompt] = useState<BeforeInstallPromptEvent | null>(null);
  const [isInstallable, setIsInstallable] = useState(false);
  const [isIOS, setIsIOS] = useState(false);

  useEffect(() => {
    // 检测 iOS
    const isIOSDevice = /iPad|iPhone|iPod/.test(navigator.userAgent) && !(window as any).MSStream;
    setIsIOS(isIOSDevice);

    // 监听 beforeinstallprompt 事件
    const handler = (e: Event) => {
      e.preventDefault();
      setDeferredPrompt(e as BeforeInstallPromptEvent);
      setIsInstallable(true);
    };

    window.addEventListener("beforeinstallprompt", handler);
    return () => window.removeEventListener("beforeinstallprompt", handler);
  }, []);

  const promptInstall = async () => {
    if (!deferredPrompt) return;
    deferredPrompt.prompt();
    const { outcome } = await deferredPrompt.userChoice;
    if (outcome === "accepted") {
      setIsInstallable(false);
    }
    setDeferredPrompt(null);
  };

  const dismissPrompt = () => {
    setDeferredPrompt(null);
    setIsInstallable(false);
  };

  return {
    isInstallable: isInstallable || (isIOS && !isInstallable),
    isIOS,
    promptInstall,
    dismissPrompt,
  };
};
```

**新建文件**: `web/src/components/PWAInstallPrompt.tsx`

```tsx
import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { usePWAInstall } from "@/hooks/usePWAInstall";

export const PWAInstallPrompt = () => {
  const { t } = useTranslation("pwa");
  const { isInstallable, isIOS, promptInstall, dismissPrompt } = usePWAInstall();

  if (!isInstallable) return null;

  if (isIOS) {
    return (
      <div className="fixed bottom-4 left-4 right-4 p-4 bg-blue-50 dark:bg-blue-950 rounded-lg shadow-lg flex items-start gap-3 z-50">
        <p className="text-sm text-foreground">
          {t("ios_instruction")}
        </p>
        <button
          onClick={dismissPrompt}
          className="shrink-0 text-foreground/50 hover:text-foreground"
        >
          <X className="w-4 h-4" />
        </button>
      </div>
    );
  }

  return (
    <div className="fixed bottom-4 left-4 right-4 p-4 bg-background rounded-lg shadow-lg border flex items-center justify-between gap-3 z-50">
      <div className="flex items-center gap-3">
        <span className="text-sm font-medium">{t("install_title")}</span>
        <span className="text-xs text-muted-foreground">{t("install_description")}</span>
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={dismissPrompt}
          className="text-sm text-muted-foreground hover:text-foreground px-3 py-1"
        >
          {t("cancel")}
        </button>
        <button
          onClick={promptInstall}
          className="text-sm bg-primary text-primary-foreground px-4 py-1.5 rounded-md"
        >
          {t("install")}
        </button>
      </div>
    </div>
  );
};
```

**i18n 翻译** (`web/src/locales/en.json`):
```json
{
  "pwa": {
    "install_title": "Install App",
    "install_description": "Add to home screen for quick access",
    "install": "Install",
    "cancel": "Not now",
    "ios_instruction": "To install: tap Share → Add to Home Screen"
  }
}
```

---

## 3. 验收标准

| 标准 | 验证方法 |
|:-----|:---------|
| `pnpm lint` 通过 | `cd web && pnpm lint` |
| Lighthouse PWA ≥ 90 | Chrome DevTools → Lighthouse → PWA |
| iOS Safari 可安装 | iOS Safari → 分享按钮 → 添加到主屏幕 |
| Android Chrome 可安装 | Chrome → 地址栏图标 → 安装 |
| 开发环境 SW 运行 | Chrome DevTools → Application → Service Workers |
| `make check-i18n` 通过 | `make check-i18n` |

---

## 4. 参考资源

| 资源 | 链接 |
|:-----|:-----|
| PWA 安装标准 | https://web.dev/learn/pwa/ |
| Web App Manifest | https://developer.mozilla.org/en-US/docs/Web/Manifest |
| Service Worker API | https://developer.mozilla.org/en-US/docs/Web/API/Service_Worker_API |
| Lighthouse PWA | https://developer.chrome.com/docs/lighthouse/pwa |

---

*调研完成时间: 2025-02-02*

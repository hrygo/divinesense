# 语音交互 - 调研报告

> **调研日期**: 2025-02-02
> **版本**: v1.0
> **相关 Issue**: [#46](https://github.com/hrygo/divinesense/issues/46)

---

## 执行摘要

DivineSense 当前仅支持文本输入。本调研报告提出一套基于 Web Speech API 的三阶段渐进式语音交互方案，使 DivineSense 支持语音输入、语音命令和语音对话。

**工作量**: 4-6 周（分阶段交付）
**风险等级**: 中（浏览器兼容性需充分测试）
**预期收益**: 移动端用户可语音输入，双手被占用时也能使用 AI 代理

---

## 1. 现状分析

### 1.1 当前输入方式

| 组件 | 位置 | 功能 |
|:-----|:-----|:-----|
| `ChatInput.tsx` | `web/src/components/AIChat/` | 文本输入框 + 发送按钮 |
| `Textarea` | UI 组件 | 支持 Ctrl/Cmd+Enter 快捷键发送 |

### 1.2 现有问题

1. **移动端打字效率低** — 手机输入体验不佳
2. **双手被占用无法操作** — 开车、做饭时无法使用
3. **无语音交互入口** — 未利用浏览器原生语音能力

---

## 2. 技术方案

### 2.1 Web Speech API 兼容性

#### 浏览器支持矩阵

| 平台 | STT (语音识别) | TTS (语音合成) | PWA STT | 备注 |
|:-----|:---------------|:---------------|:---------|:-----|
| Android Chrome | ✅ 完整 | ✅ 完整 | ✅ | 最佳体验 |
| Desktop Chrome | ✅ 完整 | ✅ 完整 | ✅ | Blink 引擎 |
| iOS Safari (浏览器) | ⚠️ 部分支持 | ✅ | — | WebKit, 需 workaround |
| iOS Chrome | ❌ **不支持** | ✅ | ❌ | 强制使用 WebKit |
| iOS PWA | ❌ **不支持** | ✅ | ❌ | WebView 限制 |
| Safari Desktop | ❌ 不支持 | ✅ | — | 无 STT API |

#### 关键发现

1. **iOS Chrome = Safari WebKit**
   - Apple 要求所有 iOS 浏览器使用 WebKit 引擎
   - iOS Chrome 不支持 STT（与 Safari 不同）

2. **iOS PWA 限制**
   - Safari Mobile PWA 模式下 STT 不可用
   - WebView 直接触发错误

3. **iOS Safari 事件异常**
   - 事件顺序与 Chrome 不同
   - 需要 750ms workaround

### 2.2 三阶段实施方案

#### Phase 1: 语音输入 (4-5 人天)

**目标**: 按住麦克风说话，文本自动填充到输入框

**文件清单**:

| 文件 | 操作 | 说明 |
|:-----|:-----|:-----|
| `web/src/hooks/useSpeechRecognition.ts` | 新建 | STT Hook 封装 |
| `web/src/hooks/useSpeechCapability.ts` | 新建 | 平台能力检测 |
| `web/src/components/VoiceInputButton.tsx` | 新建 | 麦克风按钮 |
| `web/src/components/AIChat/ChatInput.tsx` | 修改 | 集成语音按钮 |
| `web/src/locales/en.json` | 修改 | 添加翻译 |
| `web/src/locales/zh-Hans.json` | 修改 | 添加翻译 |

**核心代码**:

```typescript
// web/src/hooks/useSpeechRecognition.ts
interface SpeechRecognitionState {
  isSupported: boolean;
  isListening: boolean;
  transcript: string;
  interimTranscript: string;
  error: string | null;
}

export const useSpeechRecognition = (options?: {
  lang?: string;
  continuous?: boolean;
  interimResults?: boolean;
}) => {
  const [state, setState] = useState<SpeechRecognitionState>({
    isSupported: false,
    isListening: false,
    transcript: "",
    interimTranscript: "",
    error: null,
  });

  // 浏览器支持检测
  useEffect(() => {
    const SpeechRecognition = (window as any).SpeechRecognition
      || (window as any).webkitSpeechRecognition;
    setState(prev => ({ ...prev, isSupported: !!SpeechRecognition }));
  }, []);

  // iOS Safari 750ms workaround
  const start = useCallback(() => {
    const SpeechRecognition = (window as any).webkitSpeechRecognition;
    if (!SpeechRecognition) return;

    const recognition = new SpeechRecognition();
    recognition.interimResults = true;
    recognition.lang = options?.lang || 'zh-CN';

    let timeoutId: ReturnType<typeof setTimeout>;
    let audioEnded = false;

    recognition.onresult = (event: SpeechRecognitionEvent) => {
      clearTimeout(timeoutId);

      if (audioEnded) {
        handleResult(event);
      } else {
        timeoutId = setTimeout(() => handleResult(event), 750);
      }
    };

    recognition.onaudioend = () => {
      audioEnded = true;
    };

    recognition.start();
  }, []);

  return { state, start, stop, reset };
};
```

#### Phase 1.5: TTS 语音播报 (2-3 人天)

**目标**: AI 回复自动语音播报

**文件清单**:

| 文件 | 操作 | 说明 |
|:-----|:-----|:-----|
| `web/src/hooks/useSpeechSynthesis.ts` | 新建 | TTS Hook 封装 |

**核心代码**:

```typescript
// web/src/hooks/useSpeechSynthesis.ts
export const useSpeechSynthesis = () => {
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [voices, setVoices] = useState<SpeechSynthesisVoice[]>([]);

  useEffect(() => {
    const synth = window.speechSynthesis;

    const loadVoices = () => {
      setVoices(synth.getVoices());
    };

    loadVoices();
    synth.onvoiceschanged = loadVoices;
  }, []);

  const speak = useCallback((text: string, lang: string = 'zh-CN') => {
    const synth = window.speechSynthesis;
    const utterance = new SpeechSynthesisUtterance(text);

    const voice = voices.find(v => v.lang.startsWith(lang));
    if (voice) utterance.voice = voice;

    utterance.onstart = () => setIsSpeaking(true);
    utterance.onend = () => setIsSpeaking(false);
    utterance.onerror = () => setIsSpeaking(false);

    synth.speak(utterance);
  }, [voices]);

  return { isSupported: !!window.speechSynthesis, isSpeaking, speak, cancel, voices };
};
```

#### Phase 2: 语音命令 (5-7 人天)

**目标**: 说出关键词触发特定操作

**文件清单**:

| 文件 | 操作 | 说明 |
|:-----|:-----|:-----|
| `web/src/hooks/useVoiceCommand.ts` | 新建 | 命令识别 Hook |
| `web/src/utils/voiceCommands.ts` | 新建 | 命令映射表 |

**命令列表**:

```typescript
export const voiceCommands: VoiceCommand[] = [
  { keywords: ["新建笔记", "创建笔记", "写笔记"], action: () => navigate("/") },
  { keywords: ["搜索", "查找"], action: () => focusSearch() },
  { keywords: ["今天日程", "日程"], action: () => navigate("/schedule") },
  { keywords: ["清除", "清空"], action: () => clearInput() },
  { keywords: ["发送", "提交"], action: () => sendMessage() },
];
```

#### Phase 3: 语音对话 (8-12 人天)

**目标**: 完整 STT → AI → TTS 循环

**文件清单**:

| 文件 | 操作 | 说明 |
|:-----|:-----|:-----|
| `web/src/components/AIChat/VoiceModePanel.tsx` | 新建 | 语音模式面板 |

---

## 3. 降级策略

### 3.1 平台检测

```typescript
export interface SpeechCapability {
  stt: 'full' | 'partial' | 'none';
  tts: boolean;
  isPWA: boolean;
  platform: 'ios' | 'android' | 'desktop';
  guidanceMessage?: string;
  guidanceAction?: string;
}

export const detectCapability = (): SpeechCapability => {
  const ua = navigator.userAgent;
  const isIOS = /iPad|iPhone|iPod/.test(ua);
  const isAndroid = /Android/.test(ua);
  const isChrome = /Chrome/.test(ua) && !/Edge/.test(ua);
  const isSafari = /Safari/.test(ua) && !/Chrome/.test(ua);
  const isPWA = (window as any).matchMedia?.('(display-mode: standalone)').matches;

  const SpeechRecognition = (window as any).SpeechRecognition
    || (window as any).webkitSpeechRecognition;

  // iOS Chrome: 不支持 STT
  if (isIOS && isChrome) {
    return {
      stt: 'none',
      tts: true,
      isPWA: isPWA,
      platform: 'ios',
      guidanceMessage: "iOS Chrome 暂不支持语音输入",
      guidanceAction: "请使用 Safari 浏览器",
    };
  }

  // iOS PWA: 不支持 STT
  if (isIOS && isPWA) {
    return {
      stt: 'none',
      tts: true,
      isPWA: true,
      platform: 'ios',
      guidanceMessage: "PWA 模式暂不支持语音输入",
      guidanceAction: "请在 Safari 浏览器中使用",
    };
  }

  // iOS Safari 浏览器: 部分支持
  if (isIOS && isSafari && !isPWA && SpeechRecognition) {
    return {
      stt: 'partial',
      tts: true,
      isPWA: false,
      platform: 'ios',
    };
  }

  // Android/Chrome: 完整支持
  if (SpeechRecognition) {
    return {
      stt: 'full',
      tts: true,
      isPWA: isPWA,
      platform: isAndroid ? 'android' : 'desktop',
    };
  }

  // 其他: 不支持
  return {
    stt: 'none',
    tts: 'speechSynthesis' in window,
    isPWA: isPWA,
    platform: 'desktop',
  };
};
```

### 3.2 UI 降级

| 环境 | STT | TTS | UI 表现 |
|:-----|:-----|:-----|:---------|
| iOS Chrome | ❌ | ✅ | 提示"请使用 Safari" |
| iOS PWA | ❌ | ✅ | 引导"在浏览器中使用" |
| iOS Safari (浏览器) | ⚠️ | ✅ | 正常 |
| Android Chrome | ✅ | ✅ | 完整功能 |
| Safari Desktop | ❌ | ✅ | 隐藏按钮 |
| Desktop Chrome | ✅ | ✅ | 完整功能 |

---

## 4. 验收标准

| 标准 | 验证方法 |
|:-----|:---------|
| `pnpm lint` 通过 | `cd web && pnpm lint` |
| Chrome/Android 可用 | 浏览器测试 |
| iOS Safari (浏览器) 可用 | iOS 设备测试 |
| iOS Chrome 显示提示 | iOS 设备测试 |
| TTS 播报可用 | 所有平台测试 |
| `make check-i18n` 通过 | `make check-i18n` |

---

## 5. 参考资源

| 资源 | 链接 |
|:-----|:-----|
| Web Speech API | https://developer.mozilla.org/en-US/docs/Web/API/Web_Speech_API |
| Taming the Web Speech API | https://webreflection.medium.com/taming-the-web-speech-api-ef64f5a245e1 |
| HTML5语音合成Speech Synthesis API简介 | https://www.zhangxinxu.com/wordpress/2017/01/html5-speech-recognition-synthesis-api/ |
| Can I use - Speech Recognition | https://caniuse.com/speech-recognition |
| Stack Overflow - iOS Speech Recognition | https://stackoverflow.com/questions/28592170/add-ios-speech-recognition-support-for-web-app |

---

*调研完成时间: 2025-02-02*

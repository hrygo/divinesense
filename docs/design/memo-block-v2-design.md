# MemoBlockV2 Design Specification

## Design Philosophy: "Fluid Card"

AI-Native applications require a fundamentally different approach to UI design:

1. **Content-First** - The user's content is the hero, UI elements should recede
2. **Gesture-Driven** - Mobile interactions should feel natural, not click-heavy
3. **Progressive Disclosure** - Show what's needed, when it's needed
4. **Subtle Intelligence** - AI capabilities should be hinted, not shouted

---

## Visual System

### Color Palette

| Usage | Light Mode | Dark Mode | Purpose |
|:-----|:-----------|:----------|:--------|
| **Card BG** | `rgba(255,255,255,0.8)` | `rgba(24,24,27,0.8)` | Glassmorphism base |
| **Border** | `zinc-200/60` | `zinc-800/60` | Subtle separation |
| **Primary** | `zinc-900` | `zinc-100` | Main content |
| **Secondary** | `zinc-500` | `zinc-400` | Metadata |
| **Accent** | `violet-600` | `violet-400` | AI features |
| **Pinned** | `amber-500` | `amber-400` | Pinned state |
| **Archived** | `zinc-400` | `zinc-500` | Archived state |

### Typography

| Element | Font | Size | Weight | Line-height |
|:-------|:-----|:-----|:-------|:-----------|
| **Preview** | System (optimized) | 14px | 400 | 1.5 |
| **Metadata** | System (optimized) | 12px | 400 | - |
| **Buttons** | System (optimized) | 12px | 500 | - |

### Spacing Scale

| Token | Value | Usage |
|:-----|:------|:-----|
| `xs` | 4px | Icon padding |
| `sm` | 8px | Gap between small elements |
| `md` | 16px | Card padding |
| `lg` | 24px | Gap between cards |
| `xl` | 32px | Section spacing |

---

## Component Structure

```
MemoBlockV2
├── StatusIndicator (left border, 4px)
├── SwipeHint (overlay during swipe)
├── MemoCompactHeader
│   ├── Icon (Bookmark/Pin)
│   ├── PreviewText
│   └── MetadataRow
│       ├── Time
│       ├── VisibilityBadge
│       └── CommentIndicator (if applicable)
├── ExpandableContent (conditional)
│   └── MemoView (full content + reactions)
└── MemoCompactFooter
    ├── ToggleButton
    └── ActionButtons
        ├── Edit
        ├── Pin/Unpin
        ├── Copy
        ├── Share (conditional)
        └── MoreMenu (dropdown)
```

---

## Interaction Patterns

### Desktop

| Action | Trigger | Feedback |
|:-------|:--------|:---------|
| **Expand/Collapse** | Click card / Chevron | Spring animation |
| **Quick Actions** | Hover + Click button | Button highlight |
| **Context Menu** | Click "More" button | Dropdown appears |

### Mobile

| Action | Trigger | Feedback |
|:-------|:--------|:---------|
| **Expand/Collapse** | Tap card | Spring animation |
| **Archive** | Swipe left | Yellow bg + text hint |
| **Delete** | Swipe right | Red bg + text hint |
| **Quick Actions** | Long-press | Haptic + Menu |
| **Copy** | Swipe + hold | Toast confirmation |

---

## Animation Specifications

### Spring Animation

```css
transition: all 300ms cubic-bezier(0.34, 1.56, 0.64, 1);
```

This creates a subtle "bounce" effect that feels alive and responsive.

### Staggered Reveal

Items appear sequentially with decreasing delays:
- Items 1-5: 50ms increments (0, 50, 100, 150, 200ms)
- Items 6+: 30ms increments (230, 260, 290ms...)

### Swipe Feedback

- Threshold: 30px horizontal movement
- Cancel zone: 50px vertical movement
- Visual: Background color change with overlay

---

## Responsive Behavior

### Breakpoints

| Size | Width | Layout |
|:-----|:------|:-------|
| **Mobile** | < 640px | Single column, full-width cards |
| **Tablet** | 640-1024px | Single column, centered cards |
| **Desktop** | > 1024px | 2-column grid |

### Mobile Optimizations

- Full-width cards (negative margins)
- Larger touch targets (44px min)
- Swipe gestures primary
- Simplified footer (3 buttons + more)
- Status bar time hidden (redundant)

---

## Accessibility

- Keyboard navigation: Tab through all interactive elements
- Focus indicators: 2px violet ring
- Screen reader: Semantic HTML, ARIA labels
- Reduced motion: Respects `prefers-reduced-motion`
- Color contrast: WCAG AA compliant

---

## Migration Guide

### From MemoBlock to MemoBlockV2

1. Import the new component:
```tsx
import { MemoBlockV2 } from "@/components/Memo/MemoBlockV2";
```

2. Replace in your list:
```tsx
// Before
<MemoBlock memo={memo} onEdit={onEdit} />

// After
<MemoBlockV2 memo={memo} onEdit={onEdit} />
```

3. For grid layout:
```tsx
import { MemoGrid } from "@/components/Memo/MemoGrid";

<MemoGrid memos={memos} onEdit={onEdit} />
```

---

## Future Enhancements

- [ ] AI chip indicator for AI-generated summaries
- [ ] Drag-and-drop reordering
- [ ] Multi-select with batch actions
- [ ] Voice memo integration
- [ ] Real-time collaboration indicators

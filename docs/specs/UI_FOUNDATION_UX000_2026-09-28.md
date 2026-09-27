# SPEC — UI foundation (`CCMAI-UX-000`)

**Date:** 2026-09-28 · **Author:** Claude (`SPEC_AUTHOR`, under the owner's UI/UX design delegation) · **Risk:** R2 (the theme is global and the new components display reviewed source-status, confidence and sync semantics) · **Work order:** [`CCMAI_UX_000`](../work_orders/CCMAI_UX_000.md)

**Design authority:** [UI design direction](../decisions/UI_DESIGN_DIRECTION_2026-09-27.md) (tokens, font, dark mode, components) and [redesign roadmap](../roadmaps/UI_UX_REDESIGN_ROADMAP_2026-09-27.md), Phase 0. Approved canvas: https://claude.ai/artifact/DsSy7rS8zkAr4Gk6DB6vxt (private to the owner; approved by Claude under delegation on 2026-09-28 in the state recorded by the handoff's Phase 0 canvas entry; the design decision document is the textual source of truth for every token). **Baseline:** [UI/UX review](../reviews/UI_UX_REVIEW_BASELINE_2026-09-27.md) at `3a41252`.

## 1. Intent

Give every later screen tranche (`CCMAI-UX-010+`) one shared, reviewed foundation: design tokens in the Vuetify theme (light and dark), a self-hosted Vietnamese font, shared display components that encode the reviewed R004–R007 semantics exactly once, locale-aware formatting helpers, a UI glossary, and a repeatable screenshot tool.

This tranche does **not** redesign any screen. Existing views keep their markup and logic. They change appearance only through the global theme and font.

## 2. Requirements

### 2.1 Theme tokens (`frontend/src/plugins/vuetify.ts`)

- T1. Replace the inherited colors and the "SePay" comment with the token table in the design decision §2.1, for both `light` and `dark`.
- T2. Map Vuetify semantic colors so existing views stay legible: `primary`, `background`, `surface`, `on-surface`/`on-background` (text), `success` (pass), `error`, `warning`, `info`, `secondary`.
- T3. Add named custom colors for the status palette: `pass`, `pass-bg`, `fail`, `fail-bg`, `skip`, `skip-bg`, `src-changed`, `src-changed-bg`, `src-unavailable`, `src-unavailable-bg`, `src-legacy`, `danger`, `border`, `text-muted`. Components read them as `rgb(var(--v-theme-<name>))`.
- T4. Every status text/background pair keeps the contrast recorded in the decision (≥ 4.5:1). A unit test recomputes WCAG contrast from the theme object for every pair in both themes.
- T5. The chosen theme persists per browser (`localStorage`, key `ccma_theme`). Storage failure falls back to `light` without error.
- T6. Defaults: radius 8 for buttons and chips, 12 for cards, flat cards with a border instead of elevation, minimum 44px interactive height on mobile for the new components.

### 2.2 Font

- F1. Add the dependency `@fontsource/be-vietnam-pro` (SIL OFL 1.1). Import only the `vietnamese` and `latin` subsets for weights 400/500/600/700. Files are bundled by Vite into `dist`; no request goes to Google Fonts or any CDN.
- F2. Set `--v-font-body` and `--v-font-heading` to `"Be Vietnam Pro"` with a system fallback stack. Numbers in the new components use `font-variant-numeric: tabular-nums`.

### 2.3 Shared components (`frontend/src/components/ui/`)

Each component is presentational, takes data by props, calls no API and reads no store.

| Component | Contract |
|---|---|
| `VerdictChip` | `verdict`: `pass` / `fail` / `skip` / `classified`. Filled style (status background + status text) with icon and text. Helper `verdictFromSeverity()` maps API `PASS`→pass, `SKIP`→skip, anything else→fail, matching current `Results.vue`. |
| `SourceStatusChip` | `status`: one of the four R004 values. A missing or unknown value renders as `verification_unavailable`, never hidden. Outlined style with icon and the existing R004 label keys. Style never reads as reassurance: no green, no check icon. `changed_since_analysis` uses the warning-triangle icon. |
| `SourceStatusPanel` | Takes `statuses: string[]` (one per result). Always renders the local-only note (`results_source_note`), with no prop that hides it. Lists the count per present status in `SOURCE_INTEGRITY_ORDER` (most concerning first); unknown values count as unavailable. Shows nothing reassuring when the list is empty (note only). |
| `SyncStatusChip` | `status`: `''`/`never` → "Chưa đồng bộ"; `syncing` → "Đang đồng bộ" (started, not complete); `success`; `partial`; `error`; anything else → "Không rõ" (never success). Error and partial are visually stronger than success. |
| `MetricCard` | `label`, `value` (`number | null`; null renders "—", never 0), optional `to` (router location for drill-down; card becomes a real link), optional `hint`, `tone`. Tabular numbers. |
| `AppDialog` | Wraps `v-dialog`: title, always a close button in the header corner (accessible label), closes on Esc, optional `aiGenerated` label, default and `actions` slots, full screen at ≤ 600px. |
| `ConfirmDialog` | Destructive confirmation: title, message, confirm label, `danger` confirm button, cancel; emits `confirm`/`cancel`; `loading` disables both buttons. |
| `ActionMenu` | "⋯" icon button (44px target, accessible label "Thao tác khác") opening a list of `items` `{ key, label, icon?, danger? }`; emits `select(key)`. Danger items are listed last after a divider. It never runs a destructive action itself. |
| `AiGeneratedLabel` | Small label "Nhận xét do AI tạo" with icon. |
| `ConfidenceText` | `confidence: number | null`, `basis`. Null or basis `unavailable` → "Không có" (never 0% or 100%). Otherwise the rounded percentage plus the R005 qualifier "mô hình tự ước lượng, chưa hiệu chuẩn". |
| `FilterBar` | Horizontal chip row that scrolls sideways on narrow screens instead of wrapping or clipping; slot-based. |
| `ResultCard` | Mobile result card: customer name, time, verdict slot/prop, one-line summary (ellipsis), source status chip, optional score; emits `open`. |

### 2.4 Logic helpers

- `frontend/src/utils/format.ts`: `formatDate` (dd/mm/yyyy for `vi`, locale default for `en`), `formatDateTime` (24-hour), `formatRelative` (never negative: a future instant within 1 minute or later reads "vừa xong"/"just now"; < 60 min in minutes; < 24 h in hours; otherwise days), `formatNumber`, `formatCurrency` (VND without decimals, USD with two). Invalid input returns "—".
- `frontend/src/utils/review.ts`: `needsReview(result)` is true when the verdict is fail, or source status is `changed_since_analysis` or `verification_unavailable` (unknown counts as unavailable). This feeds the future "Cần xem lại" filter; no view uses it in this tranche.

### 2.5 Glossary and preview

- `docs/reference/UI_GLOSSARY.md`: one term per concept in `vi` and `en` (Tác vụ AI, vấn đề/nhãn, hội thoại/tin nhắn, đánh giá, verdicts, the four source statuses, sync statuses, confidence wording).
- A static route `/wireframes/design-system` (same precedent as the existing `/wireframes/*` routes: no API call, no auth) renders every component in every state, so screenshots can prove light/dark and desktop/mobile rendering.

### 2.6 Screenshot tool

- `scripts/ui-screenshots.mjs`: dependency-free Node 24 script driving local Chrome/Edge headless over the DevTools Protocol. Input: base URL, route list, optional access token, output folder. It captures each route at desktop 1440×900 and mobile 390×844 in light and dark, and writes a JSON report with every uncaught exception and console error. Exit code is non-zero on any JS error.
- `scripts/ui-screenshots.ps1`: `-Mode preview` builds the frontend and serves `dist` locally for the static routes. `-Mode app` starts a **disposable** Compose project (`scripts/ui-screenshots/compose.yml`, unique project name, generated throwaway secrets written outside the repo, own volumes), creates a throwaway admin through `/api/v1/setup`, imports the synthetic demo data through the existing demo endpoint, optionally adds one synthetic snapshot so a result shows `changed_since_analysis`, captures the app routes, then removes the project and its volumes. It never reads the repo `.env`, never touches Compose project `ccma` and never calls a provider.
- `docs/reference/UI_SCREENSHOTS.md`: how to run both modes and where output goes.

## 3. Invariants (review checklist)

1. No API contract, backend, schema or store change. `stores/jobs.ts` constants are reused, not edited.
2. The four source statuses stay distinct, keep their reviewed labels, and none renders as reassurance; the local-only note cannot be hidden in `SourceStatusPanel`.
3. Empty confidence renders "Không có"; sync `syncing` never reads as complete.
4. No external font/CDN request at runtime.
5. i18n `vi`/`en` key parity holds (existing test).
6. Existing views compile and render with 0 JS errors under the new theme in light and dark.

## 4. Acceptance evidence

`vue-tsc` + `vite build`; vitest for format, review, theme contrast and component semantics (unknown source status, null confidence, syncing wording, panel note always present); screenshots of `/wireframes/design-system` and the main app routes at desktop/mobile × light/dark from the new tool with 0 JS errors; a network check that the built app loads no external font; contrast and 44px target check recorded in the BUILD evidence.

## 5. Out of scope

Screen redesigns, wiring the new components into existing views, the "Cần xem lại" filter UI, UX-001 data fixes, backend changes, provider calls, deployment.

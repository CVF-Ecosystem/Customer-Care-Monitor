# UI glossary — one term per concept

**Tranche:** `CCMAI-UX-000` · **Date:** 2026-09-28 · **Source:** [UI design direction](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/blob/main/docs/decisions/UI_DESIGN_DIRECTION_2026-09-27.md), existing i18n keys in `frontend/src/i18n/vi.ts` / `en.ts`.

Screens redesigned from `CCMAI-UX-010` onward use exactly these words. When a screen still shows an older word, its screen tranche replaces it. Terms marked **fixed** carry reviewed semantics (R004–R007) and must not be reworded into something more reassuring.

## Objects

| Concept | Tiếng Việt | English | Do not use |
|---|---|---|---|
| A configured AI analysis (QC or classification) | Tác vụ AI | AI job | "Công việc", "job" in Vietnamese UI |
| One execution of a job | Lần chạy | Run | "Phiên" |
| A customer conversation (thread) | Hội thoại | Conversation | "Cuộc chat" in headings |
| A single message inside a conversation | Tin nhắn | Message | Using "tin nhắn" to count conversations |
| A connected chat source (Zalo OA, Facebook, Pancake) | Kênh | Channel | "Nguồn" (reserved for source status) |
| A QC rule violation found in a conversation | Vấn đề | Issue | "Lỗi" (reserved for system errors), "nhãn" |
| A classification label | Nhãn | Tag | "Vấn đề" |
| The act of AI assessing a conversation | Đánh giá | Evaluate / review | "Chấm" |
| Score 0–100 | Điểm | Score | "Điểm số AI" |

## Verdicts

| Value | Tiếng Việt | English | Style |
|---|---|---|---|
| `PASS` | Đạt | Passed | Filled chip, green, check-circle outline |
| anything not PASS/SKIP | Không đạt | Failed | Filled chip, red, close-circle outline |
| `SKIP` | Bỏ qua | Skipped | Filled chip, grey, minus-circle outline |
| classification evaluation | Đã phân loại | Classified | Filled chip, primary tint, tag outline |
| failed, source changed, or cannot verify | Cần xem lại | Needs review | Filter name, not a verdict |

## Source status (fixed, R004)

| Value | Tiếng Việt | English |
|---|---|---|
| `changed_since_analysis` | Nguồn đã đổi kể từ khi đánh giá | Source changed since analysis |
| `verification_unavailable` (also any missing/unknown value) | Không xác minh được | Could not verify |
| `legacy_unverified` | Chưa xác minh (kết quả cũ) | Unverified (legacy result) |
| `bound_currentness_unverified` | Chưa xác minh đầy đủ | Not fully verified |
| Always-visible note | So sánh cục bộ với dữ liệu hiện tại, không đảm bảo đã kiểm tra hết lịch sử chỉnh sửa hoặc xóa ở nguồn. | This is a local comparison against current data only — it does not guarantee the full edit or deletion history at the source was checked. |

No source status is ever called "đã xác minh", "còn hiện hành", "an toàn" or shown in green.

## Sync status (fixed, R001/R007)

| Value | Tiếng Việt | English |
|---|---|---|
| empty / never | Chưa đồng bộ | Not synced yet |
| `syncing` | Đang đồng bộ (đã bắt đầu, chưa hoàn tất) | Syncing (started, not finished) |
| `success` | Đã đồng bộ | Synced |
| `partial` | Đồng bộ một phần | Partially synced |
| `error` | Đồng bộ lỗi | Sync failed |
| anything else | Không rõ trạng thái | Unknown status |

The "Đồng bộ ngay" button reports "đã bắt đầu đồng bộ", never "đồng bộ xong".

## AI output (fixed, R005)

| Concept | Tiếng Việt | English |
|---|---|---|
| Label on AI-written reviews and verdicts | Nhận xét do AI tạo | AI-generated review |
| Confidence with a value | Độ tin cậy: 72% · mô hình tự ước lượng, chưa hiệu chuẩn | Confidence: 72% · model self-estimate, not calibrated |
| Confidence without a value | Độ tin cậy: Không có | Confidence: Not available |

## Formatting

| Item | Tiếng Việt | English |
|---|---|---|
| Date | 05/09/2026 | Sep 5, 2026 |
| Date and time | 05/09/2026 19:30 (24 giờ) | Sep 5, 2026 19:30 |
| Relative time | vừa xong · 5 phút trước · 2 giờ trước · 3 ngày trước (never negative) | just now · 5 min ago · 2 h ago · 3 days ago |
| Number | 1.234.567 | 1,234,567 |
| Money | 125.000 ₫ · 1,50 US$ | $1.50 |
| Unknown value | — | — |

Helpers: `frontend/src/utils/format.ts`, `frontend/src/utils/review.ts`.

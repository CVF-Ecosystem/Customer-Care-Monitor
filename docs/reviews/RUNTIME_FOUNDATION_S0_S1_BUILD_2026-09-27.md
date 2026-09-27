# BUILD evidence: runtime foundation S0 và sync truth S1

**Work order:** `CCMAI-RUNTIME-001` · **Ngày:** 2026-09-27 · **Kết quả:** BUILD_PASS / REVIEW_PENDING.

## Delivered

- Khóa corpus synthetic tiếng Việt `s0-vi-intervention-v1` gồm 12 ca bắt buộc của S0; không chứa dữ liệu khách hàng.
- Ghi baseline thật từ volume phát triển: 16 bảng, chưa có tenant/channel/conversation/message/job/run/result/AI usage.
- Chọn Pancake cho pilot “khiếu nại/khách chờ phản hồi”, trong khi hợp đồng sync vẫn dùng chung cho mọi adapter.
- Một lỗi conversation, message, attachment coverage hoặc message-count update làm lượt kênh thành `partial`.
- `partial` và `error` không cập nhật `last_sync_at`; chỉ `success` cập nhật checkpoint và kích after-sync jobs.
- `SyncAllChannels` trả lỗi tổng hợp thay vì báo thành công khi có channel lỗi.
- Update conversation/message hiện hữu và JSON serialization không còn bỏ qua lỗi database/encoding trong đường đã sửa.
- Scheduler không ghi activity success/error trùng với `SyncChannel`; activity có `sync.partial` riêng.
- UI danh sách, chi tiết, lịch sử và thông báo sau sync phân biệt `partial` bằng cảnh báo.

## Validation

- Containerized Go 1.26 `go test ./engine -count=1`: PASS.
- Containerized Go 1.26 `go build ./...`: PASS.
- Frontend `npm run build`: PASS.
- Compose application image rebuild: PASS.
- Retained `CCMA` app/database restart: PASS; MySQL healthy, app connected, migration completed, scheduler loaded zero jobs.
- Startup log confirms pricing sync disabled and contains no provider/AI call.
- S0 database counts remained zero; no channel credential or real sync was used.

## Claim boundary and open work

This tranche proves a bounded data-reliability behavior and synthetic baseline. It does not prove full S1, provider admission, cost savings, classification quality or CVF control of AI runtime. Snapshot digest/evidence spans, typed decisions, local gate, provider budget/admission and human disposition remain open under S1/S2/S3/S5.

Independent R2 review should inspect checkpoint semantics, partial error observability, attachment-coverage behavior, scheduler activity ownership and UI wording before FREEZE.

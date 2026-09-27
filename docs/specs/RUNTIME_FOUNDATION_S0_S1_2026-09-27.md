# Runtime foundation S0/S1: baseline và đồng bộ không báo thành công giả

**Trạng thái:** BUILD_COMPLETE / REVIEW_PENDING · **Ngày:** 2026-09-27 · **Tranche:** `CCMAI-RUNTIME-001` · **Rủi ro:** R2.

## Mục tiêu và phạm vi

Tranche đầu của roadmap hoàn tất S0 bằng baseline/corpus synthetic được phép dùng và triển khai lát cắt S1 đầu tiên: một lỗi ở bất kỳ hội thoại hoặc tin nhắn nào không được biến thành trạng thái đồng bộ thành công của cả kênh.

Pilot chọn adapter Pancake và use case “hội thoại có dấu hiệu khiếu nại hoặc khách chờ phản hồi cần người phụ trách can thiệp”. Logic trạng thái đồng bộ vẫn dùng chung cho mọi `ChannelAdapter`; chọn Pancake không biến CRM thành product dependency.

Không gọi AI/LLM, không gửi dữ liệu khách hàng, không triển khai machine gate/provider admission và không claim CVF đang kiểm soát runtime AI trong tranche này.

## Baseline S0

- Database phát triển mới có 16 bảng và chưa có tenant, channel, conversation, message, job, run, result hoặc AI usage log.
- Corpus `backend/engine/testdata/s0_vietnamese_intervention_corpus.json` là synthetic-only, version `s0-vi-intervention-v1`.
- Corpus phải có chat rỗng, thiếu lịch sử, trùng, khiếu nại, phủ định, mỉa mai, không dấu, xen ngôn ngữ, PII, prompt injection, claim tài chính chưa có SoT xác nhận và nguồn bị sửa/stale.
- Baseline chưa có call/token/cost/latency hoặc thời gian human review để so sánh; mọi giá trị bằng 0 hiện tại có nghĩa “chưa có lượt chạy”, không phải hiệu quả đã đo.

## Hợp đồng trạng thái đồng bộ

| Trạng thái | Điều kiện | Checkpoint `last_sync_at` | After-sync job |
|---|---|---|---|
| `success` | Toàn bộ conversation/message được xử lý không lỗi | Cập nhật sau khi hoàn tất | Được kích hoạt |
| `partial` | Fetch danh sách thành công nhưng ít nhất một conversation/message/upsert/count thất bại | Không thay đổi | Không kích hoạt |
| `error` | Không thể bắt đầu/hoàn thành bước cấp kênh như decrypt, adapter init hoặc fetch danh sách | Không thay đổi | Không kích hoạt |
| `syncing` | Lượt đồng bộ đang chạy | Không thay đổi | Không kích hoạt |

`partial` phải lưu thông báo tổng hợp có scope và external ID đủ để vận hành, ghi activity `sync.partial`, trả lỗi cho caller và hiển thị cảnh báo trên UI. Lỗi DDL/DB khi ghi trạng thái không được bỏ qua. Chỉ `success` được ghi `sync.completed`.

## Quy tắc ghi dữ liệu

- Update conversation/message hiện hữu phải kiểm tra lỗi từ database.
- Lỗi marshal metadata/attachment/raw payload phải được trả về, không biến thành JSON rỗng âm thầm.
- Message lỗi không được tăng bộ đếm thành công.
- Danh sách lỗi hiển thị phải có giới hạn kích thước nhưng giữ tổng số lỗi và số lỗi bị lược bớt.
- Retry dùng checkpoint thành công gần nhất với buffer hiện có, nên có thể replay; unique index và upsert phải giữ idempotency.

## Acceptance

1. Unit test chứng minh `success` chỉ khi không có failure; có một failure tạo `partial`.
2. Unit test chứng minh `partial`/`error` không tạo trường update `last_sync_at`; `success` có tạo.
3. Frontend phân biệt `partial` bằng cảnh báo, không hiển thị như chưa từng đồng bộ hoặc success.
4. Backend/frontend/docs build đạt; Docker app restart được trên volume `CCMA` hiện tại.
5. Không có provider call. Evidence chỉ claim data reliability foundation, không claim AI gate.

## Deferred

Snapshot digest/evidence span, typed eligibility/execution/disposition, provider admission, budget, proposal review và live-provider governance proof thuộc các tranche S1/S2/S3/S5 tiếp theo.

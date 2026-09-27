# S1 snapshot và evidence contract

**Trạng thái:** SPEC_APPROVED / BUILD_COMPLETE / REVIEW_PASS / FREEZE_OPEN · **Ngày:** 2026-09-27 · **Tranche:** `CCMAI-RUNTIME-002` · **Rủi ro:** R2. Independent REVIEW: `docs/reviews/CCMAI_RUNTIME_002_GATE_B_FINAL_REREVIEW_2026-09-27.md`.

## Mục tiêu

Mọi kết quả phân tích phải chỉ rõ nó được tạo từ phiên bản dữ liệu nào và finding chi tiết phải trỏ về tin nhắn có thật. Đây là nền dữ liệu cho machine gate S2; tranche này chưa quyết định có gọi AI hay không và chưa chứng minh CVF kiểm soát runtime AI.

Trước BUILD mới, assignee phải review độc lập `CCMAI-RUNTIME-001`. Chỉ disposition `PASS` hoặc `PASS_WITH_REPAIRS` sau khi repair đạt mới mở cổng triển khai spec này.

## Hợp đồng snapshot

Một snapshot thuộc đúng một `job_run`, tenant và conversation. Snapshot schema version đầu tiên là `ccma.snapshot.v1`.

Snapshot builder phải dùng chung cho đường single và batch, đồng thời:

1. đọc lỗi truy vấn thay vì coi lỗi là danh sách rỗng;
2. sắp xếp ổn định theo `sent_at` UTC, sau đó `message.id`;
3. giữ `message_id`, `external_message_id`, role/name, content, content type, timestamp RFC3339Nano UTC và trạng thái attachment coverage;
4. tạo transcript có `message_id`, timestamp đầy đủ và timezone trên từng dòng;
5. tạo canonical manifest không phụ thuộc map order, giờ build hoặc timezone máy;
6. tính SHA-256 lowercase hex trên canonical manifest;
7. phân loại coverage thành `complete`, `partial` hoặc `empty`, kèm reason code có kiểu; content type không hỗ trợ, attachment không được biểu diễn, JSON attachment lỗi hoặc thiếu lịch sử phải không được gọi là `complete`.

`AnalysisSnapshot` lưu identity, schema version, digest, coverage và canonical manifest cần cho audit. Manifest ưu tiên ID, metadata và content hash; không nhân bản toàn bộ transcript/nội dung thô. Dữ liệu vẫn phải xóa được theo tenant/conversation cùng các kết quả hiện hành.

## Hợp đồng evidence

Finding `qc_violation` và `classification_tag` phải có ít nhất một evidence ref:

```json
{
  "message_id": "internal-message-uuid",
  "quote": "đoạn trích chính xác",
  "start": 0,
  "end": 21
}
```

- `message_id` phải thuộc snapshot của đúng conversation.
- `quote` phải là substring chính xác trong content của message đó.
- `start`/`end` là offset theo Unicode code point, `start` inclusive và `end` exclusive; nếu có thì phải khớp quote.
- Evidence text cũ có thể dùng để hiển thị, nhưng không thay thế evidence refs có cấu trúc.
- `conversation_evaluation` có thể không có evidence refs; nó vẫn phải tham chiếu snapshot qua quan hệ snapshot.
- Toàn bộ response của một conversation bị từ chối theo transaction nếu bất kỳ finding nào có ref sai, thiếu hoặc trỏ chéo conversation.

Prompt JSON contract cho QC/classification phải yêu cầu evidence refs. Test double/mocks được cập nhật để kiểm tra parser và persistence, nhưng không được dùng làm proof về governance hay chất lượng AI.

## Persistence và compatibility

- Thêm model/bảng `analysis_snapshots` và liên kết `analysis_snapshot_id` từ `job_results`; tên cụ thể có thể theo convention hiện hữu nhưng API JSON phải rõ nghĩa.
- Không backfill giả cho kết quả cũ. Row cũ được phép có snapshot ref rỗng và API phải biểu diễn được trạng thái legacy/unverified.
- AutoMigrate phải khởi động lặp lại được trên schema `CCMA` đang rỗng dữ liệu nghiệp vụ.
- API/export/notification hiện hành không được crash khi gặp row legacy hoặc field mới.
- Tranche này không thêm PostgreSQL, pg-jev, Jev service/SDK, vector DB hoặc provider dependency.

## Acceptance

1. Unit test chứng minh cùng dữ liệu tạo cùng digest; đổi content, role, timestamp, content type hoặc attachment coverage làm digest đổi.
2. Unit test gồm tiếng Việt có dấu và emoji để chứng minh offset theo code point.
3. Unit test từ chối message ID lạ, quote sai, offset sai, evidence refs rỗng và ref chéo conversation.
4. Single và batch cùng dùng một snapshot/transcript builder và lưu snapshot relation cho mọi kết quả hợp lệ.
5. Lỗi load/build/persist snapshot được đếm là lỗi conversation; không gọi save như thành công và không tạo kết quả mồ côi.
6. Backend tests/build, frontend build, Compose restart trên `CCMA`, docs build, catalog check, workspace doctor và `git diff --check` đạt.
7. Không gọi provider thật, không sync kênh thật, không gửi dữ liệu khách hàng và không claim machine gate/cost saving đã hoạt động.

## Deferred

Candidate selector, typed eligibility/execution/disposition, provider admission, budget, cache reuse, human proposal lifecycle và live-provider evidence thuộc S2/S3/S5. S2 được phép dùng `coverage` và digest của tranche này, nhưng không được triển khai ngầm trong S1.

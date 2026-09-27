# Independent review: CCMAI-RUNTIME-002 Gate B

**Reviewer:** Codex (`REVIEWER`, độc lập với implementer) · **Ngày:** 2026-09-27 · **Target:** `57951b0655f320382d2686a5bc2261cf33b98037` · **Disposition:** `CHANGES_REQUIRED`.

## Authority và claim boundary

Review đối chiếu source/commit với:

- `docs/specs/RUNTIME_SNAPSHOT_EVIDENCE_S1_2026-09-27.md`;
- `docs/work_orders/CCMAI_RUNTIME_002.md` Gate B;
- `docs/reviews/RUNTIME_SNAPSHOT_EVIDENCE_S1_BUILD_2026-09-27.md`.

Review chỉ xác minh data contract, persistence và test-double behavior. Không provider API, API key, channel sync hoặc dữ liệu khách hàng được dùng. Kết quả này không chứng minh AI quality, Vietnamese understanding, machine gate, cost saving hay CVF runtime governance.

## Phần đạt

- Single và batch cùng gọi `loadConversationSnapshot`/`buildConversationSnapshot`.
- Manifest struct, stable message ordering, UTC timestamp và sorted reason codes tạo canonical bytes ổn định cho projection hiện tại.
- Transcript có internal message ID và timestamp đầy đủ; prompt QC/classification yêu cầu evidence refs.
- Validation từ chối empty/foreign/cross-conversation/wrong-quote/wrong-offset refs trước transaction.
- Snapshot và toàn bộ results của một conversation được ghi trong cùng transaction; test rollback đạt.
- Nullable snapshot reference giữ compatibility cho legacy result; AutoMigrate và core persistence tests đạt trên MySQL.
- Gate A repairs về scheduler retry, UI wording và corpus test hợp lý; review này không thay disposition Gate A.

## Blocking findings

### R2-B1 — HIGH — đường xóa làm sai evidence lineage và retention

`DeleteChannel` xóa `analysis_snapshots` nhưng không xóa `job_results`. Các result còn `analysis_snapshot_id`; `AfterFind` chỉ kiểm tra pointer khác rỗng nên vẫn trả `evidence_status=snapshot_bound` dù snapshot đã mất. Đây là trạng thái audit sai và vi phạm yêu cầu xóa snapshot cùng result/conversation.

Theo hướng ngược lại, `ApplyPrunePlan` chỉ xóa stale `job_results` và giữ snapshot tương ứng. Manifest còn tenant/conversation/message IDs, sender name, external message ID và content hash, nên prune result không hoàn tất retention lifecycle.

Repair acceptance:

1. Channel delete xóa result và snapshot đúng thứ tự trong transaction, kiểm tra lỗi và không để dangling reference.
2. Prune xóa snapshot của đúng `(tenant, conversation, stale run)` sau khi mọi result tham chiếu đã bị xóa; giữ snapshot nếu còn result tham chiếu.
3. Có DB-backed regression tests chứng minh không còn orphan result/snapshot và rollback khi một bước lỗi.

### R2-B2 — MEDIUM — digest không định danh thay đổi attachment

Manifest chỉ giữ `attachment_coverage` và `attachment_count`. Hai attachment khác URL/name/type/local path nhưng cùng count và cùng coverage tạo cùng digest. Vì vậy claim “exact conversation version” và mục tiêu source-version chưa đúng cho message có attachment.

Repair acceptance:

1. Manifest thêm fingerprint deterministic của attachment metadata mà không nhân bản payload thô.
2. Với JSON hợp lệ, fingerprint dùng canonical typed fields; với JSON lỗi, lưu hash của raw bytes cùng coverage reason `ATTACHMENT_JSON_INVALID` để source change vẫn đổi digest.
3. Test chứng minh đổi URL/name/type/local path hoặc raw invalid attachment làm digest đổi; cùng attachment JSON hợp lệ tạo cùng digest.

### R2-B3 — LOW — DB test còn mặc định CQA

`connectTestDB` trong file mới `snapshot_db_test.go` fallback về `cqa:cqa_password@.../cqa`. Điều này tái đưa identity/database cũ vào tranche CCMA và có thể làm test chạm nhầm schema nếu môi trường cũ còn tồn tại. Lượt review trên host đã thực tế thử fallback này rồi skip vì port không mở.

Repair acceptance: DB-backed test chỉ chạy khi `TEST_DB_DSN` được cấp rõ, nếu thiếu thì skip với thông báo; không hardcode credential/schema CQA hoặc CCMA.

## Verification đã chạy

- Workspace doctor: PASS `25/25`.
- Snapshot/evidence unit tests, Gate A regression tests và `backend/ai` tests: PASS.
- Go 1.26 container trên network Compose `ccma_default`, MySQL `CCMA`: DB-backed single/batch, invalid evidence, cross-conversation, rollback và legacy tests PASS.
- Source inspection xác nhận R2-B1 tại `DeleteChannel`/`ApplyPrunePlan` và R2-B2 từ các field digest-bearing của `snapshotMessage`.

## Disposition và next move

Gate B giữ `REVIEW`; không FREEZE và không mở S2. Claude được phép chuyển sang `REPAIR_WORKER` trong cùng objective/path/risk/external-effect/commit boundary của `CCMAI-RUNTIME-002`, sửa ba finding trên, cập nhật BUILD evidence và tạo local repair commit không push. Sau đó Codex re-review changed set và regression evidence.

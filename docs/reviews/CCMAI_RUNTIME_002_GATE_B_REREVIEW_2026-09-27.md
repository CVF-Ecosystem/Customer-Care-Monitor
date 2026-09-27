# Re-review: CCMAI-RUNTIME-002 Gate B repair round 1

**Reviewer:** Codex (`REVIEWER`) · **Ngày:** 2026-09-27 · **Target repair commit:** `f924a6ba4c7e43030e6c165aa8fdad1a80904cfa` · **Disposition:** `CHANGES_REQUIRED_ROUND_2`.

## Kết quả ba finding trước

- **R2-B1:** prune cleanup đã đạt; channel-delete cleanup mới đạt happy path, còn thiếu atomic read/rollback boundary.
- **R2-B2:** digest đã có attachment fingerprint, nhưng encoding fingerprint còn collision.
- **R2-B3:** đạt; `snapshot_db_test.go` không còn fallback DSN CQA.

Các regression test của repair đều PASS trên Go 1.26 container và MySQL `CCMA`: snapshot digest test, channel-delete happy path, prune orphan/reference guard, single/batch contract và result transaction rollback.

## Blocking findings round 2

### R2-RR1 — HIGH — channel cascade chưa nằm trọn trong transaction

`DeleteChannel` đọc `convIDs` trước `db.DB.Transaction` và bỏ qua lỗi `Pluck`. Một conversation được tạo giữa lần đọc này và câu `DELETE conversations WHERE channel_id=...` sẽ bị xóa, nhưng message/result/snapshot của nó không nằm trong `convIDs` nên bị bỏ lại. Nếu `Pluck` lỗi, handler vẫn có thể tiếp tục xóa channel. Ngoài ra `os.RemoveAll` chạy trong transaction trước DB commit; DB rollback không thể phục hồi file đã xóa.

Repair acceptance:

1. Đọc/lock channel và conversation IDs bên trong transaction, kiểm tra lỗi; dùng locking đủ để ngăn insert cùng channel xuyên qua khoảng cascade hoặc dùng ràng buộc/cascade tương đương.
2. Chỉ thực hiện filesystem cleanup sau DB commit; lỗi file được log/quan sát mà không làm giả rollback DB.
3. Thêm DB-backed failure-path test buộc một bước cascade lỗi và chứng minh channel, conversation, result, snapshot vẫn nguyên vẹn; happy-path test tiếp tục đạt.

### R2-RR2 — MEDIUM — attachment fingerprint có delimiter collision

Fingerprint hiện nối typed fields bằng `0x1f` và các attachment bằng `0x1e` mà không escape. Hai attachment khác nhau vẫn có canonical string và SHA-256 giống nhau, ví dụ:

- A: `type = "a\u001fb"`, `url = "c"`
- B: `type = "a"`, `url = "b\u001fc"`

Re-review tái hiện `separator_collision=True` và `hash_collision=True` với hash `76e29e67593af52c713c94c1b8bacba503f828c2dc3df09523bbfa41ea3558f8`.

Repair acceptance:

1. Serialize typed attachment slice bằng encoding không nhập nhằng, ưu tiên `json.Marshal` trên struct/slice có field order cố định, rồi SHA-256 canonical bytes.
2. Invalid JSON phải hash raw bytes đúng nghĩa; không trim trước hash nếu claim source-byte change làm digest đổi.
3. Test delimiter-collision pair tạo digest khác; JSON hợp lệ khác whitespace/key order nhưng cùng typed values tạo digest giống; invalid raw bytes khác nhau tạo digest khác.

## Claim boundary và next move

Gate B giữ `REVIEW_PENDING`; không FREEZE và không mở S2. Claude được phép repair round 2 trong cùng work order/path/risk/external-effect/commit boundary, cập nhật repair evidence và tạo local commit không push. Sau đó Codex re-review changed set. Không provider call, channel sync, customer data, deployment hoặc governance claim được phép.

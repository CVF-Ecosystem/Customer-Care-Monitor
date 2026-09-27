# Re-review: CCMAI-RUNTIME-002 Gate B repair round 2

**Reviewer:** Codex (`REVIEWER`) · **Ngày:** 2026-09-27 · **Target repair commit:** `a17b50a0b5659b68b2bf5dc49893f7d0a71e904d` · **Disposition:** `CHANGES_REQUIRED_ROUND_3`.

## Kết quả R2-RR1/R2-RR2

- Attachment fingerprint dùng `json.Marshal` trên typed slice, phân biệt collision pair, ổn định qua whitespace/key order và hash raw invalid bytes: **PASS**.
- Conversation locking read đã chuyển vào channel-delete transaction; file cleanup chạy sau commit; trigger failure test chứng minh DB rollback: **PASS cho cascade đã thực thi**.
- Targeted Go 1.26/MySQL tests cho attachment, channel happy/failure paths, single/batch và save rollback: **PASS**.

## Blocking finding mới

### R2-RR3 — HIGH — evidence writer không tham gia deletion locking protocol

`saveResults` tạo `analysis_snapshots` và `job_results` trong transaction nhưng không khóa hoặc xác nhận `Conversation`/`JobRun` còn tồn tại. Hai bảng evidence cũng không có foreign key tới conversation/job run. Vì vậy một analysis đã load snapshot có thể hoàn tất sau khi channel, conversation hoặc job run bị xóa và ghi lại evidence rows mồ côi.

Locking read trong `DeleteChannel` chỉ chặn writer cùng khóa/constraint. Nó không chặn insert thẳng vào `analysis_snapshots`/`job_results`. Schema probe trong transaction xác nhận insert snapshot với tenant/run/conversation đều không tồn tại vẫn thành công (`count=1` trước rollback, `0` sau rollback cleanup).

Đây là root cause mới ở writer/deletion dependency edge, khác với hai repair trước. Vì vậy repair round 3 được phép theo Governance Latency rule; chưa cần `REVIEW_COST_ESCALATION_REQUIRED`.

## Repair acceptance round 3

1. Đảm bảo invariant ở tầng database hoặc locking protocol: snapshot/result không thể commit nếu conversation hoặc job run đã bị xóa; parent deletion không thể commit và để writer tạo evidence sau cascade.
2. Audit mọi parent-deletion path liên quan: `DeleteChannel`, `PurgeChannelConversations`, `DeleteJob`, `ClearJobRuns`, demo reset và analyzer `saveResults`. Dùng transaction/error handling nhất quán hoặc database constraints để đóng dependency edge thay vì chỉ sửa một handler.
3. Giữ legacy compatibility có chủ đích: nullable snapshot ref cũ vẫn đọc được; migration phải nêu cách xử lý row hiện hữu không hợp lệ, không backfill giả.
4. DB-backed concurrency tests chạy cả hai thứ tự:
   - writer giữ parent trước, delete chờ rồi xóa sạch evidence;
   - delete giữ/xóa parent trước, writer phải fail và không tạo orphan.
5. Regression tests round 1/2 và single/batch/evidence validation tiếp tục đạt; AutoMigrate lặp lại được trên `CCMA`.

## Claim boundary và next move

Gate B giữ `REVIEW_PENDING`; không FREEZE và không mở S2. Claude được phép repair round 3 cho root cause mới này trong cùng R2 work order, local commit không push, rồi Codex re-review. Không provider call, channel sync, customer data, deployment hoặc AI-runtime-governance claim được phép.

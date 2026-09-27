# Re-review: CCMAI-RUNTIME-002 Gate B repair round 3

**Reviewer:** Codex (`REVIEWER`) · **Ngày:** 2026-09-27 · **Target repair
commit:** `7a284b8` · **Disposition:**
`CHANGES_REQUIRED / REVIEW_COST_ESCALATION_REQUIRED`.

## Phần đạt

- `saveResults` khóa và xác nhận `Conversation` rồi `JobRun` trong transaction
  trước khi tạo snapshot/result: **PASS**.
- `DeleteChannel`, `PurgeChannelConversations`, `DeleteJob`, `ClearJobRuns` và
  demo reset đều lấy conflicting parent locks trước child cleanup: **PASS cho
  locking order**.
- Hai MySQL race tests đạt độc lập: delete-first làm writer fail và không tạo
  orphan; writer-first giữ lock làm delete chờ rồi dọn sạch evidence: **PASS**.
- Targeted rollback, channel/job happy path và repair-round regressions được
  chạy lại trên disposable MySQL `CCMA`: **PASS**.
- Nullable legacy snapshot reference và `legacy_unverified` không bị đổi; round
  này không thêm constraint/migration: **PASS**.

## Blocking finding

### R3-E1 — HIGH — demo reset bỏ qua lỗi child delete và commit orphan evidence

`ResetDemoData` kiểm lỗi của hai locking reads và `Commit`, nhưng không kiểm
`.Error` của các lệnh `DELETE` tại `backend/api/handlers/demo.go:447-457` hoặc
lệnh cập nhật tenant tại dòng 460. Trong MySQL, một statement lỗi không tự làm
toàn bộ transaction thất bại. Các statement sau vẫn chạy và `Commit` vẫn có thể
thành công.

Failure injection độc lập dùng một `BEFORE DELETE` trigger trên `job_results`
đã tái hiện chính xác:

```text
Error 1644 (45000): forced demo reset failure
ResetDemoData returned HTTP 200
job_results=1, job_runs=0, conversations=0
```

Như vậy handler xóa parent `JobRun`/`Conversation`, giữ lại `JobResult` mồ côi,
xóa demo flag và báo thành công. Điều này vi phạm acceptance Round 3 số 1 và 2:
parent deletion phải atomic, mọi deletion path phải có transaction/error
handling nhất quán.

Test tái hiện được tạo tạm, chạy bằng:

```powershell
$env:TEST_DB_DSN='ccma:***@tcp(127.0.0.1:33064)/CCMA?charset=utf8mb4&parseTime=True&loc=Local'
go test ./api/handlers -run '^TestRound3ReviewDemoResetRollsBackDeleteFailure$' -count=1 -v
```

Test thất bại đúng tại assertion trên; trigger, fixture, file test tạm và
disposable container đã được dọn. Không có secret thật, provider call, customer
data hoặc persistent Compose mutation.

## Escalation và acceptance nếu owner cho phép repair tiếp

Đây không phải root cause độc lập mới. Nó là phần error handling còn thiếu của
chính R2-RR3 và đã được nêu rõ trong work order Round 3. Vì tranche đã tới repair
round ba, Governance Latency rule yêu cầu dừng và ghi
`REVIEW_COST_ESCALATION_REQUIRED`; Codex không tự cấp round tiếp theo.

Nếu owner cho phép một repair hẹp, acceptance là:

1. Demo reset phải rollback và trả lỗi khi **bất kỳ** lock, delete, update hoặc
   commit step nào lỗi; không statement nào bị bỏ qua `.Error`.
2. Thêm permanent MySQL failure-path test tương đương trigger reproduction:
   response không được là `200`, và result/snapshot/run/job/conversation/channel
   cùng demo flag phải còn nguyên sau rollback.
3. Chạy lại hai race ordering, toàn bộ repair-round regressions, `go test ./...`,
   AutoMigrate hai lần, catalog và workspace doctor.

Gate B giữ `REVIEW_PENDING`. Không FREEZE, S2, provider call, channel sync,
customer data, deployment hoặc push được phép từ review này.

## Owner disposition

Owner approved the exact narrow repair above on 2026-09-27. The escalation is
resolved only for `ResetDemoData` atomic error handling, its permanent MySQL
failure-path test, required regressions/evidence and continuity synchronization.
No broader repair or product scope is authorized.

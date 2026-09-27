# CCMAI-RUNTIME-002: independent review rồi xây snapshot/evidence S1

**Trạng thái:** GATE_A_PASS_WITH_REPAIRS / GATE_B_REVIEW_CHANGES_REQUIRED · **Rủi ro:** R2 · **Ngày:** 2026-09-27 · **Assignee dự kiến:** agent do owner chọn (Claude).

## Authority và mục tiêu

Owner giao Codex làm `ORCHESTRATOR/REVIEWER` và sẽ giao work order này cho Claude thi công. Authority kỹ thuật là `docs/specs/RUNTIME_SNAPSHOT_EVIDENCE_S1_2026-09-27.md`; roadmap chỉ định hướng, không thay acceptance của spec.

Work order có hai gate tuần tự. Gate A đóng review độc lập cho tranche trước. Gate B xây phần S1 còn thiếu để mỗi finding có source version và evidence ref kiểm chứng được. Không được bắt đầu Gate B nếu Gate A chưa đạt.

## Gate A — review độc lập `CCMAI-RUNTIME-001`

Claude nhận vai `REVIEWER`, rehydrate continuity và ghi role acknowledgment vào active handoff trước khi chạy review. Review commit `ade74addfd9890c3418c99ee02aecd6b73ee3b4a` theo:

- `docs/specs/RUNTIME_FOUNDATION_S0_S1_2026-09-27.md`;
- `docs/work_orders/CCMAI_RUNTIME_001.md`;
- `docs/reviews/RUNTIME_FOUNDATION_S0_S1_BUILD_2026-09-27.md`.

Phải kiểm source và test cho checkpoint `success/partial/error`, after-sync suppression, aggregate error propagation, bounded observability, attachment failure, scheduler activity ownership và UI wording. Ghi `docs/reviews/CCMAI_RUNTIME_001_INDEPENDENT_REVIEW_2026-09-27.md` với disposition `PASS`, `PASS_WITH_REPAIRS` hoặc `FAIL` và lệnh/evidence thật.

Nếu có defect trong phạm vi tranche cũ, chuyển vai có ghi nhận sang `REPAIR_WORKER`, sửa trong scope của `CCMAI-RUNTIME-001`, test lại và cập nhật review. Nếu đến repair round ba mà không có root cause độc lập mới, dừng với `REVIEW_COST_ESCALATION_REQUIRED`. `FAIL` chưa sửa xong chặn Gate B.

## Gate B — BUILD `CCMAI-RUNTIME-002`

Sau khi Gate A đạt, Claude ghi chuyển vai `REVIEWER/REPAIR_WORKER -> IMPLEMENTATION_WORKER` trong active handoff, rồi triển khai đúng spec.

### Allowed paths

- `backend/db/models/` và `backend/db/mysql.go` cho snapshot relation/migration;
- `backend/engine/` cho builder, canonical digest, evidence validation, single/batch integration và test;
- `backend/ai/prompts.go` cùng test để bổ sung contract JSON và transcript có source identity;
- API/export/notification test hoặc code tối thiểu cần thiết để giữ compatibility với field mới;
- `docs/specs/`, `docs/work_orders/`, `docs/reviews/`, indexes/catalog, `IMPLEMENTATION_STATUS.json` và `CVF_SESSION/`;
- local Compose project `ccma` và schema phát triển `CCMA` để xác minh AutoMigrate/restart.

Nếu cần sửa file ngoài danh sách để thỏa acceptance, phải ghi dependency và lý do vào handoff trước khi sửa. Thay đổi vẫn phải nằm trong snapshot/evidence contract và không được mở rộng external-effect class.

### Forbidden scope

- Không gọi Claude/Gemini/OpenAI/xAI hay provider API nào; không dùng API key.
- Không chạy channel sync thật, không nhập dữ liệu khách hàng, không deploy, push hoặc public release.
- Không triển khai candidate machine gate, provider admission, budget, cache, auto reply hay human disposition UI.
- Không thêm Jev/pg-jev/PostgreSQL/TypeSafe dependency và không sửa CVF core.
- Không dùng mock để tuyên bố governance, chất lượng tiếng Việt hoặc tiết kiệm chi phí.

## Required evidence

Tạo `docs/reviews/RUNTIME_SNAPSHOT_EVIDENCE_S1_BUILD_2026-09-27.md` gồm:

- changed-set và schema/migration outcome;
- bảng ánh xạ từng acceptance sang test/lệnh/kết quả;
- bằng chứng single/batch dùng chung builder;
- test digest, Unicode offset, cross-conversation ref và transaction rollback;
- backend test/build, frontend build, Compose restart/migration, docs/catalog/doctor/diff checks;
- xác nhận không provider call, không customer data và claim boundary.

Sau BUILD, chuyển vai sang `SESSION_SYNC_STEWARD`, đồng bộ active state, handoff, session memory, implementation truth và indexes. Trạng thái phải là `REVIEW_PENDING`; Codex giữ independent `REVIEWER` cho Gate B nên Claude không được tự FREEZE tranche này.

## Commit authority

Owner đã cấp quyền thi công qua work order này. Claude được tạo local commit bằng Git identity đã cấu hình `Blackbird081 <nmtienctt@gmail.com>` sau khi checks đạt. Không push. Commit phải chứa code, test, evidence và continuity tương ứng; báo commit hash cho owner/Codex review.

## Failure conditions

Dừng Gate B nếu snapshot digest không deterministic, evidence sai vẫn được lưu, single/batch dùng hai contract khác nhau, migration làm mất dữ liệu hoặc khởi động lại thất bại. Dừng và báo boundary change nếu giải pháp cần provider/network, credential, dữ liệu thật, destructive database action, deploy hoặc file CVF core.

Role route: ORCHESTRATOR/WORK_ORDER_AUTHOR (Codex) → REVIEWER (Claude, Gate A) → REPAIR_WORKER nếu cần → IMPLEMENTATION_WORKER (Claude, Gate B) → SESSION_SYNC_STEWARD/COMMIT_STEWARD (Claude) → REVIEWER (Codex, Gate B) → CLOSER sau khi review đạt.

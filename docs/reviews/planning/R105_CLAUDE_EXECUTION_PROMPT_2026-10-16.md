# R105 — prompt giao Claude thực hiện UI, authority đã chuẩn bị

**HOLD 2026-10-16:** formal order is now BLOCKED_WITH_REASON following owner request for Codex/subagent execution and actual missing browser surface. Historical prompt retained; do not execute until Root issues an effective reviewed route.

Đọc và thực hiện work order chính thức:
`docs/work_orders/CCMAI_RUNTIME_105_R101_UI_EVIDENCE.md`

Bạn là REVIEW_EVIDENCE_OPERATOR được seed chỉ định: Claude / Antigravity-IDE. Codex /root là orchestrator, independent reviewer và commit steward. Không dùng prompt cũ hoặc planning draft làm authority.

Trước effects: tuân thủ AGENTS First-Request Protocol; đọc canonical state/handoff R105, formal order, immutable seed `CVF_SESSION/authority/CCMAI-RUNTIME-105.json`, tranche record, source review/receipt và learning dispatch-readiness. Phát declaration và ghi acknowledgment vào báo cáo mới. Xác minh checkout đã có dispatch commit thật, work order ACTIVE / record BUILD / current105BUILD, baseCommit `760726e691114a85af315ab5823112375c6af570`, immutable source/pins đúng. Root đã cấp quyền campaign hữu hạn này; không dừng để hỏi lại authority đã rõ. Dừng chỉ khi có drift/thay đổi thực tế, không tự sửa seed/runner/manifest.

Mục tiêu là B01-B10 của bản R101 lịch sử200df3e qua browser_subagent thật, backend/MySQL synthetic thật, không phải chụp lại trang CLICKED. R104 data-URL provenance đã review; không chạy lại chẩn đoán bridge, không cài lại Playwright.

Ghi `git rev-parse HEAD` và thực hiện đầy đủ các lệnh `git cat-file` / `git diff HEAD --exit-code` ở mục Execution của work order trước runner. Tất cả phải exit0, HEAD phải chứa order ACTIVE và record/state BUILD; admission của runner riêng lẻ không chứng minh metadata đã commit. Lưu kết quả thật trong hồ sơ trả về.

Chạy đúng runner theo work order; không tự start fixture/backend/Vite thủ công. `start` là supervisor foreground: giữ tool/session đó sống khi thao tác browser ở tool khác. Chờ BOTH ready.json và running.json mới trong `.cvf/runtime/r105_claude/`, dùng đúng origin3011 và UUID/email của campaign mới. Không dùng old browser_ready.json, port/PID cũ hay native tests.

Runtime budget: existing fixture start1/backend1/Vite1; new fixtures0; retries0; compile/test/build/install/download/provider/mock/secret/customer/channel0; tối đa3 tab product mới/10 action submissions, backend30min/campaign35min bao gồm cleanup. Outputdir đã tồn tại nghĩa là campaign đã bị tiêu thụ: không xóa/reset/đổi suffix/chạy lại.

Dùng Antigravity browser_subagent theo operations được tool hỗ trợ trên HTTP origin thực. Không file://, raw CDP, standalone browser, interception hoặc workaround khi policy từ chối. Khi block/defect, lưu lỗi thật, dừng phần phụ thuộc và cleanup đúng work order; không sửa source hoặc 3 lỗi Dashboard baseline. Tab diagnostic R104 cũ giữ nguyên, không dùng/đóng nó hoặc tab của người dùng.

Thực hiện ma trận B01-B10 theo thứ tự trong order; chỉ PASS khi có quan sát thật. Thiếu/không hỗ trợ ghi NOT_RUN có lý do. Chụp original proposal/history trước đổi dữ liệu, lưu tool invocation/response/recording và before/after DOM cùng ảnh gốc. Không dựng raw log từ lời kể. Theo dõi action count và giữ các lần fail.

Cuối campaign gọi cooperative `stop` theo order, giữ supervisor để finally cleanup bằng process handle của nó. Chờ supervisor exit/cleanup.json, ghi actual exits/errors, fixture stopped/protection comparison và đóng chỉ tab product mới do nhiệm vụ tạo. Supervisor bị kill hoặc cleanup không hoàn tất: report residue và BLOCKED, không broad-kill/retry.

Bàn giao:
- `docs/reviews/R105_R101_UI_WORKER_RETURN_2026-10-16.md`
- `docs/reviews/probes/r105_claude/`: toàn bộ runtime receipts/log export, browser raw evidence, ảnh/recording, bảng B01-B10, cleanup/protection và SHA256+bytes manifest.
- Static worker-return gate với explicit formal order + report; lưu command/stdout/stderr/exit và tất cả N/A/FAIL.

Status COMPLETE_PENDING_REVIEW hoặc BLOCKED_WITH_REASON. Không tự tạo Root review, commit/push/merge/deploy/approve/FREEZE. Không tuyên bố currentR102 backend, fullM3/M2/pilot/release hoặc AI governance đã PASS. Trả hồ sơ để Codex /root review độc lập.

Bắt đầu làm sau rehydration/admission; không dừng ở việc đề xuất kế hoạch.

# R139: đề xuất gói fresh campaign trong cùng work order

Status: PROPOSED_NOT_AUTHORIZED
Risk: R2
Worker: Root ORCHESTRATOR / SPEC_AUTHOR
Reviewer: Sol Medium /root/r139_source_evidence_sol — DESIGN_READY_FOR_OWNER_REVIEW_ONLY
Baseline: `bb7ded1b5b5b2fca410a50a4144b25520e3da32e`

R139 đã nhận bản sửa receipt/Status ở tầng source/IO: Go6/6, workerPython30/30, RootPython30/30. Runtime trước vẫn MANUAL_HOLD/cleanupHOLD; slot API1/1 MAY_HAVE_STARTED đã tiêu thụ, request/response/usage/billing còn thiếu theo receipt. Không thể dùng lại fixture_r2 hoặc ngân sách cũ.

Gói này đề xuất một phê duyệt gộp có điều kiện để tránh tách từng hằng số, compile và effect thành các lần xin lại. Nó chưa cho phép source/check/runtime/key/API. Hợp đồng chi tiết và cap chuẩn là `probes/r139_integrated_20261021/r5_fresh_campaign_proposal.json`; patch dự kiến là `r5_fresh_campaign_source_proposal.patch` cùng thư mục. Patch chỉ là tài liệu review, chưa áp dụng.

## Phạm vi nhỏ nhất

Ba file source gốc; đúng bảy literal changes: Go đổi external fixture_r2→fixture_r3; supervisor đổi tên fixture, database, ignored runtime root, external evidence path; Python pure test đổi tên bài và expected path. Không sửa hành vi sản phẩm, policy, shared transport/helper, dependency hay Core; không tổng quát hóa path. Mọi evidence/seed/failure cũ giữ nguyên.

Đề xuất đúng một source round: offline compile1, six-test pureGo1 qua argv-list/shellFalse, workerPython1, RootPython1 độc lập; không retry. Chỉ sau source review PASS và commit mới có thể ghim source/binary mới cho runtime. Lỗi ở bất cứ bước nào dừng chain, không tự phát sinh round tiếp.

## Campaign đề xuất và chi phí

Một fixture fresh r3, một supervisor/heldGo/backend/Vite/productIABtab; native8 POST + browser1 POST, grants3/authPOST4, tối đa một API Alibaba synthetic mới, request6000 byte/completion768/thinkingfalse, không retry. Namespace/resource/path mới nêu rõ trong JSON. Campaign1200s, cleanup180s, MySQL finalTCP60s/max45probes; raw inventory và exact-owned stop/wait/listener proof bắt buộc. Không thử Docker/browser/API như một campaign phụ.

Đề xuất **thêm tối đa USD1 cho đúng một API mới**. Đây chưa phải ngân sách đã duyệt, không dùng lại USD1 cũ. Nếu Owner duyệt, tổng các trần đã cho phép là USD2 (cũ1+mới1), không phải số tiền thực đã dùng; billing lần cũ UNKNOWN. Root chỉ đọc staged raw key hiện có một lần ở runtime, không đọc lại config/restage/hash/echo key. Không giả định free quota hay giá tài khoản đã xác minh.

Runtime admission sẽ chỉ được tạo sau source PASS: đủ9sourcehash/tree/binary/executable/Vite/browserproof, exact fresh resources, UTC deadline còn hiệu lực, Owner cost/key/effect grant, reviewer độc lập; commit trước mọi effect. Không dùng placeholder hoặc reuse admission/deadline cũ. Root execute; Sol khác review runtime độc lập.

## Mốc đạt để nghiệm thu

| Mốc | Điều kiện đạt |
| --- | --- |
| P1 source | Đúng patch; focused checks PASS; Root review độc lập và commit source/binary pin. |
| P2 admission | Exact pins, authority, deadline, tài nguyên fresh, browser/source/readiness đủ trước fixture; bất kỳ thiếu sót HOLD. |
| G1 native | Native8POST/rejection7/18 prebrowser reads; trap0; đủ no-write/accounting. |
| G2 browser/provider | Một execute; request/response thật; publication lineage/receipt; OPEN/detail/status/history/reload/reader facts. Manual hold/uncertain dừng, không resend. |
| Cleanup/review | Original waits, containerPID0,3listenersabsent, protected inventory nguyên vẹn ngoài normalize đã duyệt; independent disposition. |
| G3 roadmap | Gán receipt vào từng E01–E06/M3; không suy fullM3 từ một campaign dương tính. |

Các nhánh E04 crash/storage/concurrency/5xx/offline không được tự nhận đã kiểm bằng luồng này. M1/M2/pilot/release, hosted CI và Facebook/Zalo OA chưa đóng. Không cam kết ngày hoàn tất hoặc phần trăm.

Owner cần phê duyệt gói source/check/fresh fixture/new API/new USD1 này vì authority hiện hành đã hết source/check/call caps; không có quyền thực thi từ chữ PROPOSED. Phê duyệt gộp vẫn giữ independent review và exact committed admission trước runtime; chỉ quay lại Owner nếu vượt scope/cap hoặc có uncertainty.

## Qualifications sau rà soát độc lập

Patch phát hành dưới dạng UTF-8/LF; khi checkout Windows đổi line endings, lấy đúng committed git-blob bytes vào vùng ignored và xác minh hash trước default git apply --check. Không dùng bỏ qua whitespace để mở rộng patch. Lỗi check bản patch CRLF ban đầu được giữ trong review, đã sửa tài liệu mà không chạm source.

Một lượt đọc trang giá/model công khai chính thức, không secret, được đề xuất ở preflight sau source PASS và chỉ sau Owner grant; nếu không xác minh được model/giá cho request giới hạn trong USD1 thì HOLD trước key/fixture/API. Không đọc tài khoản/billing/config riêng và không retry.

Forced kill có thể bỏ qua Go t.Cleanup; journal đã fsync/slot/raw log không biến missing final receipt thành PASS. Protected Status chỉ nhận Up/Exited; format khác hoặc State/health/exitcode/RunningFor đổi vẫn HOLD. E04 crash/storage/concurrency/error branches vẫn cần disposition riêng; không hứa fullM3.

## Preparation findings retained

Default apply-check of the initial CRLF documentary patch failed; LF bytes and updated hash passed without source edits. Catalog regeneration later hit an INDEX.md file lock while overlapping catalog preflight reported generated-view drift (6/7). Root serialized regeneration and requires a clean catalog/preflight/profile pass before commit; this is documentary tooling repair, not a new source/check/runtime round. Original failed tool outputs remain in the session/local preparation record.

The generated unified-diff artifact intentionally contains diff context prefixes before source TABs and blank lines. Unscoped git diff --cached --check reports those context markers as whitespace defects in patch-as-data; they are not changed source whitespace. Root checks every other staged path with that single generated .patch excluded and validates the patch itself with default git apply --check and its exact byte hash. No governance gate is skipped and source remains untouched.

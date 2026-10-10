# R067 visual failure review và đề nghị một lượt kiểm chứng bổ sung

Ngày 2026-10-09; root REVIEWER độc lập với Luna xhigh source/evidence worker. Disposition **CHANGES_REQUIRED**, FREEZE OPEN. Source `bd62c7e4e4868cc05528aa2b8a1ae05561a78df1` không đổi. Chưa push vì batch chưa đủ điều kiện local closure. Không chạy lại tự động.

Root campaign đã PASS 316/316 test trong 30 file suite (299 test cũ và 17 mới), hai mutation semantic detector FAIL đúng với healthy control PASS, restored 17/17 PASS. Worker tiếp tục đúng ba lượt còn lại: cùng hai controls và restored 17/17 PASS; positive đầu 315/1 FAIL được giữ, không biến thành worker positive final PASS. Tổng 8/8 Vitest, 3/3 forced builds, Go0. Compiler/Vite PASS tại product-identical `4d9214b`; chỉ test body mới khác ở `bd62c7e`, 120 file còn lại giống hệt, không compiler lại trên archive cuối. [Exact-source/raw audit](probes/r067_final_independent_source_evidence_audit_2026-10-09.json).

Root visual đầu thất bại do module specifier percent-encoded thành literal `%20` trước khi component render. NEW fixture bỏ encodeURI ở đúng ba import path, nhưng worker visual duy nhất vẫn exit1/capture exit2 tại assertion Tab/Enter/Space. Vite log lần này chỉ ghi server ready, không ENOENT; không PNG/report. **Nguyên nhân lần hai và việc component đã mount hay chưa chưa xác định.** Capture cũ throw trước khi lưu keyboard booleans/DOM/browser errors; không đủ dữ liệu để kết luận lỗi accessibility của sản phẩm hay sửa xong harness. [Root failure disposition](probes/r067_visual_fixture_failure_disposition_2026-10-09.json), [worker actual failure receipt](probes/r067_worker_visual_failure_receipt_2026-10-09.json). Worker không lưu riêng native top-level stdout/stderr; hai merged child logs thật được sao nguyên byte, không dựng lại stream thiếu.

| Contract | Independent disposition |
|---|---|
| UIR01 | Source + mounted integration PASS: details ở năm trạng thái, row binding, actions/results/cancel giữ nguyên. |
| UIR02 | Strict version/scope/tuple/metadata/size/JSON projection và malformed/binding tests PASS. |
| UIR03 | SP sparse outcomes là known zero; selected/unvisited null là unknown; sums/retained bounds reviewed và tests PASS. |
| UIR04 | EX counters/sums/prefix/trimmed retained calls và legacy optional UO reviewed, tests PASS. |
| UIR05 | RO bounded fingerprint, permission/policy/WAIT_DATA không được suy thành approval; source/copy/tests PASS. |
| UIR06 | UO known zero/unknown/null, tokens/cost/overflow/coherence reviewed; tests và mutation control PASS. |
| UIR07 | Fixed aggregate labels; no raw content/HTML dump/actions; independent unavailable sections reviewed/tested PASS. |
| UIR08 | EN/VI/source/mocked component tests có bằng chứng; actual browser keyboard và desktop/mobile readability **NOT ESTABLISHED**. |
| UIR09 | Mounted JobDetail no extra request/mutation, failed/no-results/running and mismatched row tests PASS. |
| UIR10 | 27 suite cũ/299 test, backend/tooling/seeds unchanged; root full suite PASS; compiler/Vite inherited with exact120-file identity and explicit no final-test-archive compile. |
| UIR11 | Hai actual single-source mutations, named detector failures/healthy controls/restored17 PASS; manifests/full121-file restoration verified. |
| UIR12 | Exact source/raw/archive/inventory/restoration audited; budgets retained, but actual visual proof incomplete. **PARTIAL**. |

**REVIEW_COST_ESCALATION_REQUIRED**: cả hai visual allowance đã hết, failure thứ hai chưa phân biệt được root cause. Không tự thêm lượt hay reset budget. Root đã chuẩn bị NEW capture/fixture/runner, hoàn tất syntax-only checks, không thực thi:

- Capture chờ điều kiện mount đủ bốn section, deadline 8 giây trong cùng navigation; không reload/retry. Lưu DOM trạng thái giới hạn, từng keyboard boolean và browser error packet trước khi throw, kể cả catch tổng quát.
- Fixture chỉ đổi capture entry của literal-path fixture; source UI/CSP/cached dependencies unchanged. Đây là sửa observability/readiness của harness, **chưa chứng minh sửa được nguyên nhân failure thứ hai**.
- Runner yêu cầu NEW owner grant `OWNER_APPROVED_EXACTLY_ONE_VISUAL`; kiểm source/archive121 file trước/sau, capture native stdout/stderr/raw child logs/report/PNG ở paths exclusive và backup hash; không overwrite packet cũ. Một launch, không retry.

Đề nghị **đúng một lượt visual bổ sung do root chạy**, EN/VI desktop/mobile với keyboard controls; tổng lịch sử visual tối đa3 thay vì2. Không thêm typecheck/build/Vitest/Go, không đổi product/source/test/seed/scope/risk/provider/customer/dependency, không dùng dữ liệu thật. Nếu PASS thì root kiểm trực tiếp bốn ảnh/report trước formal REVIEW lại và conditional scoped local FREEZE/ordinary authorized branch push. Nếu FAIL thì giữ packet và dừng, không duyệt trước lượt tiếp theo.

Lý do cần owner: [work order hiện hành](../work_orders/CCMAI_RUNTIME_067.md) giới hạn “One forced typecheck-build per role; accessible actual isolated synthetic browser visual QA (desktop/mobile) separate”; SPEC nêu “separately bounded one per role” và “no automatic retry”. Giới hạn visual đã dùng 1/1 mỗi role. `duyet` trước chỉ cấp thêm đúng một forced typecheck/build, đã dùng; không cấp lượt visual này. Đây là ranh giới ngân sách thực tế, không yêu cầu xác nhận lại sửa metadata cùng scope.

Root chịu trách nhiệm probe không lưu failure diagnostics và static approval fixture thiếu; đây không phải toàn bộ lỗi Luna. [Đánh giá tạm thời Luna/Sol](R067_LUNA_XHIGH_VS_SOL_MEDIUM_ASSESSMENT_2026-10-09.md) chưa phải đánh giá sau closure. Full backend DB/race/live/provider/channel/GitHub Actions/hosted governance NOT RUN trong tranche; core doctor chỉ local prerequisite, không AI governance proof. Facebook/Zalo OA vẫn parked.

# Reviewer tự xử lý sửa chữa nhỏ trong scope

Ngày: 2026-10-06. Nguồn: owner chỉ ra việc Codex chuyển lại subagent để sửa đoạn Current R053 trong memory, rồi yêu cầu mọi agent tự tuân thủ thay vì chờ owner nhắc. Đây là lỗi áp dụng quy tắc có sẵn. Maintainer: ORCHESTRATOR / SESSION_SYNC_STEWARD. Phạm vi lần cập nhật: tài liệu R1, không thay đổi policy, authority seed, source, test, bằng chứng runtime hay acceptance.

## Finding và nguồn kiểm chứng

Front marker/state/handoff đã ở REVIEW_PENDING nhưng đoạn current memory còn BUILD. Codex đúng khi dừng nghiệm thu, nhưng giao lại worker cho một sửa chữa metadata có giá trị đúng đã xác định, tạo thêm vòng chờ không cần thiết. Worker đã sửa tại `6a8919f983c46381f08e124352419e5bccb3a48a`; [review R053](../CCMAI_RUNTIME_053_SUCCESSOR_INDEPENDENT_REVIEW_2026-10-06.md) ghi nhận lỗi và acceptance tại `23d847a66ff2bc1ffa10a7270e555ad329916dc6`. Lỗi continuity và cách phân công không bị xóa khỏi lịch sử.

Căn cứ: project `AGENTS.md`, Provider-Neutral Role Contract và Governance Latency and Approval Continuity; [learning repair hiện hành](feedback_cvf_repair_workflow.md), các ví dụ reviewer sửa pointer R042 và metadata R044. Owner xác nhận reviewer tự sửa việc nhỏ. Yêu cầu độc lập cho sửa hành vi R2+ vẫn áp dụng; việc này không mở rộng quyền BUILD của reviewer.

## Bước bắt buộc trước khi trả finding hoặc giao sửa

Agent đọc record này ở INTAKE, trước REVIEW và trước sửa continuity. Agent tự phân loại finding trước khi hỏi owner hoặc giao lại worker:

1. Xác định giá trị đúng từ authority/evidence hiện hành; đọc cả current prose, marker, state, handoff, implementation truth và pointer liên quan. Không chọn một phía khi authority còn mâu thuẫn hoặc chưa đủ bằng chứng.
2. Kiểm tra đây có phải sửa nhỏ, reversible trong metadata/pointer/cách trình bày đã được phép, với tiêu chí nghiệm thu, source/test/runtime evidence và trách nhiệm BUILD/commit hiện hành không đổi.
3. Nếu đủ điều kiện, reviewer tự sửa trong scope đang được phép: ghi finding và chuyển vai trò sang SESSION_SYNC_STEWARD hoặc trách nhiệm sửa tài liệu tương ứng trước khi edit; giữ dissent/failure/history; kiểm tra bộ artifact liên quan trong một lượt; chạy checks phù hợp, sync và commit theo quyền hiện hành. Không yêu cầu owner nhắc, không thêm vòng giao subagent chỉ vì phát hiện trong REVIEW.
4. Chỉ trả worker hoặc escalate khi cần sửa hành vi/logic/test, thay authority/acceptance, chưa xác định được sự thật, vượt allowed paths/commit ownership, hoặc chạm ranh giới risk/provider/network/secrets/destructive/public release/deployment. Ghi ranh giới cụ thể và chuyển đúng trách nhiệm; áp dụng review độc lập và cost disposition khi thực sự cần.

`BLOCKED_CONTINUITY_DRIFT` vẫn dừng nghiệm thu cho đến khi continuity khớp. Nếu cách sửa nhỏ đã rõ và nằm trong quyền metadata của reviewer, reviewer thực hiện sửa rồi rehydrate/check trước khi tiếp tục nghiệm thu. Không biến việc dừng nghiệm thu thành yêu cầu giao worker cho mọi typo hoặc current prose lỗi thời. Không thay source, test, seed, historical receipt hoặc kết quả thất bại để làm gate xanh.

## Tự kiểm tra trước khi báo blocked hoặc yêu cầu xác nhận

- Finding đã được phân loại nhỏ trong scope hay cần độc lập? Đã ghi lý do dựa trên artifact/authority cụ thể?
- Nếu nhỏ, agent đang giữ trách nhiệm xử lý đã sửa, kiểm tra và commit chưa?
- Nếu chưa thể sửa, ranh giới hoặc fact thiếu cụ thể là gì? Có thật sự cần owner trả lời, hay tiếp tục được dưới authority đang có?
- Current prose/nested routing và machine pointers đã đồng nhất? Có giữ failed checks và hạn chế claim?

## Disposition và giới hạn

Project: owner clarification được đưa vào shared learning, startup memory và active handoff để dùng ngay; reviewer trực tiếp cập nhật tài liệu trong lần này. Publication checks được ghi ở `docs/reviews/probes/minor_reviewer_repairs_publication_2026-10-06.json`. R050/R051/R053 REVIEW_PASS / FREEZE_OPEN, runtime budget1/4 đã hết, live accounts parked; không có runtime mới.

Upstream: DEFERRED; có thể chuyển use case về phân loại sửa nhỏ và tránh operator wait cho CVF cha trong một intake được phép. Lần này không sửa CVF core hoặc gate/tooling. Các checks xác minh repository/tài liệu; không chứng minh mọi agent tự tuân thủ hoặc CVF điều khiển AI bằng runtime.

## Nested implementation projections (2026-10-08)

R064 resumed audit found current front tuple accepted while nested ruleObservation still REVIEW_PENDING/NOT_ESTABLISHED; direct repair at `0b97c05330eb0f3a5662591766c8165c3f1e1c2f`. A full nested status/acceptance/next-move audit then found the same stale projection in older executionObservationPlanning, despite canonical R061/R062 already FROZEN at99052ab. [R064 closure audit](../R064_LOCAL_RULE_OBSERVATION_CLOSURE_2026-10-08.md) records direct synchronization and retained historical snapshot. Source/old reviews/runtime untouched; no new acceptance decision or other-tranche closure.

Apply before continuity publication: enumerate every current nested status, sourceImplementation, independent acceptance, routing and next-move field; compare with canonical tranche/review, not just the front marker. Label historical snapshots explicitly and preserve counters/failures. Portable gate PASS does not establish completeness of these projections. Project applied locally by root; upstream DEFERRED, no machine gate or CVF core change.

R065 root audit additionally found ownerRouting still dispatching R060 and buildInProgress mixing stale R063/R060/current-historical prose. Root directly synchronized current projections and preserved entire historical snapshots at55272c8 before own campaign; [R065 review](../R065_INDEPENDENT_USAGE_OBSERVATION_REVIEW_AND_CLOSURE_2026-10-08.md) records verification. Include top-level routing/build lists in the same projection audit; no source/test/old packet change or owner wait. Upstream DEFERRED.

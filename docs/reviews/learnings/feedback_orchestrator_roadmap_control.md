# Orchestrator mất kiểm soát tiến độ roadmap

Ngày: 2026-10-12. Project: Customer-Care-Monitor-AI. Maintainer: ORCHESTRATOR. Phạm vi cập nhật: learning/tài liệu R1 do owner yêu cầu; không sửa source, gate, authority seed, ngân sách thực thi hoặc CVF core.

## Finding và tác động

Owner phải hỏi vì sao liên tục có tranche sửa, có lệch roadmap không, rồi yêu cầu ghi chỉ đạo vào handoff/memory và learning. Owner nhận định đây là lỗi orchestrator tự mất kiểm soát: người không viết code không thể biết bao giờ roadmap hoàn tất. Đây là phản hồi trực tiếp của owner trong phiên 2026-10-12, chưa có nghiên cứu độc lập về trải nghiệm người dùng.

Root nhận trách nhiệm điều phối: tập trung vào acceptance của từng scope nhỏ nhưng không duy trì đường đi hữu hạn tới kết quả sản phẩm. Các việc hỗ trợ có lý do kỹ thuật vẫn có thể cộng lại thành trì hoãn kéo dài. Người dùng phải phát hiện và điều chỉnh thay cho orchestrator. Điều này đi ngược mục tiêu điều phối có trách nhiệm của dự án: [roadmap](../../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md) ưu tiên hoàn thiện một luồng CSKH từ nguồn tới hành động có bằng chứng. Không quy lỗi chung cho Luna hoặc viện CVF như lý do cho mọi tranche mới.

## Nguồn và phần đã kiểm chứng

- Repository tại `edbecda1d5501a953c41386e3a19f95ce478496b`: R079–R082 chủ yếu intake/design/contract prerequisite; R083–R086 authoring/repair/execution/completion công cụ fixture. Nguồn: [R079 intake](../R079_RECOVERY_FENCE_PREREQUISITE_INTAKE_2026-10-11.md), [R081 next-scope](../R081_BOUNDED_BLOCKER_RESOLUTION_NEXT_SCOPE_2026-10-11.md), [R083 readiness](../R083_FIXTURE_PREREQUISITE_READINESS_2026-10-11.md).
- [Review R085](../R085_INDEPENDENT_LOCAL_FIXTURE_REVIEW_2026-10-12.md) giữ thất bại native trước start/SQL và cleanup; [review R086](../R086_INDEPENDENT_MOUNT_LIFECYCLE_REVIEW_2026-10-12.md) chỉ chấp nhận source/unit. `IMPLEMENTATION_STATUS.json` vẫn ghi fixtureRuntime NOT_RUN ở R086, fenceImplementation/integrationReadiness OPEN, R081 implementationReady false. Đây là kiểm tra bản ghi/source, không chạy lại native hay đo thời gian hoàn thành roadmap.
- Commit `062c067` ghi chỉ đạo roadmap vào memory/handoff/state. Gate local/range PASS7/7 và46 unit PASS kiểm tra repository, không đánh giá hiệu quả điều phối hay chứng minh đạt milestone sản phẩm.

## Nguyên nhân và trách nhiệm

Root xác định lỗi áp dụng nguyên tắc sẵn có: chia scope quá nhỏ và nhiều order không cho repair làm tăng successor/metadata overhead; thiếu checkpoint tổng thể sau prerequisite thất bại; báo test/tranche nhiều hơn khoảng cách tới acceptance sản phẩm. Review root cũng bỏ sót mount lifecycle trước R085. Đây là trách nhiệm ORCHESTRATOR/REVIEWER, khác với lỗi code/test cụ thể của worker.

AGENTS.md đã cho phép tiếp tục sửa cùng scope dưới authority không đổi, chỉ escalate ở ranh giới thật và có quy tắc repair round three. Learning này bổ sung cách phát hiện drift và báo cáo tiến độ tại dự án; chưa có machine gate thực thi nó. Không kết luận mọi prerequisite là thừa hoặc bỏ kiểm tra an toàn để tăng tốc.

## Kiểm soát phải áp dụng chủ động

Đọc record này khi resume, chọn scope, nhận failure hoặc đề xuất repair tiếp theo, và trước handback. ORCHESTRATOR chịu trách nhiệm, không chờ owner nhắc.

1. Trước dispatch, ghi stage roadmap, một khả năng sản phẩm kiểm chứng được, acceptance còn thiếu và chuỗi prerequisite hữu hạn tới khả năng đó. Mỗi prerequisite phải nêu vì sao chặn milestone, deliverable/budget/điều kiện dừng; không mở thêm chuỗi hỗ trợ chỉ vì có thể làm tốt hơn.
2. Trigger kiểm tra drift của dự án: sau hai scope hỗ trợ liên tiếp mà chưa đạt acceptance sản phẩm, hoặc ngay khi prerequisite thất bại, dừng tự động chọn scope kế tiếp và rà lại đường đi. Ghi chọn tiếp tục/gộp/thu hẹp/hoãn, lý do, dependency còn lại và giới hạn; tiếp tục việc đã được phép khi có route hữu hạn. Đây là yêu cầu điều phối tài liệu, không cấp thêm quyền hoặc yêu cầu owner xác nhận lại mỗi lần.
3. Dùng same-scope repair khi authority/budget cho phép. Successor chỉ khi có ranh giới thật hoặc hết quyền/ngân sách đã ghi, kèm lý do; không đặt zero-repair mặc định để tạo successor. Không đổi seed/budget lịch sử. Áp dụng REVIEW_COST_ESCALATION_REQUIRED ở round three khi không có root cause mới độc lập.
4. Mỗi handback báo bằng ngôn ngữ người không viết code hiểu được: đã dùng được việc gì; milestone nào còn thiếu; đang bị chặn bởi gì; số bước được lập kế hoạch tới milestone kế tiếp và điều kiện hoàn tất. Nếu chưa có route hữu hạn hoặc chưa đủ căn cứ ước lượng, nói rõ chưa biết và nêu việc cụ thể sẽ giải quyết bất định; không bịa ETA, phần trăm hoàn thành hoặc biến số test/tranche thành tiến độ.
5. Reviewer kiểm cả scope đạt contract và phạm vi đóng góp vào milestone. Bản review phải phân biệt source/unit, native integration, live/provider và deployed evidence. Theo dõi chất lượng worker phục vụ delivery; root scope/priority/review misses được ghi riêng. Gate xanh không thay cho kiểm tra roadmap.

## Áp dụng và disposition

Project: RECORDED_AND_LINKED / EFFECTIVENESS_NOT_YET_VERIFIED. Memory/handoff/state có trigger đọc learning; registry/index giúp agent khác tìm được. Việc ghi tài liệu chưa chứng minh agent sẽ tự tuân thủ hoặc roadmap đã trở lại đúng nhịp. R086 FREEZE OPEN và55 deadlinks vẫn giữ; fixture/fence/recovery gaps chưa đóng. Lần chọn scope tới phải trình bày milestone, route hữu hạn và stop condition; lần handback tới phải báo theo mục4. Chỉ đánh giá cải thiện bằng các lần thực hiện tiếp theo, không tự đóng finding bằng PASS gate tài liệu.

Kiểm tra publication tài liệu: doctor PASS WITH NOTE25/1 (stale pin/GATE_PROFILE_NOT_INSTALLED giữ nguyên); local và cached origin/main..HEAD preflight PASS7/7;46 gate unit tests PASS42.932s; catalog Windows PowerShell qua preflight và PowerShell7 riêng PASS; git diff --check PASS. Không chạy lại docs build đang có55 deadlinks, không có runtime/provider/push. Các kết quả này chỉ kiểm tra repository/tài liệu, không đánh giá hiệu quả kiểm soát roadmap.

Upstream: PROPOSED_FOR_CVF_INTAKE / NOT_SENT / NOT_ACCEPTED. Đề xuất CVF cha xem xét checkpoint tiến độ tổng thể và mẫu handback cho owner không viết code, bổ sung cho gate cấp tranche; record này là packet nguồn. Chưa gửi thông điệp, sửa core hay khẳng định CVF cha tiếp nhận. Root ORCHESTRATOR chuẩn bị transfer khi có scope riêng được phép. Learning mô tả lỗi điều phối của dự án, không khẳng định CVF đã kiểm soát hành vi AI; không có provider call hoặc live governance proof trong lần cập nhật này.

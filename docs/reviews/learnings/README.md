# Kho finding và learning dùng chung của dự án

Quy ước được owner thống nhất ngày 2026-10-02. Đường dẫn chính thức: `docs/reviews/learnings/`.

Thư mục này lưu các finding, vấn đề gặp trong quá trình làm việc và bài học có thể tái sử dụng, với hai mục đích song song:

1. **Dự án học và áp dụng ngay:** giúp mọi agent nhận biết lỗi đã gặp, tránh lặp lại và cải thiện cách thực hiện, kiểm thử, review hoặc ghi evidence trong phạm vi được giao.
2. **Upstream lên CVF cha:** cung cấp use case, nguồn evidence và đề xuất cải tiến để CVF cha đánh giá, khử trùng lặp và chuyển thành quy tắc, template, tooling hoặc gate dùng chung khi phù hợp.

Đây là nguồn learning chung được lưu trong Git. Memory riêng của Claude, Codex hoặc provider khác là nguồn bổ sung; finding có bài học tái sử dụng phải được đưa vào đây và có pointer để agent khác tìm thấy. Khi gặp lại cùng cơ chế lỗi, cập nhật record hiện có và thêm nguồn evidence; tạo record mới khi có vấn đề hoặc nguyên nhân riêng. Không cần chờ upstream xử lý mới áp dụng bài học tại dự án.

## Ghi gì trong mỗi record

Mỗi record cần ghi đủ thông tin để agent chưa tham gia phiên trước hiểu và hành động được:

- Finding: thời điểm, bối cảnh/trigger, hành vi thực tế và hành vi mong đợi.
- Nguồn: project, tranche/commit, đường dẫn evidence hoặc lệnh tái hiện đã khử bí mật. Phân biệt owner-reported, worker-reported và independently verified; thông tin chưa xác minh vẫn lưu được với nhãn rõ ràng.
- Nguyên nhân: cơ chế đã xác định, hoặc giả thuyết và phần còn chưa giải thích; phân biệt lỗi áp dụng quy tắc có sẵn với thiếu quy tắc, tooling, test hoặc gate.
- Bài học áp dụng tại dự án: bước phòng tránh/sửa, lúc phải đọc lại, người/vai trò chịu trách nhiệm, kết quả kiểm tra và việc còn mở.
- Đề xuất upstream: phần có thể tổng quát hóa, control cần xem xét, nguồn liên quan và agent/owner tiếp nhận; nếu chỉ liên quan dự án thì ghi lý do không đề xuất upstream.
- Disposition và giới hạn: trạng thái xử lý tại dự án và trạng thái upstream riêng biệt; ngày cập nhật, evidence, quyết định và next move cho từng phần.

Ví dụ: một lỗi đã sửa và kiểm tra tại dự án có thể vẫn chờ CVF cha đánh giá. Ghi riêng “project: đã áp dụng, kiểm chứng tại …” và “upstream: chờ đánh giá, packet …”; không dùng một nhãn “đã đóng” cho cả hai. Chỉ ghi upstream đã tiếp nhận/triển khai khi có quyết định và evidence từ CVF cha. Nếu CVF cha hoãn hoặc từ chối, lưu lý do và pointer trả về.

Giữ record ngắn và có liên kết đến evidence gốc; không sao chép toàn bộ log, lịch sử phiên hoặc dữ liệu khách hàng. Không ghi API key, token, nội dung `.env` hoặc bí mật provider. Lỗi chưa giải thích vẫn giữ là chưa giải thích; kiểm tra không chạy ghi NOT RUN, không đổi thành PASS bằng các lần chạy lại thành công.

## Cách sử dụng và cập nhật

Khi INTAKE, BUILD/REPAIR hoặc REVIEW gặp finding có bài học tái sử dụng, agent phụ trách ghi/cập nhật record trong cùng lượt xử lý và liên kết từ handoff hoặc evidence liên quan. Trước công việc tương tự, đọc record theo trigger của nó; không cần đọc lại toàn bộ thư mục hay lịch sử.

SESSION_SYNC_STEWARD cập nhật pointer trong `CVF_SESSION_MEMORY.md` và active handoff. Record cần discovery trực tiếp được đăng ký trong `docs/catalog/ARTIFACT_REGISTRY.json`, rồi tái sinh `docs/INDEX.md` bằng catalog manager; không sửa chỉ mục sinh tự động riêng lẻ. Đồng bộ `IMPLEMENTATION_STATUS.json` khi trạng thái hoặc evidence thay đổi, chạy các kiểm tra bắt buộc và commit các artifact trong phạm vi được phép.

ORCHESTRATOR đánh giá khả năng tổng quát hóa và chuẩn bị packet upstream. Agent/owner của CVF cha quyết định tiếp nhận, gộp vào learning hiện có, hoãn hoặc từ chối; nếu triển khai, liên kết work order và evidence kiểm chứng của cha trở lại record nguồn. Lưu packet ở đây là chuẩn bị chuyển giao, không đồng nghĩa đã gửi hay đã được CVF cha chấp nhận.

Quy ước này hướng dẫn ghi và dùng learning; việc lưu record không tự cấp quyền sửa source/tooling, mở rộng work order, chỉnh CVF cha hoặc đóng tranche. Mọi hành động tiếp theo vẫn theo authority, phase, risk và review của dự án. Chưa có machine gate mới thực thi quy ước này; tài liệu không chứng minh CVF điều khiển AI runtime.

## Record hiện có

- [Downstream gate learning intake cho CVF cha](CCMAI_TO_CVF_DOWNSTREAM_GATE_LEARNING_INTAKE_2026-10-01.md)
- [Shell cleanup và chuyển đổi đường dẫn MSYS](feedback_shell_cleanup_and_paths.md)
- [Repair workflow, continuity, mutation và evidence](feedback_cvf_repair_workflow.md)

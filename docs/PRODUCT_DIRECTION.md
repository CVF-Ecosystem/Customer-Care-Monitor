# Định hướng Customer Care Monitor AI

## Người dùng và ranh giới

Một bản cài đặt phục vụ một công ty hoặc một cá nhân. Chủ workspace cấu hình kênh chăm sóc khách hàng, tiêu chí QC, phân loại, AI provider và người cùng làm việc. Ứng dụng không bán/điều phối nhiều khách hàng độc lập trên cùng một database.

Nguồn dữ liệu đầu vào là hội thoại từ các kênh. Bản ghi tin nhắn và phản hồi AI cần được lưu đủ để truy vết. Điểm QC, nhãn phân loại và cảnh báo AI là đề xuất hỗ trợ kiểm tra; các quyết định có tác động đến nhân viên/khách hàng cần người có quyền xác nhận theo chính sách của tổ chức.

Mục tiêu phát triển được owner chấp nhận: giúp người phụ trách biết hội thoại nào cần can thiệp, vì sao, ai xử lý và đã giải quyết đến đâu. Hoàn thiện CSKH trước; sau nghiệm thu và vận hành, dùng bằng chứng của dự án để nhân rộng sang các project khác và cân nhắc nâng nền CVF theo work order riêng.

Bộ lọc SoT và phương pháp học từ skill Jev hỗ trợ chọn dữ kiện, phân loại xác định và điều phối; AI/LLM tiếp tục phân tích ngữ nghĩa, tạo nhận xét và gợi ý phản hồi. Chất lượng/rủi ro khách hàng là điều kiện bắt buộc trước tối ưu chi phí. Ca tiếng Việt mơ hồ cần đủ ngữ cảnh và model/người phù hợp; không tự bỏ để giảm call. Gợi ý cho người phụ trách không mặc nhiên cấp quyền tự gửi trả lời tới khách hàng.

SoT có thẩm quyền theo loại phát biểu: tin nhắn ghi nhận lời nói, nguồn giao dịch xác nhận giao dịch. Người duyệt chấp nhận đánh giá; suy luận vẫn khác sự kiện quan sát được và khác trạng thái xử lý vụ việc.

Runtime tiếp tục dùng MySQL 8. Fresh install dùng schema mặc định `CCMA`; cấu hình `DB_NAME` vẫn cho phép bản cài cũ giữ `cqa`. Luồng lọc đặt SQL candidate selection trước, snapshot/policy/rule/admission trong Go, rồi mới resolve provider cho item cần LLM. Đặc tả quản trị nằm tại `docs/specs/DATABASE_FILTER_PIPELINE_2026-09-27.md` trong source repo.

## Áp dụng pattern CVF và Shift Operations Workspace

| Pattern | Áp dụng cho chăm sóc khách hàng | Trạng thái |
|---|---|---|
| Một phạm vi nghiệp vụ rõ ràng | Một workspace được tạo khi Setup; API không cho tạo/xóa workspace, UI không có chuyển workspace | Đã có trong bản fork |
| Kiểm tra đầu ra AI trước khi thành dữ liệu | Kiểm tra schema, giá trị, bằng chứng và liên kết batch với đúng hội thoại | Đã có kiểm tra cục bộ; chưa có hợp đồng version hóa |
| Ghi dữ liệu nguyên tử và trạng thái lỗi trung thực | Kết quả một hội thoại dùng transaction; lượt chạy lỗi không báo thành công và không tiến mốc quét | Đã có cho analyzer; cần mở rộng sang đồng bộ và thông báo |
| Evidence → proposal → review → confirmed | Liên kết từng nhận xét/vi phạm với đoạn hội thoại, tạo hàng đợi duyệt, lưu người và thời điểm duyệt | Chưa triển khai |
| Risk, approval, refusal | Quy tắc riêng cho khiếu nại, dữ liệu nhạy cảm, đánh giá nhân viên và cảnh báo ra ngoài; chặn tự động khi thiếu evidence | Chưa triển khai |
| Audit và correction | Nhật ký bất biến cho phê duyệt, sửa kết quả và thay đổi rule; giữ phiên bản trước/sau | Nhật ký hoạt động CQA hiện có; chưa đủ hợp đồng audit CVF |
| Chi phí và provider boundary | Giới hạn ngân sách theo job, theo dõi provider/model, từ chối khi vượt hạn mức | Có log chi phí; chưa có gate ngân sách |

Pattern tham chiếu từ tài liệu `shift-operations-workspace/docs/cvf/` trong workspace phát triển, nhất là `EVIDENCE_AND_TRUTH.md`, `RISK_AND_APPROVAL.md` và `CVF_CONTROL_MAPPING.md`. Các quy tắc nghiệp vụ cảng/ca trực không được sao chép sang CSKH.

## Thứ tự nâng cấp tiếp theo

1. Chọn một kênh/use case pilot, xác định tiêu chí chất lượng, ca rủi ro và corpus tiếng Việt. Đo thời gian tới hành động, backlog review và chi phí; baseline provider thật cần work order và admission phù hợp.
2. Sửa đồng bộ/checkpoint khi lỗi; bảo toàn ngày giờ, vai trò, nguồn/version và coverage của tin sửa/xóa/ảnh/file. Hợp đồng kết quả phải đối chiếu evidence với snapshot; bỏ giả định confidence bằng 1 khi chưa đo.
3. Xây một luồng hoàn chỉnh gồm bộ lọc nội bộ học từ Jev, admission quyền/dữ liệu/ngân sách, AI/LLM, audit tối thiểu và hàng đợi review/giao xử lý. Tách đủ điều kiện, cách xử lý, trạng thái duyệt và trạng thái hành động; chờ có owner/deadline. Real-provider proof và independent review cần cho claim governance runtime.
4. Tối ưu sau khi có vòng phản hồi: preview tác động rule, kiểm mẫu các ca bị bỏ qua, tái dùng kết quả còn hiệu lực và giữ đủ ngữ cảnh khi chỉ phân tích phần thay đổi. Kiểm chất lượng trước chi phí; không ép giảm số call cần thiết.
5. Pilot có kiểm soát, hoàn thiện outbox/retry, quyền, backup/restore và phát hành có bằng chứng. Sau nghiệm thu CSKH mới thử chuyển mẫu sang dự án thứ hai, rồi đề xuất chia sẻ module hoặc nâng nền CVF.

Chi tiết và phụ thuộc S0–S7 nằm tại `docs/roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md`; hợp đồng thiết kế nằm tại `docs/decisions/SOT_FIRST_DATA_FILTERING_PATTERN_2026-09-27.md` trong source repo. Các mục nâng cấp trên chưa được triển khai; owner chấp nhận định hướng không thay thế independent R2 review.

## Giới hạn kiểm chứng hiện tại

Kiểm tra unit cho bộ kiểm tra phản hồi AI chứng minh các nhánh từ chối trên dữ liệu giả lập. Nó không chứng minh chất lượng nhận xét, tính đúng của bằng chứng trong hội thoại thật, hay luồng phê duyệt CVF. Chưa cấu hình API key/provider thật cho bản fork này.

`npm audit --omit=dev` trên dependency lock kế thừa CQA hiện báo 8 vấn đề (7 high, 1 moderate). Cần nâng phiên bản và kiểm thử lại trước khi triển khai thực tế; bản fork này chưa xử lý dependency audit.

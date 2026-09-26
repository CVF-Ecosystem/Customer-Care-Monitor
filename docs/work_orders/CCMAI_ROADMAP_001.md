# CCMAI-ROADMAP-001: kế hoạch kiểm soát AI runtime và chi phí

Trạng thái: BUILD tài liệu hoàn tất, chờ REVIEW độc lập. Rủi ro R2 vì kế hoạch định hướng xử lý dữ liệu hội thoại và dịch vụ AI bên ngoài; đợt này chỉ sửa tài liệu.

## INTAKE

Owner yêu cầu lập roadmap cho Customer Care Monitor AI dựa trên những phần tốt của CQA, CVF và Shift; xác minh ý tưởng lọc/phân loại trước LLM và tham khảo mẫu thiết kế trong skill của TypeSafe/Jev. Đồng thời owner yêu cầu sửa một nhận định sai trong roadmap Shift; phần sửa đó theo handoff và kiểm tra riêng của Shift.

## DESIGN

Đặt Source of Truth (SoT) gồm dữ liệu hội thoại đã lưu, provenance, trạng thái xác nhận, chính sách/phiên bản quy tắc và quyền làm căn cứ cho gate tại máy. Gate xác định phân loại trước, dùng `NO_AI`/`RULES_ONLY` khi đủ bằng chứng, chuyển người khi mơ hồ hoặc rủi ro; chỉ gọi Claude/Gemini/OpenAI/xAI khi tác vụ cần phân tích ngữ nghĩa sâu hoặc tạo nhận xét và đã qua quyền, chính sách dữ liệu, ngân sách. Học từ skill TypeSafe/Jev cách chia quyết định thành câu hỏi hẹp, kết quả có kiểu, nhánh `unknown` và kiểm soát độ chắc chắn; thiết kế lại thành hợp đồng nội bộ, không thêm Jev API/SDK hay một dịch vụ trung gian. Kết quả AI là đề xuất cần kiểm tra và có thể cần người duyệt.

## SPEC / WORK_ORDER

- Tạo một roadmap có hiện trạng đối chiếu mã, các tranche theo phụ thuộc, tiêu chí đạt và điều kiện dừng; dẫn nguồn skill TypeSafe/Jev như tham khảo phương pháp, không như dependency triển khai.
- Sửa `docs/roadmaps/README.md` để dẫn roadmap, đồng bộ handoff, active state và implementation status. Không đổi mã chạy, dữ liệu, secret, provider configuration hoặc phát hành.
- Bằng chứng BUILD: đối chiếu các nhận định hiện trạng với source, kiểm tra link/cú pháp Markdown, chạy catalog check và workspace doctor. Không dùng mock hay kiểm tra tĩnh để khẳng định CVF governance runtime.
- REVIEW độc lập kiểm claim boundary, tính ưu tiên SoT/local gate, thứ tự phụ thuộc, trường hợp false negative và phép tính chi phí Agent/AI/LLM thật sự được gọi. Không FREEZE khi thiếu review hoặc bằng chứng.

Role route: ORCHESTRATOR -> SPEC_AUTHOR -> WORK_ORDER_AUTHOR -> IMPLEMENTATION_WORKER (roadmap và continuity) -> COMMIT_STEWARD -> SESSION_SYNC_STEWARD -> ORCHESTRATOR; REVIEWER độc lập vẫn chờ.

Mỗi tranche triển khai trong roadmap cần work order riêng, phân loại R2 trở lên khi dùng dữ liệu thật hoặc provider/network, và bằng chứng provider thật cho mọi claim CVF kiểm soát hành vi AI ở runtime.

## Owner correction trong REVIEW

Chỉ đạo mới thay thế phương án Jev-as-a-service trong bản BUILD đầu (`5a994d1`). Phạm vi sửa vẫn là roadmap, work order và continuity hiện có; không mở tích hợp Jev, không thay source ứng dụng. Bản sửa tiếp tục chờ independent R2 review.

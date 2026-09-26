# CCMAI-ROADMAP-001: kế hoạch kiểm soát AI runtime và chi phí

Trạng thái: BUILD tài liệu hoàn tất, chờ REVIEW độc lập. Rủi ro R2 vì kế hoạch định hướng xử lý dữ liệu hội thoại và dịch vụ AI bên ngoài; đợt này chỉ sửa tài liệu.

## INTAKE

Owner yêu cầu lập roadmap cho Customer Care Monitor AI dựa trên những phần tốt của CQA, CVF và Shift; xác minh ý tưởng lọc/phân loại trước LLM và đánh giá Jev của TypeSafe. Đồng thời owner yêu cầu sửa một nhận định sai trong roadmap Shift; phần sửa đó theo handoff và kiểm tra riêng của Shift.

## DESIGN

Giữ hội thoại và bằng chứng gốc làm nguồn sự thật. Đặt quy tắc máy xác định trước mọi dịch vụ AI; thử Jev như tầng phân loại ngữ nghĩa tùy chọn, có đo chất lượng và tổng chi phí. Chỉ chuyển cho Claude/Gemini/OpenAI/xAI khi tác vụ thật sự cần tạo nhận xét hoặc phân tích sâu, sau khi qua quyền, chính sách dữ liệu và ngân sách. Kết quả AI là đề xuất cần kiểm tra và có thể cần người duyệt.

## SPEC / WORK_ORDER

- Tạo một roadmap có hiện trạng đối chiếu mã, các tranche theo phụ thuộc, tiêu chí đạt và điều kiện dừng; dẫn nguồn chính thức cho Jev.
- Sửa `docs/roadmaps/README.md` để dẫn roadmap, đồng bộ handoff, active state và implementation status. Không đổi mã chạy, dữ liệu, secret, provider configuration hoặc phát hành.
- Bằng chứng BUILD: đối chiếu các nhận định hiện trạng với source, kiểm tra link/cú pháp Markdown, chạy catalog check và workspace doctor. Không dùng mock hay kiểm tra tĩnh để khẳng định CVF governance runtime.
- REVIEW độc lập kiểm claim boundary, thứ tự phụ thuộc, trường hợp false negative và phép tính chi phí Jev + LLM. Không FREEZE khi thiếu review hoặc bằng chứng.

Role route: ORCHESTRATOR -> SPEC_AUTHOR -> WORK_ORDER_AUTHOR -> IMPLEMENTATION_WORKER (roadmap và continuity) -> COMMIT_STEWARD -> SESSION_SYNC_STEWARD -> ORCHESTRATOR; REVIEWER độc lập vẫn chờ.

Mỗi tranche triển khai trong roadmap cần work order riêng, phân loại R2 trở lên khi dùng dữ liệu thật hoặc provider/network, và bằng chứng provider thật cho mọi claim CVF kiểm soát hành vi AI ở runtime.

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

## Owner amendment trong REVIEW: mẫu tái sử dụng và quy tắc CVF

Owner xác nhận phương pháp lọc dữ liệu SoT-first là use case có thể áp dụng cho nhiều dự án tương tự và yêu cầu áp dụng các quy tắc tốt của CVF khi phát triển ứng dụng này. Đây là sửa đổi DESIGN/SPEC trong cùng mục tiêu lập roadmap; phạm vi tài liệu mở thêm `docs/decisions/SOT_FIRST_DATA_FILTERING_PATTERN_2026-09-27.md` và `docs/decisions/README.md`, cùng roadmap, work order và continuity. Không sửa CVF core hoặc mã runtime.

Acceptance của đợt sửa: quyết định kiến trúc phân biệt evidence, dữ liệu chuẩn hóa, fact được xác nhận và policy authority; định nghĩa hợp đồng quyết định có kiểu, nhánh local/AI/human, dấu vết nguồn và quy tắc; chỉ ra phần lõi có thể tái dùng và phần cấu hình riêng từng dự án; ánh xạ các phase, risk, role, review và bằng chứng CVF vào việc phát triển ứng dụng. Roadmap phải dẫn quyết định này và giữ đúng ranh giới: mẫu thiết kế chưa phải thư viện dùng chung, CVF SOT3 chưa được tích hợp vào ứng dụng, và chưa có proof provider thật cho runtime gate. Kiểm `git diff --check`, catalog và workspace doctor; độc lập R2 review trước FREEZE.

## Owner chấp nhận phản biện: CSKH trước, chất lượng trước chi phí

Chỉ đạo hiện hành bổ sung acceptance: hoàn thiện một luồng CSKH có dữ liệu đáng tin, hàng đợi can thiệp, người review và kết quả xử lý trước khi nhân rộng sang dự án khác hoặc đề xuất nâng nền CVF. Bộ lọc nội bộ học từ skill/cách xử lý Jev kết hợp với AI/LLM phân tích ngữ nghĩa và tạo nhận xét/phản hồi; vẫn giữ định hướng không thêm Jev như dịch vụ trung gian. Tiết kiệm phải vượt cổng chất lượng/rủi ro khách hàng, kiểm trên ngữ cảnh tiếng Việt và kiểm mẫu ở cả nhánh bị bỏ qua. Ngân sách thiếu không được biến thành PASS hoặc bỏ việc âm thầm.

Phản biện tư vấn được owner chấp nhận làm đầu vào thiết kế, chưa phải independent R2 review. Sửa các điểm: observed fact khác inference/accepted assessment; gate tách eligibility, execution và disposition; có nhánh chờ dữ liệu; đưa review tối thiểu vào luồng đầu tiên; bổ sung preview rule và tái dùng kết quả có điều kiện. Các phát hiện sync/transcript/confidence từ source là backlog cần xác minh khi BUILD, chưa phải lỗi đã tái hiện bằng runtime test.

Allowed paths bổ sung `docs/PRODUCT_DIRECTION.md` và các index roadmap/decision hiện có. Mọi thay đổi đợt này chỉ là tài liệu và continuity/status; không cấp quyền tích hợp provider, gửi trả lời trực tiếp cho khách hàng hay triển khai runtime. Kiểm catalog, doctor, diff và sự nhất quán giữa các tài liệu. Giữ REVIEW_PENDING tới review độc lập.

## Owner amendment trong REVIEW: F01–F08 và quyền test (2026-09-30)

Owner chấp nhận tám phát hiện F01–F08 từ báo cáo `7481196` sau đối chiếu độc lập với HEAD local `698612f`, yêu cầu đưa chúng vào roadmap hiện có. Phạm vi sửa của amendment này là roadmap, một review source-level, work order lập kế hoạch và continuity/status; không sửa mã sản phẩm, chạy provider, triển khai hoặc tác động dữ liệu trong đợt tài liệu. Review phải phân biệt cơ chế mã với sự cố đã xảy ra và giữ các finding OPEN cho tới khi có repair/review riêng.

Owner xác nhận dữ liệu hiện tại là dữ liệu thử và cho phép dùng Alibaba API key **khi cần** để lấy bằng chứng/test về sau. Sự cho phép này tiếp tục áp dụng cho các work order cùng dự án/phạm vi test, không cần hỏi lại cho cùng ranh giới; mỗi work order ghi rõ provider/model, dữ liệu gửi, số call hoặc trần chi phí, evidence đã làm sạch và rollback/cleanup. Không lưu giá trị key trong Git, tài liệu, log hoặc receipt. Nếu dữ liệu được nhập mới/không còn là dữ liệu thử, hoặc thay đổi mục đích, đường tác động bên ngoài, secret, phát hành hay trần R2, phải đánh giá lại ranh giới. Mọi claim CVF governance runtime vẫn cần API call thật và request/response đã làm sạch theo AGENTS.md; quyền dùng key không tự tạo proof.

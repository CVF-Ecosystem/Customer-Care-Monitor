# Định hướng Customer Care Monitor AI

## Người dùng và ranh giới

Một bản cài đặt phục vụ một công ty hoặc một cá nhân. Chủ workspace cấu hình kênh chăm sóc khách hàng, tiêu chí QC, phân loại, AI provider và người cùng làm việc. Ứng dụng không bán/điều phối nhiều khách hàng độc lập trên cùng một database.

Nguồn dữ liệu đầu vào là hội thoại từ các kênh. Bản ghi tin nhắn và phản hồi AI cần được lưu đủ để truy vết. Điểm QC, nhãn phân loại và cảnh báo AI là đề xuất hỗ trợ kiểm tra; các quyết định có tác động đến nhân viên/khách hàng cần người có quyền xác nhận theo chính sách của tổ chức.

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

1. Định nghĩa hợp đồng kết quả QC/classification có version; lưu `conversation_id`, ID tin nhắn, trích dẫn, model và phiên bản rule. Thêm kiểm tra evidence đối chiếu tin nhắn gốc.
2. Thêm trạng thái `proposed`, `confirmed`, `rejected`, `corrected` cho đánh giá; hàng đợi duyệt và chính sách ai được xác nhận. Dashboard/tin báo phải phân biệt rõ đề xuất với kết quả đã xác nhận.
3. Chuẩn hóa audit append-only cho thay đổi rule, phê duyệt, sửa đánh giá và gửi thông báo. Có khóa idempotency để retry không nhân đôi tác động.
4. Sửa đồng bộ kênh và gửi thông báo để checkpoint/trạng thái chỉ tiến khi tác vụ tương ứng thực sự thành công; thêm retry có giới hạn và quan sát lỗi.
5. Đặt gate ngân sách, quyền và dữ liệu nhạy cảm trước khi gọi AI hoặc gửi dữ liệu ra kênh ngoài. Kiểm chứng với provider thật trước khi tuyên bố CVF governance hoạt động end-to-end.

## Giới hạn kiểm chứng hiện tại

Kiểm tra unit cho bộ kiểm tra phản hồi AI chứng minh các nhánh từ chối trên dữ liệu giả lập. Nó không chứng minh chất lượng nhận xét, tính đúng của bằng chứng trong hội thoại thật, hay luồng phê duyệt CVF. Chưa cấu hình API key/provider thật cho bản fork này.

`npm audit --omit=dev` trên dependency lock kế thừa CQA hiện báo 8 vấn đề (7 high, 1 moderate). Cần nâng phiên bản và kiểm thử lại trước khi triển khai thực tế; bản fork này chưa xử lý dependency audit.

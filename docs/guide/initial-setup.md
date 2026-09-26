# Thiết lập lần đầu

Sau khi [chạy ứng dụng](/guide/installation), mở `http://localhost:8088` hoặc URL đã cấu hình. Trang Setup chỉ xuất hiện khi database chưa có người dùng.

1. Nhập **tên công ty hoặc cá nhân** (ít nhất 2 ký tự), email admin, tên hiển thị và mật khẩu.
2. Xác nhận mật khẩu, rồi chọn **Tạo tài khoản**.
3. Ứng dụng tạo admin, workspace duy nhất và liên kết Owner trong cùng một transaction; sau đó chuyển vào ứng dụng.
4. Vào **Cài đặt** để cấu hình thông tin chung, AI provider/model/API key; thử kết nối trước khi tạo công việc.
5. Kết nối kênh chat, đồng bộ tin nhắn, tạo công việc và kiểm tra kết quả.

Không có bước **Thêm công ty** sau Setup. Luồng tạo, chuyển và xóa công ty của CQA gốc không áp dụng cho bản fork này. Các đường dẫn `/tenants/:tenantId` còn trong API để định phạm vi dữ liệu nội bộ.

Kết quả AI hiện là đề xuất. Người có trách nhiệm cần kiểm tra nội dung hội thoại và bằng chứng trước khi dùng kết quả để đánh giá nhân viên hoặc ra quyết định với khách hàng. Bản fork chưa có quy trình xác nhận kết quả trong ứng dụng.

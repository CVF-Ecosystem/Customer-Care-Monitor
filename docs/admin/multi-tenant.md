# Phạm vi một workspace

Customer Care Monitor AI phục vụ **một công ty hoặc cá nhân trên mỗi bản cài đặt**. Trang Setup tạo workspace duy nhất cùng tài khoản Owner đầu tiên. Có thể thêm người dùng và phân quyền trong workspace này.

Các chức năng tạo, chuyển hoặc xóa công ty của CQA gốc đã bị khóa. `POST /api/v1/tenants` và `DELETE /api/v1/tenants/:tenantId` trả lỗi; các cột `tenant_id` và đường dẫn `/tenants/:tenantId` còn lại để giữ tương thích nội bộ và kiểm tra phạm vi truy cập.

Nếu database nhập từ CQA chứa nhiều workspace, ứng dụng từ chối khởi động. Không xóa một công ty trong database để ép chạy: cần kiểm kê dữ liệu, sao lưu đã thử phục hồi và kế hoạch chuyển đổi riêng.

Xem [người dùng và phân quyền](/admin/users) cho các vai trò trong workspace. Trang này thay thế hướng dẫn đa công ty của CQA gốc.

# Cài đặt từ source

Hiện chưa có script cài đặt hoặc Docker image chính thức. Hãy build Customer Care Monitor AI từ source của repo này.

## Chuẩn bị

- Docker Engine và Docker Compose v2, hoặc Docker Desktop trên máy cá nhân.
- Một thư mục trống cho repo và đủ dung lượng cho MySQL, file đính kèm, log và bản sao lưu. Chưa có đo kiểm tải để xác nhận cấu hình CPU/RAM tối thiểu.
- API key của provider AI chỉ cần khi bắt đầu dùng công việc phân tích.

## Khởi chạy

```bash
git clone https://github.com/CVF-Ecosystem/Customer-Care-Monitor-AI.git
cd Customer-Care-Monitor-AI
cp .env.example .env
```

Trong `.env`, đặt `DB_PASSWORD`, `MYSQL_ROOT_PASSWORD`, `JWT_SECRET` (ít nhất 32 ký tự) và `ENCRYPTION_KEY` (32 byte; `openssl rand -hex 16` tạo 32 ký tự ASCII). Trên máy dùng PowerShell, sao chép bằng `Copy-Item .env.example .env`. Không commit `.env`.

Bản cài mới dùng database MySQL tên `CCMA`. Nếu đang nâng cấp một bản cài cũ có dữ liệu trong schema `cqa`, giữ nguyên `DB_NAME=cqa` trong `.env`. Chỉ đổi tên sau khi có work order migration, bản sao lưu đã phục hồi thử và rollback; thay `DB_NAME` không tự chuyển dữ liệu, còn biến khởi tạo MySQL chỉ có hiệu lực khi volume được tạo lần đầu.

```bash
docker compose up -d --build
docker compose ps
docker compose logs --tail=100 app nginx db
```

Mặc định truy cập `http://localhost:8088`. Cổng host lấy từ `HTTP_PORT` trong `.env`; `HTTPS_PORT` mặc định là `8443`. Nếu dùng Let's Encrypt, xem [Tên miền và SSL](/guide/domain-ssl) trước khi mở ra Internet.

Trang Setup tạo **tên công ty/cá nhân, admin và workspace trong cùng một bước**. Xem [Thiết lập lần đầu](/guide/initial-setup).

## Lưu ý dữ liệu cũ

Ứng dụng chỉ chấp nhận tối đa một workspace trong database. Nếu nhập database có nhiều công ty, ứng dụng sẽ dừng khởi động thay vì tự chọn một công ty và che dữ liệu còn lại. Cần kế hoạch chuyển đổi riêng và sao lưu đã kiểm chứng trước khi nhập dữ liệu cũ.

`docker compose down` dừng dịch vụ và giữ volume. `docker compose down -v` xóa các volume dữ liệu; chỉ dùng khi chủ động hủy bản cài đặt.

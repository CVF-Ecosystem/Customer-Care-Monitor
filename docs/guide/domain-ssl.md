# Tên miền và HTTPS

Compose của bản fork ánh xạ HTTP ra cổng host `HTTP_PORT` (mặc định `8088`) và HTTPS ra `HTTPS_PORT` (mặc định `8443`). Nginx trong container nghe cổng 80 và 443. Vì vậy `http://localhost` hoặc `https://domain` không tự hoạt động trên cổng chuẩn nếu chưa đổi hai biến này.

Để dùng Let's Encrypt với một domain công khai, trỏ DNS A/AAAA tới máy chủ, bảo đảm cổng 80 và 443 từ Internet đi tới host, rồi đặt trong `.env`:

```env
HTTP_PORT=80
HTTPS_PORT=443
LEGO_DOMAIN=care.example.com
LEGO_EMAIL=admin@example.com
```

Sau đó chạy `docker compose up -d --build`, xem `docker compose logs --tail=100 nginx` và kiểm tra chứng chỉ qua trình duyệt. Nếu có reverse proxy hoặc hệ thống cấp chứng chỉ riêng, để `LEGO_DOMAIN` trống và cấu hình proxy theo kiến trúc của bạn.

Trong môi trường local không có domain, để trống `LEGO_DOMAIN` và dùng `http://localhost:8088`. Kết nối Facebook hoặc MCP công khai có thể cần HTTPS và URL truy cập được từ bên ngoài; kiểm tra yêu cầu của từng tích hợp trước khi dùng.

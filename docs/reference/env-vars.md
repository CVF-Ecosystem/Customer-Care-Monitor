# Biến môi trường

Tham khảo cho bản build từ source; đối chiếu `.env.example`, `docker-compose.yml` và `backend/config/config.go` khi triển khai. Trang này kế thừa CQA và chưa được xác minh hết các biến tùy chọn.

| Biến Compose | Mặc định | Ý nghĩa |
|---|---|---|
| `HTTP_PORT` | `8088` | Cổng HTTP trên host |
| `HTTPS_PORT` | `8443` | Cổng HTTPS trên host |
| `COMPOSE_PROJECT_NAME` | `ccma` | Tên ổn định cho container, network và volume Compose |

## Bắt buộc

| Biến | Mô tả | Ví dụ |
|------|-------|-------|
| `DB_PASSWORD` | Mật khẩu MySQL cho user `ccma` | `openssl rand -hex 16` |
| `MYSQL_ROOT_PASSWORD` | Mật khẩu root MySQL | `openssl rand -hex 16` |
| `JWT_SECRET` | Secret cho JWT tokens, tối thiểu 32 ký tự | `openssl rand -hex 32` |
| `ENCRYPTION_KEY` | Key 32 bytes cho mã hóa AES-256-GCM | `openssl rand -hex 16` |

## Server

| Biến | Mô tả | Mặc định |
|------|-------|----------|
| `SERVER_PORT` | Port của ứng dụng | `8080` |
| `SERVER_HOST` | Host bind | `0.0.0.0` |
| `APP_ENV` | Môi trường (`development` / `production`) | `production` |
| `APP_URL` | URL công khai (cho links trong notification) | |

## Database

| Biến | Mô tả | Mặc định |
|------|-------|----------|
| `DB_HOST` | MySQL host | `db` |
| `DB_PORT` | MySQL port | `3306` |
| `DB_USER` | MySQL username | `ccma` |
| `DB_PASSWORD` | MySQL password | |
| `DB_NAME` | Tên database; fresh install dùng `CCMA`, bản cài cũ giữ tên schema hiện tại | `CCMA` |

Workspace phát triển của project khởi tạo mới với database `CCMA` và user `ccma`. `DB_NAME` và `DB_USER` vẫn có thể được đặt rõ khi triển khai vào một môi trường khác; đổi các biến này không tự đổi tên hoặc sao chép dữ liệu trong volume MySQL.

## Rate Limiting

| Biến | Mô tả | Mặc định |
|------|-------|----------|
| `RATE_LIMIT_PER_IP` | Số request/phút cho mỗi IP | `500` |
| `RATE_LIMIT_PER_USER` | Số request/phút cho mỗi user | `1000` |

## Lưu file đính kèm (tùy chọn)

| Biến | Mô tả | Mặc định |
|------|-------|----------|
| `STORAGE_LOCAL_DIR` | Thư mục cất file đính kèm trên máy chủ | `/var/lib/cqa/files` |

Muốn cất file lên S3 thì cấu hình trong giao diện, **Cài đặt > Lưu trữ file**, riêng cho từng
công ty — không có biến môi trường S3 nào. Xem [Lưu file đính kèm lên S3](/guide/s3-storage).

## Xuất file kết quả (tùy chọn)

| Biến | Mô tả | Mặc định |
|------|-------|----------|
| `EXPORT_MAX_ROWS` | Trần số dòng cho mỗi lần xuất CSV/Excel ở [trang Kết quả](/usage/results) | `20000` |

Vượt trần, hệ thống báo để thu hẹp bộ lọc thay vì dựng một file quá lớn trong bộ nhớ máy chủ.

## Nhật ký hệ thống (tùy chọn)

| Biến | Mô tả | Mặc định |
|------|-------|----------|
| `ACTIVITY_LOG_RETENTION_DAYS` | Số ngày giữ [nhật ký hệ thống](/usage/activity-logs). Đặt `0` để giữ mãi | `90` |

Nhật ký chỉ ghi thêm chứ không bao giờ tự vơi, chạy lâu là thành một trong những bảng nặng nhất
database. Mỗi ngày lúc 3h15 sáng, CQA xoá các dòng cũ hơn số ngày cấu hình, xoá theo lô để không
khoá bảng lâu.

Cần giữ dài hơn cho mục đích kiểm toán thì tăng số ngày, hoặc đặt `0` rồi tự sao lưu bảng
`activity_logs` định kỳ.

## Đồng bộ bảng giá AI (tùy chọn)

Chi phí AI hiển thị trong ứng dụng được tính từ bảng đơn giá theo token. Bảng này có
sẵn trong chương trình và có thể cập nhật định kỳ từ một nguồn công khai khi được
bật rõ ràng, để không phải chờ bản phát hành mới mỗi khi nhà cung cấp đổi giá hoặc
ra model mới.

| Biến | Mô tả | Mặc định |
|------|-------|----------|
| `PRICING_SYNC_ENABLED` | Bật/tắt đồng bộ. Tắt thì chỉ dùng bảng có sẵn trong chương trình | `false` |
| `PRICING_SYNC_URL` | Nguồn dữ liệu giá, bắt buộc HTTPS | Bảng giá công khai của LiteLLM |
| `PRICING_SYNC_INTERVAL_HOURS` | Số giờ giữa hai lần đồng bộ | `168` (7 ngày) |

Dữ liệu tải về được coi là không đáng tin và phải qua kiểm tra trước khi dùng: chỉ
nhận HTTPS, không đi theo chuyển hướng sang tên miền khác, giới hạn dung lượng tải,
chỉ đọc hai con số đơn giá của mỗi model, loại bỏ bản ghi có tên bất thường hoặc giá
âm/quá lớn, và nếu số model hợp lệ thu được quá ít thì giữ nguyên bảng đang dùng.
Đồng bộ hỏng không ảnh hưởng vận hành — bảng có sẵn trong chương trình vẫn phục vụ.

Mặc định kết nối này tắt. Chỉ đặt `PRICING_SYNC_ENABLED=true` khi chủ động cho phép tải bảng giá công khai. Khi tắt,
model mới chưa có trong bảng sẽ hiện là chưa rõ giá, chứ không bị tính nhầm theo
giá của model khác.

## SSL (tùy chọn)

| Biến | Mô tả | Mặc định |
|------|-------|----------|
| `LEGO_DOMAIN` | Domain cho SSL tự động (Let's Encrypt) | _(trống = HTTP mode)_ |
| `LEGO_EMAIL` | Email cho Let's Encrypt | |

::: tip
Để trống `LEGO_DOMAIN` nếu bạn không cần SSL hoặc đã có reverse proxy riêng (Cloudflare, Caddy...).
:::

## Tạo giá trị bảo mật

```bash
# Mật khẩu database
openssl rand -hex 16

# JWT secret (32+ ký tự)
openssl rand -hex 32

# Encryption key (đúng 32 bytes)
openssl rand -hex 16
```

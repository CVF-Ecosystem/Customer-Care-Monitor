# REST API

Trang API này được kế thừa từ CQA và **chưa được đối chiếu toàn bộ endpoint** với bản fork. Dùng cùng [bản kiểm toán tài liệu](/reviews/USER_DOCS_AUDIT_2026-09-26). Endpoint Setup và đăng nhập là ngoại lệ không cần JWT; các endpoint nghiệp vụ khác yêu cầu xác thực và quyền phù hợp.

## Base URL

```
http://localhost:8088/api/v1
```

## Setup (lần đầu)

### Kiểm tra trạng thái setup
```
GET /setup/status
Response: { "needs_setup": true }
```

### Tạo tài khoản admin đầu tiên
```
POST /setup
Body: { "email": "...", "password": "...", "name": "...", "workspace_name": "..." }
Response: { "access_token": "..." }
```
Chỉ hoạt động khi chưa có user nào; tạo luôn workspace duy nhất.

## Authentication

### Đăng nhập
```
POST /auth/login
Body: { "email": "...", "password": "..." }
Response: { "access_token": "..." }
```

### Refresh token
```
POST /auth/refresh
Cookie: refresh_token (HttpOnly)
```

### Đăng xuất
```
POST /auth/logout
```

## Tenant endpoints

Các endpoint nghiệp vụ còn dùng scope nội bộ `/api/v1/tenants/:tenantId/...`; bản fork chỉ có một workspace. `POST /tenants` và `DELETE /tenants/:tenantId` bị khóa. Đừng dùng phần tên `tenant` để suy ra hỗ trợ nhiều công ty.

### Kênh chat
| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/channels` | Danh sách kênh |
| POST | `/channels` | Thêm kênh mới |
| GET | `/channels/:id` | Chi tiết kênh |
| PUT | `/channels/:id` | Cập nhật kênh |
| DELETE | `/channels/:id` | Xóa kênh |
| POST | `/channels/:id/sync` | Đồng bộ tin nhắn |
| POST | `/channels/:id/test` | Test kết nối |

### Cuộc hội thoại
| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/conversations` | Danh sách cuộc hội thoại |
| GET | `/conversations/:id/messages` | Tin nhắn trong cuộc hội thoại |
| GET | `/conversations/:id/evaluations` | Kết quả đánh giá |
| GET | `/conversations/export` | Xuất dữ liệu |

### Công việc
| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/jobs` | Danh sách công việc |
| POST | `/jobs` | Tạo công việc mới |
| GET | `/jobs/:id` | Chi tiết công việc |
| PUT | `/jobs/:id` | Cập nhật công việc |
| DELETE | `/jobs/:id` | Xóa công việc |
| POST | `/jobs/:id/trigger` | Chạy ngay |
| POST | `/jobs/:id/test-run` | Chạy thử |
| GET | `/jobs/:id/results` | Kết quả đánh giá |

### Kết quả (mọi công việc)
| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/results` | Một trang kết quả của cả công ty, kèm số đếm theo từng nhãn |
| GET | `/results/facets` | Dữ liệu dựng bộ lọc: loại công việc đang có, danh sách công việc, kênh, nhãn |
| GET | `/results/export` | Xuất CSV hoặc Excel theo đúng bộ lọc |

Tham số lọc dùng chung cho cả ba: `job_type` (`qc_analysis` mặc định hoặc `classification`),
`job_ids`, `channel_ids`, `tags` (ngăn nhau bằng dấu phẩy), `verdict` (`all`, `pass`, `fail`,
`skip`, `classified`), `date_field` (`conv` là ngày hội thoại, `eval` là ngày đánh giá), `from`,
`to` (`YYYY-MM-DD`), `q` (tên khách), `score_min`, `score_max`, `sort` (`recent`, `score_asc`,
`score_desc`), `page`, `page_size` (tối đa 100). Riêng export nhận thêm `format` (`csv` hoặc
`xlsx`) và trả HTTP 400 với `error: export_too_large` khi vượt trần dòng.

### Dashboard
| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/dashboard` | Thống kê tổng quan |

### Cài đặt
| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/settings` | Xem cài đặt |
| PUT | `/settings/ai` | Cấu hình AI |
| PUT | `/settings/general` | Cài đặt chung |
| POST | `/settings/ai/test` | Test kết nối AI |
| GET | `/settings/ai/models` | Danh sách model khả dụng (ưu tiên bản đã lưu) |
| POST | `/settings/ai/models/refresh` | Lấy lại danh sách model từ nhà cung cấp |

### Người dùng
| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/users` | Danh sách thành viên |
| POST | `/users/invite` | Mời thành viên |
| PUT | `/users/:id/role` | Thay đổi role |
| PUT | `/users/:id/reset-password` | Đặt lại mật khẩu (owner/admin) |
| DELETE | `/users/:id` | Xóa thành viên |

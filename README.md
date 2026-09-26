# Customer Care Monitor AI

**Theo dõi chất lượng chăm sóc khách hàng từ các cuộc hội thoại, trong một không gian làm việc của riêng bạn.**

Customer Care Monitor AI là sản phẩm của [Blackbird081](https://github.com/Blackbird081), dành cho một công ty hoặc cá nhân trên mỗi bản cài đặt. Ứng dụng tập hợp hội thoại từ các kênh đã kết nối, dùng AI hỗ trợ đánh giá chất lượng và phân loại nội dung, rồi trình bày kết quả để người phụ trách kiểm tra.

## Bạn có thể làm gì?

- Kết nối Zalo OA, Facebook Messenger và Pancake; đồng bộ cuộc hội thoại và tin nhắn.
- Tạo công việc đánh giá chất lượng hoặc phân loại hội thoại theo quy tắc của mình.
- Xem kết quả, tìm cuộc hội thoại cần chú ý, theo dõi chi phí AI và nhật ký hoạt động.
- Mời người cùng làm việc và phân quyền trong **một workspace** của bản cài đặt.

Kết quả AI là **gợi ý để con người xem xét**. Ứng dụng hiện chưa có bước xác nhận chính thức của người duyệt; không nên dùng điểm số hoặc nhãn AI làm quyết định cuối cùng khi chưa kiểm tra hội thoại và bằng chứng.

## Bắt đầu

Cần Docker và Docker Compose. Từ thư mục dự án:

```bash
cp .env.example .env
# Điền DB_PASSWORD, MYSQL_ROOT_PASSWORD, JWT_SECRET và ENCRYPTION_KEY trong .env
docker compose up -d --build
```

Mở `http://localhost:8088`. Ở lần truy cập đầu tiên, nhập tên công ty hoặc cá nhân và tạo tài khoản quản trị. Workspace được tạo cùng tài khoản này; bạn không cần tạo thêm công ty sau khi đăng nhập.

Xem [hướng dẫn cài đặt và sử dụng](https://cvf-ecosystem.github.io/Customer-Care-Monitor-AI/). Bản này được build từ source; chưa có Docker image phát hành riêng.

## Phát triển dự án

Backend dùng Go (module trong `backend/`), frontend dùng Vue và Node.js 24. Để kiểm tra khi sửa mã:

```bash
cd backend
go test ./...
go build ./...
```

```bash
cd frontend
npm ci
npm run build
```

Blackbird081 định hướng và phát triển sản phẩm, với Claude và Codex là hai agent hỗ trợ. Các thay đổi trong repo được quản lý theo [quy trình CVF](AGENTS.md). CVF hiện kiểm soát **quy trình thay đổi mã và tài liệu**; các cơ chế CVF cho luồng AI khi ứng dụng chạy vẫn là phần [đang được thiết kế](docs/PRODUCT_DIRECTION.md).

## Giấy phép

Dự án được phát hành theo [giấy phép MIT](LICENSE). File LICENSE chứa điều khoản và các thông báo bản quyền cần giữ lại đối với mã được sử dụng trong sản phẩm.

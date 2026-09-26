# Customer Care Monitor AI

Ứng dụng giám sát chất lượng chăm sóc khách hàng cho **một công ty hoặc một cá nhân trên mỗi bản cài đặt**. Ứng dụng lấy hội thoại từ các kênh đã kết nối, dùng AI đánh giá chất lượng hoặc phân loại, và cho người phụ trách xem kết quả, chi phí và nhật ký hoạt động.

Đây là dự án riêng của [Blackbird081](https://github.com/Blackbird081), phát triển từ mã nguồn [Chat Quality Agent (CQA)](https://github.com/tanviet12/chat-quality-agent) của SePay theo [giấy phép MIT](LICENSE). Các thay đổi của sản phẩm này được quản lý trong [repo Customer Care Monitor AI](https://github.com/CVF-Ecosystem/Customer-Care-Monitor-AI).

## Người phát triển

- **Blackbird081** — chủ repo, định hướng và phát triển sản phẩm.
- **Claude** — agent hỗ trợ phát triển.
- **Codex** — agent hỗ trợ phát triển.

SePay được ghi công là **nguồn mã CQA**, tách biệt với nhóm phát triển sản phẩm mới. Lịch sử đóng góp của repo nguồn được giữ ở [CQA upstream](https://github.com/tanviet12/chat-quality-agent) và [tag lưu trữ lịch sử nhập mã](https://github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/tree/cqa-import-history-2026-09-26).

## Khác gì so với CQA gốc?

| Nội dung | CQA gốc | Customer Care Monitor AI hiện tại |
|---|---|---|
| Đối tượng sử dụng | Một hệ thống có thể quản lý nhiều công ty | Một bản cài đặt dành cho một công ty hoặc cá nhân; workspace được tạo cùng admin đầu tiên |
| Công ty/workspace | Có luồng tạo, chuyển và xóa công ty | Các luồng đó bị khóa; dữ liệu cũ có nhiều workspace bị chặn khi khởi động |
| Mã và tính năng kế thừa | Đồng bộ kênh, công việc AI, kết quả, thống kê | Giữ nền tảng CQA và bổ sung kiểm tra đầu ra AI, trạng thái lỗi và ghi kết quả nguyên tử cho analyzer |
| CVF trong quy trình phát triển | Không phải phạm vi của repo nguồn | Thay đổi repo đi qua manifest, policy, trạng thái phiên, work order, review và kiểm tra cấu trúc CVF |
| CVF trong luồng AI của ứng dụng | Không có khẳng định từ bản fork | **Chưa tích hợp và chưa kiểm chứng** gate rủi ro, phê duyệt, audit và provider routing theo CVF ở runtime |
| Cài đặt/phát hành | Script và image CQA gốc | Build từ source của repo này; chưa có image hoặc release chính thức cho bản fork |

Vì vậy, “có CVF tham gia kiểm soát” hiện có nghĩa là **kiểm soát cách repo này được thay đổi và ghi bằng chứng**. Nó chưa có nghĩa rằng mọi phản hồi AI trong ứng dụng đã đi qua CVF. [Bản đồ trạng thái và lộ trình](docs/PRODUCT_DIRECTION.md) nêu rõ phần đã có, phần mới là kế hoạch.

## Phạm vi sản phẩm

- Một workspace được tạo cùng tài khoản quản trị đầu tiên. Có thể thêm nhiều nhân viên và phân quyền trong workspace đó.
- Không có luồng tạo, chuyển hoặc xóa công ty/workspace qua giao diện hay API. Dữ liệu cũ có nhiều workspace sẽ bị chặn khi khởi động để xử lý chuyển đổi một cách có chủ đích.
- Các cột `tenant_id` và đường dẫn API `/tenants/:tenantId` vẫn là khóa phạm vi nội bộ để giữ tương thích với mã CQA. Chúng không thể hiện mô hình SaaS nhiều công ty.
- Kết quả AI là **đề xuất đánh giá** dựa trên hội thoại. Chưa có quy trình xác nhận của con người để biến kết quả thành kết luận chính thức.

## Chức năng hiện có

- Kết nối Zalo OA, Facebook Messenger và Pancake theo khả năng kế thừa từ CQA.
- Đánh giá QC, phân loại hội thoại, thống kê, nhật ký chi phí AI và gửi thông báo.
- Kiểm tra trường bắt buộc và bằng chứng trong phản hồi AI trước khi lưu. Batch phải trả đủ kết quả và giữ đúng liên kết với từng hội thoại.
- Lưu các bản ghi kết quả của một hội thoại trong một transaction. Lượt chạy có lỗi được ghi `partial` hoặc `error` và không tiến mốc quét khi còn lỗi.

Tình trạng từng kiểm soát CVF, phần chưa triển khai và thứ tự nâng cấp nằm tại [định hướng sản phẩm](docs/PRODUCT_DIRECTION.md). Xem [hướng dẫn cho bản fork](docs/home.md) và [kết quả rà soát tài liệu](docs/reviews/USER_DOCS_AUDIT_2026-09-26.md) trước khi dùng các trang kế thừa CQA.

## Kiểm soát CVF cho thay đổi tiếp theo

Đọc [AGENTS.md](AGENTS.md), [manifest](.cvf/manifest.json), [policy](.cvf/policy.json), [session memory](CVF_SESSION_MEMORY.md) và [documentation index](docs/INDEX.md) trước khi sửa repo. Core CVF nằm ở thư mục sibling `../.Controlled-Vibe-Framework-CVF`; trên máy mới chạy `scripts/initialize_cvf_clone.ps1` để lấy đúng commit đã ghim.

Bootstrap, phạm vi kiểm soát và giới hạn của lần áp dụng này được ghi ở [quyết định CVF](docs/decisions/CVF_ADOPTION_2026-09-26.md) và [kết quả kiểm tra](docs/reviews/CVF_ONBOARDING_CHECK_2026-09-26.md). Doctor xác nhận cấu trúc quản trị repo; các kiểm soát CVF ở runtime vẫn cần thiết kế và kiểm chứng riêng.

## Chạy từ source

Yêu cầu Docker Compose. Sao chép `.env.example` thành `.env`, điền các secret và cấu hình cần thiết, sau đó chạy:

```bash
docker compose up -d --build
```

Mặc định truy cập `http://localhost:8088`. Lần đầu, trang Setup yêu cầu tên công ty hoặc cá nhân và tài khoản quản trị. Nếu dùng Let's Encrypt, đặt `HTTP_PORT=80` và `HTTPS_PORT=443` cùng domain hợp lệ. Script cài đặt, Compose image và release script của CQA đã được bỏ khỏi bản fork vì chúng triển khai image CQA gốc.

Để phát triển trực tiếp: backend dùng Go theo `backend/go.mod`, frontend dùng Node.js 24 và `npm ci` trong `frontend/`. Kiểm tra nhanh bằng `go test ./...` trong `backend/` và `npm run build` trong `frontend/`.

## Nguồn gốc và giấy phép

Mã nền được nhập từ CQA tại commit `6546574b23aded18d292c6382a06727baacfe3fe`. Tên module Go `github.com/vietbui/chat-quality-agent` hiện được giữ để không làm vỡ import nội bộ; việc đổi đường dẫn module sẽ là một thay đổi riêng. [LICENSE](LICENSE) giữ thông báo bản quyền SePay và CVF-Ecosystem theo điều khoản MIT.

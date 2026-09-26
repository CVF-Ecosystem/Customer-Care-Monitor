# Rà soát hướng dẫn sử dụng cho Customer Care Monitor AI

Trạng thái: rà soát mã nguồn và cấu hình, chưa chạy E2E với provider hoặc kênh thật. Không dùng tài liệu này làm bằng chứng rằng CVF kiểm soát runtime AI.

Kiểm tra tĩnh: `npm ci` và `npm run docs:build` trong `docs/` thành công với Node.js 24; VitePress build toàn bộ trang người dùng. `npm ci` báo 5 advisory (2 moderate, 3 high) trong cây dependency docs; đây là việc nâng cấp dependency riêng. Project doctor và catalog check cần được chạy lại trước commit. Không có thử nghiệm triển khai thật.

## Kết luận

**Bộ hướng dẫn CQA cũ không còn phù hợp nguyên trạng.** Các trang cài đặt, Setup, cập nhật và đa công ty có chỉ dẫn sai đối với bản fork. Các trang nghiệp vụ khác mô tả phần lớn chức năng kế thừa, nhưng chưa được kiểm chứng bằng một bản cài chạy thật; chúng được gắn cảnh báo và cần kiểm tra theo từng luồng trước khi bỏ cảnh báo.

| Trang/nhóm | Kết quả rà soát | Hành động |
|---|---|---|
| `guide/introduction`, README, trang chủ docs | Tên CQA, multi-tenant và phạm vi CVF dễ gây hiểu lầm | Viết lại khác biệt CQA ↔ bản fork; nêu CVF chỉ kiểm soát quy trình repo hiện tại |
| `guide/installation` | Trỏ script và repo CQA gốc, cổng 80 và container `cqa-*` sai với Compose hiện tại | Viết lại build từ source; cổng mặc định 8088/8443; không dùng image CQA |
| `guide/initial-setup` | Bắt tạo công ty sau admin, trong khi Setup mới tạo workspace cùng admin | Viết lại một bước Setup |
| `guide/updates` | `docker compose pull`/Watchtower kéo image CQA; workflow release hiện cũng trỏ image CQA | Viết lại quy trình thử nghiệm từ source; chặn suy diễn đây là hướng dẫn nâng cấp production |
| `guide/domain-ssl` | Mặc định cổng 80/443 không đúng trên host | Viết lại theo `HTTP_PORT`/`HTTPS_PORT`; cần 80/443 công khai cho Let's Encrypt |
| `admin/multi-tenant` | Trái trực tiếp với `CreateTenant`, `DeleteTenant` và kiểm tra một workspace | Thay bằng hướng dẫn một workspace |
| `reference/api` | Thiếu `workspace_name` ở Setup và khẳng định mọi endpoint cần JWT | Sửa các điểm đã đối chiếu; phần còn lại gắn trạng thái chưa kiểm hết |
| `reference/env-vars` | Thiếu cổng host; một số mô tả CQA còn nguyên | Bổ sung cổng, chỉ rõ nguồn kiểm tra tiếp theo |
| `guide/s3-storage`, `admin/users`, `admin/mcp`, `admin/demo-data` | Chức năng còn mã liên quan, nhưng câu chữ đa công ty/CQA và hướng dẫn tích hợp chưa được kiểm chứng end-to-end | Giữ như tài liệu kế thừa có cảnh báo; kiểm thử riêng trước khi coi là chính thức |
| Toàn bộ `usage/*`, `faq`, `changelog` | Nhiều chi tiết UI, nhà cung cấp, giá/model, chi phí và hành vi kênh có thể thay đổi; một số câu còn gọi sản phẩm là CQA | Giữ để tra cứu có cảnh báo; không coi nhận định chất lượng AI, tiết kiệm %, giá hay giới hạn API bên thứ ba là bằng chứng đã kiểm chứng |

## Bằng chứng mã nguồn chính

- `frontend/src/views/Setup.vue` và `backend/api/handlers/auth.go`: tên workspace là bắt buộc; Setup tạo workspace, admin và Owner trong một transaction.
- `backend/api/handlers/tenants.go` và `backend/db/single_workspace.go`: tạo/xóa workspace bị từ chối; database có hơn một workspace bị chặn khi khởi động.
- `docker-compose.yml` và `.env.example`: build `app`/`nginx` từ source, MySQL 8; host port 8088/8443 mặc định, không có tên container `cqa-app` hoặc `cqa-db` cố định.
- `backend/api/handlers/version.go`: nguồn kiểm tra release vẫn là `tanviet12/chat-quality-agent`; banner cập nhật trong app chưa trỏ bản fork.
- `.github/workflows/release.yml`: workflow thủ công vẫn push `buitanviet/chat-quality-agent` và nginx image CQA; không chạy workflow này để phát hành bản fork.
- `scripts/backup-db.sh`: mặc định `/opt/cqa/.env`, `/opt/cqa/backups`, `cqa-db`; cần điều chỉnh và thử phục hồi cho Compose bản fork.
- `.cvf/manifest.json`, `.cvf/policy.json`, `CVF_SESSION/*`: kiểm soát thay đổi repository. `docs/PRODUCT_DIRECTION.md` nêu các gate CVF runtime còn thiếu. Doctor 25/25 chỉ xác nhận cấu trúc governance repo.

## Việc còn mở

1. Kiểm thử từng luồng UI/API với một bản cài mới, sau đó sửa và bỏ cảnh báo ở các trang `usage/*`, `admin/*`, S3, FAQ.
2. Tách hoặc vô hiệu hóa các đường dẫn phát hành CQA còn trong workflow, endpoint version và script backup trước khi có quy trình cập nhật production.
3. Kiểm tra lại model/giá, giới hạn API kênh và hướng dẫn MCP với nguồn chính thức tại thời điểm phát hành; không dùng số liệu kế thừa để cam kết chi phí hay chất lượng.
4. Thực hiện kiểm chứng với provider thật trước mọi tuyên bố rằng CVF kiểm soát AI ở runtime.

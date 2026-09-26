# Cập nhật bản fork

Bản fork hiện được build từ source; chưa có Docker image và kênh phát hành riêng được xác nhận. **Không dùng `docker compose pull`, Watchtower hoặc script cập nhật CQA gốc** để cập nhật bản này. Workflow release còn chứa đích image CQA cũ và không phải quy trình phát hành được chấp nhận cho sản phẩm này.

Trước khi cập nhật một bản cài đang có dữ liệu, sao lưu và thử phục hồi database, đồng thời giữ bản `.env` và các volume file. Script `scripts/backup-db.sh` kế thừa CQA mặc định trỏ `/opt/cqa` và container `cqa-db`, trong khi Compose hiện tại không đặt tên container cố định; cần cấu hình lại và kiểm tra script riêng trước khi dùng làm bằng chứng sao lưu.

Đối với môi trường thử nghiệm không có dữ liệu cần giữ:

```bash
git fetch origin
git switch main
git pull --ff-only
docker compose up -d --build
docker compose ps
docker compose logs --tail=100 app db nginx
```

Với môi trường có dữ liệu, hãy kiểm tra thay đổi schema, phương án sao lưu/phục hồi và khả năng quay lui cho từng phiên bản trước khi chạy các lệnh trên. Không có lời hứa tương thích nâng cấp tự động từ database CQA nhiều công ty.

Endpoint kiểm tra phiên bản trong mã hiện còn trỏ GitHub Releases của CQA gốc; chip/banner cập nhật trên giao diện **không phải tín hiệu phát hành của Customer Care Monitor AI** cho đến khi nguồn phát hành được đổi và kiểm chứng.

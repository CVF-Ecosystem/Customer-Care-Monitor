# Cập nhật bản fork

Customer Care Monitor AI hiện được build từ source; chưa có Docker image hoặc kênh phát hành riêng. **Không dùng `docker compose pull` hoặc Watchtower** để cập nhật bản cài này.

Trước khi cập nhật một bản cài đang có dữ liệu, sao lưu và thử phục hồi database, đồng thời giữ bản `.env` và các volume file. Script `scripts/backup-db.sh` hiện có mặc định trỏ `/opt/cqa` và container `cqa-db`, trong khi Compose hiện tại không đặt tên container cố định; cần cấu hình lại và kiểm tra script riêng trước khi dùng làm bằng chứng sao lưu.

Đối với môi trường thử nghiệm không có dữ liệu cần giữ:

```bash
git fetch origin
git switch main
git pull --ff-only
docker compose up -d --build
docker compose ps
docker compose logs --tail=100 app db nginx
```

Với môi trường có dữ liệu, hãy kiểm tra thay đổi schema, phương án sao lưu/phục hồi và khả năng quay lui cho từng phiên bản trước khi chạy các lệnh trên. Không có lời hứa nâng cấp tự động từ database nhiều công ty.

Ứng dụng kiểm tra GitHub Releases của chính repo này. Vì chưa có release chính thức, chip/banner cập nhật **không phải tín hiệu có bản mới** cho đến khi quy trình phát hành được thiết lập và kiểm chứng.

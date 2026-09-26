# Giới thiệu

**Customer Care Monitor AI** là bản fork của [Chat Quality Agent (CQA)](https://github.com/tanviet12/chat-quality-agent) cho **một công ty hoặc cá nhân trên mỗi bản cài đặt**. Mã CQA của SePay là nền tảng; các quyền tác giả nguồn vẫn được giữ trong [LICENSE](https://github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/blob/main/LICENSE).

## Khác biệt với CQA

| Chủ đề | CQA gốc | Bản fork này |
|---|---|---|
| Phạm vi | Nhiều công ty trên một hệ thống | Một workspace cố định, tạo khi Setup |
| Quản trị công ty | Tạo, chuyển, xóa công ty | Không có các luồng này; API tạo/xóa trả lỗi |
| Phân tích AI | Kết quả QC và phân loại | Kế thừa chức năng; bổ sung kiểm tra trường/bằng chứng và ghi kết quả nguyên tử trong analyzer |
| CVF | Không là hợp đồng của bản fork | Có kiểm soát **thay đổi repository** qua manifest, policy, work order, review, doctor |
| CVF runtime | Chưa có bằng chứng của bản fork | Chưa tích hợp gate rủi ro/phê duyệt/audit/provider theo CVF vào luồng AI; không được coi doctor là kiểm chứng runtime |
| Phân phối | Script và Docker image CQA | Hiện chỉ hướng dẫn build từ source của repo này |

Ứng dụng kế thừa đồng bộ Zalo OA, Facebook Messenger, Pancake; công việc QC/phân loại; dashboard; thông báo; nhật ký chi phí. Kết quả AI là **đề xuất hỗ trợ người phụ trách**, chưa có hàng đợi phê duyệt và trạng thái xác nhận của con người. [Định hướng sản phẩm](/PRODUCT_DIRECTION) ghi chi tiết phần đã có và chưa có.

## Đọc tiếp

- [Cài đặt từ source](/guide/installation)
- [Thiết lập lần đầu](/guide/initial-setup)
- [Rà soát độ phù hợp của tài liệu kế thừa](/reviews/USER_DOCS_AUDIT_2026-09-26)

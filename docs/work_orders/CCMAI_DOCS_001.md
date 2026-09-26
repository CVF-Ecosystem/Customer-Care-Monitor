# CCMAI-DOCS-001: README và hướng dẫn sử dụng bản fork

Trạng thái: REVIEW; mức rủi ro R2 vì tài liệu có thể dẫn tới cài đặt, cập nhật và xử lý dữ liệu sai.

## INTAKE → DESIGN → SPEC

Yêu cầu của owner: nêu rõ khác biệt giữa CQA cũ và bản fork có CVF kiểm soát; rà soát độ phù hợp của hướng dẫn sử dụng. Giữ công trạng CQA/SePay. Phân biệt CVF quản lý thay đổi repo với CVF runtime chưa được chứng minh. Dùng mã nguồn/Compose làm sự thật hiện hành; trang chưa kiểm chứng phải có trạng thái rõ ràng.

## Phạm vi BUILD

- README, site tài liệu, các hướng dẫn cài đặt/Setup/cập nhật/đa công ty/SSL và điểm API/env đã xác nhận.
- Workflow build tài liệu, để CI không cố publish vào GitHub Pages chưa bật.
- Báo cáo audit và cảnh báo trên các trang CQA kế thừa chưa kiểm chứng.
- Trạng thái/handoff CVF của tranche tài liệu.

Không đổi mã ứng dụng, workflow phát hành, dữ liệu production, secret hoặc runtime gate. REVIEW kiểm tra link, build site, doctor/catalog và đối chiếu các claim chính với source. Không được tuyên bố provider-backed CVF governance từ các kiểm tra tĩnh này.

## Kết quả BUILD

Các trang cốt lõi đã viết lại; các trang nghiệp vụ chưa kiểm chứng được gắn cảnh báo. `npm run docs:build` PASS với Node.js 24. Báo cáo và giới hạn: `docs/reviews/USER_DOCS_AUDIT_2026-09-26.md`. Chờ kiểm tra độc lập trước khi FREEZE hoặc bỏ cảnh báo kế thừa.

## Owner-directed Pages follow-up

Owner chọn GitHub Actions làm Pages source và yêu cầu tiếp tục xuất bản tài liệu. Khôi phục upload artifact/deploy trên `main`; pull request chỉ build. Xác nhận Pages API đã trả `build_type: workflow` và URL dự kiến `https://cvf-ecosystem.github.io/Customer-Care-Monitor-AI/`. Kiểm tra workflow, URL công khai và ghi giới hạn; không thay đổi nội dung ứng dụng hay tuyên bố CVF runtime.

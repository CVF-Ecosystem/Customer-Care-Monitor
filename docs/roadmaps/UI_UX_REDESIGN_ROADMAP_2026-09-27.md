# Roadmap redesign UI/UX — Customer Care Monitor AI

**Trạng thái:** DRAFT, chờ owner duyệt. **Ngày:** 2026-09-27. **Tác giả:** Claude (ORCHESTRATOR / SPEC_AUTHOR cho kế hoạch thiết kế, theo phân công của owner). **Mã kế hoạch:** `CCMAI-UXROADMAP-001`. **Rủi ro:** thiết kế trên canvas là R1; triển khai vào ứng dụng tối thiểu R2 vì giao diện hiển thị trạng thái nguồn, confidence và đồng bộ đã qua review.

**Đầu vào:**
- [Đánh giá UI/UX nền tảng](../reviews/UI_UX_REVIEW_BASELINE_2026-09-27.md): 20 phát hiện UX-01 đến UX-20 và 15 ảnh chụp.
- Canvas thử nghiệm Job Detail trên claude.ai: https://claude.ai/artifact/KkF8P2m1iGRpTRa3mkf4ii (riêng tư, của owner).
- Ngữ nghĩa đã qua review độc lập ở R004–R007: bốn trạng thái nguồn, confidence để trống thay vì số giả, chỉ trả 202 sau khi đã ghi trạng thái `syncing`.

Roadmap này chỉ là kế hoạch. Nó không cấp quyền BUILD; mỗi giai đoạn triển khai cần SPEC và WORK_ORDER riêng.

## Mục tiêu

Người phụ trách CSKH mở ứng dụng trên máy tính hoặc điện thoại và trả lời được ngay: **hội thoại nào cần xem lại, vì sao, số liệu có đáng tin không.** Cụ thể:

1. Mọi con số trên màn hình đúng với cách tính và có nhãn đúng nghĩa.
2. Cảnh báo quan trọng (nguồn đã đổi, không đạt, đồng bộ lỗi) luôn nổi bật hơn trạng thái bình thường.
3. Giới hạn "chỉ so sánh cục bộ" luôn thấy được ở mọi màn hình hiển thị trạng thái nguồn.
4. Các màn danh sách dùng tốt ở 390px, không cắt chữ, không tràn ngang.
5. Một thuật ngữ cho một khái niệm; ngày, giờ và số theo ngôn ngữ đang chọn.

## Nguyên tắc không được vi phạm

- **Không đổi hợp đồng API hiện có.** Số liệu mới ưu tiên tính từ dữ liệu API đang trả. Nếu buộc phải thêm, chỉ thêm API đọc, qua tranche backend riêng.
- **Không làm yếu ngữ nghĩa đã review.** Giao diện hiển thị đủ bốn trạng thái nguồn (`changed_since_analysis`, `verification_unavailable`, `legacy_unverified`, `bound_currentness_unverified`). Không được gộp hay đổi tên chúng thành tín hiệu yên tâm hơn thực tế. Confidence rỗng hiển thị là "không có", không phải 0% hay 100%. Nút đồng bộ chỉ báo "đã bắt đầu", không báo "hoàn tất".
- **Không đổi schema DB** vì mục đích giao diện.
- **Giữ Vuetify 4 và Vue 3.** Redesign bằng theme, token và component dùng chung, không viết lại framework.
- **Hỗ trợ tiếng Việt là chính, tiếng Anh là phụ, và dark mode** như hiện tại.
- **Tiếp cận được:** tương phản chữ ≥ 4.5:1 (≥ 3:1 cho chữ ≥ 24px), vùng bấm ≥ 44px, nút và liên kết là phần tử thật, màu cảnh báo khác nhau cả về độ sáng chứ không chỉ về sắc độ.

## Cách làm việc với canvas thiết kế

Mỗi màn hình đi qua đúng chuỗi CVF, nhưng giai đoạn thiết kế diễn ra trên canvas claude.ai để owner xem trực quan:

```text
Brief màn hình -> Canvas (desktop + mobile + các trạng thái) -> Owner góp ý trên canvas
  -> Owner duyệt, ghi lại version id canvas -> SPEC màn hình -> WORK_ORDER
  -> BUILD (frontend) -> REVIEW độc lập (Codex) -> Owner nghiệm thu trực quan -> FREEZE
```

Quy ước:
- **Một canvas cho một khu vực:** Nền tảng, Job Detail, Kết quả, Trang chủ, Kênh & Tin nhắn, Tác vụ, Cài đặt & phụ trợ.
- **Tên artboard** `<Màn> — <kích thước> — <trạng thái>`, ví dụ `Job Detail — 390 — nguồn đã đổi`. Desktop 1440px, mobile 390px.
- **Trạng thái cần vẽ cho mỗi màn danh sách:** có dữ liệu, rỗng, đang tải, lỗi tải, có kết quả nguồn đã đổi, toàn kết quả cũ, đồng bộ một phần hoặc lỗi (nếu có liên quan).
- **Chỉ sửa một nơi tại một thời điểm:** hoặc owner sửa trên claude.ai, hoặc Claude sửa từ đây. Claude luôn đọc bản mới nhất trên canvas trước khi ghi.
- **Dữ liệu trong thiết kế là dữ liệu tổng hợp**, không có dữ liệu khách hàng thật và không dùng thương hiệu kế thừa.
- **Bản được duyệt là căn cứ của SPEC:** SPEC ghi link canvas và version id đã duyệt. Thay đổi thiết kế sau đó là một vòng duyệt mới.

## Các giai đoạn

### Giai đoạn 0 — Nền tảng thiết kế (`CCMAI-UX-000`)

Làm trước mọi màn hình. Mọi canvas sau dùng chung kết quả của giai đoạn này.

| Hạng mục | Nội dung | Sản phẩm |
|---|---|---|
| Design system | Màu: primary thay màu kế thừa, nền, bề mặt, viền, và bộ màu trạng thái (đạt, không đạt, bỏ qua, nguồn đã đổi, không xác minh được, kết quả cũ, đồng bộ lỗi/một phần). Font hỗ trợ tiếng Việt, tự host để chạy được khi không có internet. Thang chữ, khoảng cách, bo góc, đổ bóng. Light và dark. | Design System trên claude.ai + ánh xạ sang theme Vuetify |
| Component dùng chung | Chip kết luận, chip trạng thái nguồn, khung "Trạng thái nguồn dữ liệu" (ghi chú cục bộ + tóm tắt), chip đồng bộ, thẻ số liệu, thẻ kết quả mobile, hàng bảng, hộp thoại (luôn có nút đóng ở góc), menu thao tác ⋯ chứa thao tác phá hủy, thanh lọc cuộn ngang | Artboard component + danh sách props |
| Bảng thuật ngữ | Một từ cho một khái niệm: "Tác vụ AI" / "Công việc", "vấn đề" / "nhãn", "hội thoại" / "tin nhắn", "chấm" / "đánh giá", tên trạng thái | `docs/reference/UI_GLOSSARY.md` (vi/en) |
| Quy tắc định dạng | Ngày dd/mm/yyyy, 24 giờ, thời gian tương đối (không âm, đổi sang giờ/ngày), số và tiền theo ngôn ngữ | Mục trong SPEC nền tảng |
| Công cụ chụp ảnh | Đưa quy trình chụp của bản review vào repo: môi trường tách biệt + dữ liệu demo + trạng thái nguồn tổng hợp + Chrome headless, desktop và mobile, ghi lỗi JS | `scripts/ui-screenshots.*` + hướng dẫn |

**Quyết định cần owner chốt:** màu primary mới; font (Be Vietnam Pro như bản thử, hoặc giữ font mặc định); có giữ dark mode ở mức ngang light không.

**Hoàn thành khi:** owner duyệt Design System và bảng thuật ngữ; công cụ chụp ảnh tái tạo được bộ ảnh baseline.

### Giai đoạn 1 — Sửa dữ liệu hiển thị sai (`CCMAI-UX-001`, `CCMAI-UX-002`)

Không cần thiết kế mới, nên làm song song với Giai đoạn 0. Làm trước redesign để các thiết kế mới dựa trên số liệu đúng.

| Tranche | Phát hiện | Phạm vi | Ghi chú |
|---|---|---|---|
| `CCMAI-UX-001a` | UX-01, UX-03 | Frontend: thời gian tương đối không âm; "nhãn" cho tác vụ phân loại | Dữ liệu demo sinh mốc thời gian tương lai nằm trong `demo.go`; phối hợp với tranche thương hiệu demo của Codex |
| `CCMAI-UX-001b` | UX-02, UX-04, UX-05 | Backend chỉ đọc: số vấn đề chỉ đếm `qc_violation`; thống nhất nguồn số liệu Job Detail; gom nhóm ngày theo giờ +07:00 | Đổi ý nghĩa một trường API hiện có, phải ghi rõ trong SPEC |
| `CCMAI-UX-001c` | UX-06 | Điều tra vì sao kênh chưa đồng bộ lại có trạng thái `error` | Gần phạm vi đồng bộ R001/R007/R008; làm sau hoặc cùng tranche đồng bộ đang mở |
| `CCMAI-UX-002` | UX-07 | Trang Kết quả: đặt ghi chú cục bộ phía trên danh sách, thêm tiêu đề cột "Nguồn" | Cùng loại với R006-R1, có thể làm ngay |

### Giai đoạn 2 — Thiết kế từng màn hình

Theo thứ tự ưu tiên. Mỗi màn: brief, canvas desktop và mobile đủ trạng thái, owner duyệt.

| Thứ tự | Màn hình (route) | Phát hiện chính | Trọng tâm |
|---|---|---|---|
| 1 | Job Detail (`/jobs/:id`), QC và phân loại | UX-04, 08, 09, 10, 17, 18 | Hoàn thiện từ bản thử: thêm tab lịch sử chạy, hộp thoại chi tiết, trạng thái rỗng/lỗi, bản phân loại |
| 2 | Kết quả (`/results`) và hộp thoại chi tiết | UX-07, 08, 09 | Bảng desktop, thẻ mobile, khung trạng thái nguồn dùng chung với Job Detail |
| 3 | Trang chủ (`/`) | UX-01, 02, 11, 15, 19 | Số liệu đúng nghĩa, hoạt động gần đây, hướng dẫn bắt đầu hợp lý, banner demo trên mobile |
| 4 | Kênh chat (`/channels`, `/channels/:id`) | UX-06, 20 | Trạng thái đồng bộ (chưa đồng bộ / đang đồng bộ / một phần / lỗi) đúng ngữ nghĩa R001/R007; "Kết nối lại" chỉ nổi bật khi cần |
| 5 | Tin nhắn (`/messages`) và chi tiết hội thoại | UX-16 | Đếm đúng hội thoại, bộ lọc có nhãn, đọc hội thoại trên mobile |
| 6 | Tác vụ (`/jobs`, tạo/sửa tác vụ) | UX-12, 13, 14 | Thuật ngữ, định dạng ngày, trạng thái chạy bằng tiếng Việt, biểu mẫu tạo/sửa trên mobile |
| 7 | Cài đặt, Người dùng, Đăng nhập/Thiết lập | — | Áp design system; kiểm tra biểu mẫu và thông báo lỗi |
| 8 | Nhật ký hệ thống, Nhật ký chi phí, Lịch sử thông báo, Kết nối MCP | UX-15 | Bảng dài trên mobile, định dạng số/tiền |

**Hoàn thành mỗi màn khi:** owner duyệt trên canvas, và version id được ghi vào SPEC của màn đó.

### Giai đoạn 3 — Triển khai (`CCMAI-UX-010` trở đi, mỗi màn một tranche)

Mỗi tranche frontend:
- **Phạm vi:** file view của màn đó, component dùng chung, i18n, theme. Không đụng backend trừ khi SPEC ghi rõ và đã có tranche backend tương ứng.
- **Bằng chứng BUILD:**
  - `vue-tsc` + build;
  - vitest cho logic hiển thị (ví dụ gom trạng thái nguồn, định dạng);
  - bộ ảnh chụp desktop và mobile từ công cụ ở Giai đoạn 0, đặt cạnh ảnh baseline;
  - 0 lỗi JS;
  - kiểm tra tương phản và vùng bấm.
- **REVIEW độc lập (Codex):** đúng thiết kế đã duyệt, không làm yếu ngữ nghĩa trạng thái, không đổi hợp đồng API.
- **Nghiệm thu:** owner xem ảnh chụp và bản chạy thật. Sau đó mới FREEZE.

Khi có tranche runtime đang mở trên cùng file (ví dụ R008 đụng luồng đồng bộ và trang Kênh), tranche UX của màn đó chờ tranche runtime xong trước để tránh xung đột.

### Giai đoạn 4 — Kiểm chứng tổng thể

- Chạy lại toàn bộ bộ ảnh chụp và lập bảng so sánh với baseline cho từng phát hiện UX-01 đến UX-20.
- Cập nhật tài liệu hướng dẫn người dùng có ảnh màn hình cũ (`docs/usage/`, `docs/public/screenshots/`).
- Owner nghiệm thu toàn bộ; ghi baseline mới cho các đợt sau.

## Ánh xạ phát hiện → giai đoạn

| Phát hiện | Giai đoạn / tranche |
|---|---|
| UX-01, UX-03 | 1 · `CCMAI-UX-001a` |
| UX-02, UX-04, UX-05 | 1 · `CCMAI-UX-001b` (UX-04 cũng được thiết kế lại ở màn 1) |
| UX-06 | 1 · `CCMAI-UX-001c`, rồi màn 4 |
| UX-07 | 1 · `CCMAI-UX-002`, rồi màn 2 |
| UX-08, UX-09 | 0 (component trạng thái nguồn), màn 1 và 2 |
| UX-10 | màn 1 |
| UX-11, UX-19 | màn 3 |
| UX-12, UX-13, UX-14 | 0 (thuật ngữ, định dạng), màn 6 |
| UX-15 | 0 (định dạng), màn 3 và 8 |
| UX-16 | màn 5 |
| UX-17, UX-18 | 0 (component hộp thoại, menu ⋯), màn 1 |
| UX-20 | màn 4 |

## Thứ tự và phụ thuộc

```text
Giai đoạn 0 (nền tảng) ──┬─> Giai đoạn 2: màn 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8
                         │        (mỗi màn duyệt xong) ─> Giai đoạn 3: tranche của màn đó
Giai đoạn 1 (sửa dữ liệu) ┘  (chạy song song với 0; phải xong trước khi triển khai màn liên quan)
                                                      ─> Giai đoạn 4 khi mọi màn đã FREEZE
```

Có thể thiết kế màn tiếp theo trong khi màn trước đang triển khai. Không triển khai màn nào trước khi Giai đoạn 0 được duyệt.

## Rủi ro và cách giảm

| Rủi ro | Cách giảm |
|---|---|
| Thiết kế đẹp nhưng làm yếu giới hạn tuyên bố (ẩn "chỉ so sánh cục bộ", gộp trạng thái) | Nguyên tắc ở trên là điều kiện duyệt; Codex review riêng mục này ở mỗi tranche |
| Xung đột với tranche runtime đang chạy (R008 trở đi) | Tranche UX chờ tranche runtime trên cùng file; ghi rõ trong WORK_ORDER |
| Hai bên cùng sửa canvas | Chỉ sửa một nơi tại một thời điểm; Claude đọc bản mới nhất trước khi ghi |
| Font tải từ Google không chạy khi triển khai nội bộ không có internet | Tự host font trong frontend |
| Vuetify giới hạn tùy biến | Thiết kế trong khả năng theme/component của Vuetify 4; phần vượt quá phải ghi trong SPEC trước |
| Ảnh chụp và tài liệu hướng dẫn cũ lệch với giao diện mới | Giai đoạn 4 cập nhật `docs/usage/` và ảnh màn hình |

## Ngoài phạm vi

- Thay đổi luồng dữ liệu backend ngoài các mục đọc/đếm nêu ở Giai đoạn 1.
- Tính năng mới (hàng đợi xử lý, giao việc, S2/S3/S5 của roadmap runtime).
- Đổi framework hoặc viết lại ứng dụng.
- Bằng chứng quản trị CVF trực tiếp hay tuyên bố về độ tươi của dữ liệu nguồn.

## Bước tiếp theo

1. Owner duyệt roadmap này và chốt ba quyết định ở Giai đoạn 0 (màu primary, font, mức hỗ trợ dark mode).
2. Claude lập brief và canvas Giai đoạn 0 (design system, component, bảng thuật ngữ) để owner duyệt trực quan.
3. Song song, `CCMAI-UX-002` (ghi chú cục bộ trên trang Kết quả) và `CCMAI-UX-001a/b/c` được viết SPEC/WORK_ORDER, theo phân vai owner chọn.

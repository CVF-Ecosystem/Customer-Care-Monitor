# Roadmap redesign UI/UX — Customer Care Monitor AI

**Trạng thái:** ACTIVE. Owner ủy quyền toàn bộ phần thiết kế cho Claude (2026-09-27); Claude chốt thiết kế, còn triển khai vẫn qua review độc lập. **Ngày:** 2026-09-27. **Tác giả:** Claude (ORCHESTRATOR / SPEC_AUTHOR cho kế hoạch thiết kế, theo phân công của owner). **Mã kế hoạch:** `CCMAI-UXROADMAP-001`. **Rủi ro:** thiết kế trên canvas là R1; triển khai vào ứng dụng tối thiểu R2 vì giao diện hiển thị trạng thái nguồn, confidence và đồng bộ đã qua review.

**Đầu vào:**
- [Đánh giá UI/UX nền tảng](../reviews/UI_UX_REVIEW_BASELINE_2026-09-27.md): 20 phát hiện UX-01 đến UX-20 và 15 ảnh chụp.
- Canvas nền tảng thiết kế (Giai đoạn 0) trên claude.ai: https://claude.ai/artifact/DsSy7rS8zkAr4Gk6DB6vxt (riêng tư, của owner).
- Canvas thử nghiệm Job Detail trên claude.ai: https://claude.ai/artifact/KkF8P2m1iGRpTRa3mkf4ii (riêng tư, của owner).
- [Quyết định hướng thiết kế](../decisions/UI_DESIGN_DIRECTION_2026-09-27.md): tham chiếu thị trường, màu, chữ, dark mode, mẫu tương tác.
- Ngữ nghĩa đã qua review độc lập ở R004–R007: bốn trạng thái nguồn, confidence để trống thay vì số giả, chỉ trả 202 sau khi đã ghi trạng thái `syncing`.

Roadmap này chỉ là kế hoạch. Nó không cấp quyền BUILD; mỗi giai đoạn triển khai cần SPEC và WORK_ORDER riêng.

## Ranh giới với roadmap backend (owner xác nhận 2026-09-28)

Tiến hành các phần redesign độc lập trước. Claude chỉ thiết kế/triển khai giao diện trên hợp đồng API và ngữ nghĩa runtime đã được review. Phần nào cần sửa backend, dữ liệu hoặc hợp đồng API thì **Claude không thực hiện trong tranche UX**; ghi `BLOCKED_API_CONTRACT` hoặc đề xuất riêng, để Codex điều phối tranche R2 tương ứng. Không làm giả số liệu hoặc trạng thái trong frontend để lấp khoảng trống backend.

| Điểm giao | Quyết định ranh giới |
|---|---|
| Trang chủ: số vi phạm QC (`UX-02`) | Giữ `issues` theo nghĩa hiện tại. `qc_violation_count` là API đọc mới trong tranche backend riêng; hoãn phần thẻ/đường dẫn phụ thuộc số này. |
| Kênh demo và trạng thái đồng bộ (`UX-06`) | Scheduler bỏ qua kênh demo được đánh dấu rõ là tranche runtime riêng. Hoãn phần hành vi hoặc trạng thái UI phụ thuộc fix đó. |
| Trạng thái nguồn, confidence, kết quả AI | Chỉ trình bày các trường và giới hạn đã có từ R004–R006. Không đổi cách xác minh, phân loại, tính confidence, gọi provider hay hợp đồng kết quả. |
| Lọc, phân trang, số đếm, export Trang Kết quả | Tuân theo API hiện có; không tạo số toàn cục từ một trang dữ liệu hoặc hứa export theo scope khi endpoint không hỗ trợ. Nhu cầu API mới đưa sang tranche backend riêng. |

Tranche độc lập kế tiếp là [UX-011 Results INTAKE](../specs/RESULTS_SCREEN_UX011_INTAKE_2026-09-28.md), bắt đầu từ DESIGN rồi SPEC và WORK_ORDER trước BUILD. Dashboard/Kênh không được đưa vào work order UX-011. Các màn tiếp theo chỉ mở khi xác định được phần giao với backend theo cùng quy tắc.

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

Mỗi màn hình đi qua đúng chuỗi CVF. Giai đoạn thiết kế diễn ra trên canvas claude.ai; Claude là người chốt thiết kế theo ủy quyền của owner, và owner có thể mở canvas xem hoặc góp ý bất cứ lúc nào:

```text
Brief màn hình -> Canvas (desktop + mobile + light/dark + các trạng thái)
  -> Claude tự kiểm theo quyết định thiết kế, chốt và ghi version id canvas
  -> SPEC màn hình -> WORK_ORDER -> BUILD (frontend)
  -> REVIEW độc lập (Codex) -> FREEZE
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

**Đã chốt** trong [quyết định hướng thiết kế](../decisions/UI_DESIGN_DIRECTION_2026-09-27.md): primary `#3342A8`, nền trung tính lạnh, font Be Vietnam Pro tự host, dark mode ngang hàng light, bộ màu trạng thái đạt AA.

**Hoàn thành khi:** Claude chốt Design System và bảng thuật ngữ; công cụ chụp ảnh tái tạo được bộ ảnh baseline.

### Giai đoạn 1 — Sửa dữ liệu hiển thị sai (`CCMAI-UX-001`, `CCMAI-UX-002`)

Không cần thiết kế mới, nên làm song song với Giai đoạn 0. Làm trước redesign để các thiết kế mới dựa trên số liệu đúng.

| Tranche | Phát hiện | Phạm vi | Ghi chú |
|---|---|---|---|
| `CCMAI-UX-001a` | UX-01, UX-03 | Frontend: thời gian tương đối không âm; "nhãn" cho tác vụ phân loại | REVIEW PASS 2026-09-28 cho BUILD có UX-05 và nhãn đúng UX-02; phụ thuộc UX-000 còn mở ([review](../reviews/UI_OVERNIGHT_BUILDS_INDEPENDENT_REVIEW_2026-09-28.md)) |
| `CCMAI-UX-001b` | UX-02, UX-04, UX-05 | Backend chỉ đọc: số vấn đề chỉ đếm `qc_violation`; thống nhất nguồn số liệu Job Detail; gom nhóm ngày theo giờ +07:00 | Đổi ý nghĩa một trường API hiện có, phải ghi rõ trong SPEC **BLOCKED_API_CONTRACT** cho phần đếm lại UX-02; UX-04 cần quyết định nguồn số liệu; UX-05 đã làm ở UX-001a (chỉ frontend) |
| `CCMAI-UX-001c` | UX-06 | Điều tra vì sao kênh chưa đồng bộ lại có trạng thái `error` | Gần phạm vi đồng bộ R001/R007/R008; làm sau hoặc cùng tranche đồng bộ đang mở Đã điều tra 2026-09-28: scheduler đồng bộ kênh demo có thông tin xác thực giả nên giải mã lỗi ([hồ sơ](../reviews/UX_001C_SYNC_STATUS_INVESTIGATION_2026-09-28.md)); sửa thuộc tranche runtime |
| `CCMAI-UX-002` | UX-07 | Trang Kết quả: đặt ghi chú cục bộ phía trên danh sách, thêm tiêu đề cột "Nguồn" | REVIEW PASS 2026-09-28, FREEZE mở ([review](../reviews/UI_OVERNIGHT_BUILDS_INDEPENDENT_REVIEW_2026-09-28.md)) |

### Giai đoạn 2 — Thiết kế từng màn hình

Theo thứ tự ưu tiên. Mỗi màn: brief, canvas desktop và mobile đủ trạng thái, Claude chốt.

| Thứ tự | Màn hình (route) | Phát hiện chính | Trọng tâm |
|---|---|---|---|
| 1 | Job Detail (`/jobs/:id`), QC và phân loại | UX-04, 08, 09, 10, 17, 18 | Hoàn thiện từ bản thử: thêm tab lịch sử chạy, hộp thoại chi tiết, trạng thái rỗng/lỗi, bản phân loại Thiết kế chốt 2026-09-28 (canvas `1790540352-11c7`), [SPEC](../specs/JOB_DETAIL_SCREEN_UX010_2026-09-28.md); [work order](../work_orders/CCMAI_UX_010.md); BUILD xong 2026-09-28, chờ Codex review ([bằng chứng](../reviews/JOB_DETAIL_SCREEN_UX010_BUILD_2026-09-28.md)) |
| 2 | Kết quả (`/results`) và hộp thoại chi tiết | UX-07, 08, 09 | Bảng desktop, thẻ mobile, khung trạng thái nguồn dùng chung với Job Detail. UX-011: canvas `1790597255-fb44`, [SPEC](../specs/RESULTS_SCREEN_UX011_2026-09-28.md), [work order](../work_orders/CCMAI_UX_011.md); Codex `REVIEW_PASS` ([review](../reviews/CCMAI_UX_011_INDEPENDENT_REVIEW_2026-09-28.md)), FREEZE mở; không sửa API/backend ([ranh giới](../specs/RESULTS_SCREEN_UX011_INTAKE_2026-09-28.md)). |
| 3 | Trang chủ (`/`) | UX-01, 02, 11, 15, 19 | Số liệu đúng nghĩa, hoạt động gần đây, hướng dẫn bắt đầu hợp lý, banner demo trên mobile |
| 4 | Kênh chat (`/channels`, `/channels/:id`) | UX-06, 20 | Trạng thái đồng bộ (chưa đồng bộ / đang đồng bộ / một phần / lỗi) đúng ngữ nghĩa R001/R007; "Kết nối lại" chỉ nổi bật khi cần |
| 5 | Tin nhắn (`/messages`) và chi tiết hội thoại | UX-16 | Đếm đúng hội thoại, bộ lọc có nhãn, đọc hội thoại trên mobile. `CCMAI-UX-012`: canvas `1790601057-3d67`, [SPEC](../specs/MESSAGES_SCREEN_UX012_2026-09-28.md), [work order](../work_orders/CCMAI_UX_012.md); BUILD xong, `REVIEW_PENDING` ([evidence](../reviews/MESSAGES_SCREEN_UX012_BUILD_2026-09-28.md)) |
| 6 | Tác vụ (`/jobs`, tạo/sửa tác vụ) | UX-12, 13, 14 | Thuật ngữ, định dạng ngày, trạng thái chạy bằng tiếng Việt, biểu mẫu tạo/sửa trên mobile. `CCMAI-UX-013`: canvas `1790603381-76c3`, [SPEC](../specs/JOBS_SCREENS_UX013_2026-09-28.md), [work order](../work_orders/CCMAI_UX_013.md); BUILD xong, `REVIEW_PENDING` ([evidence](../reviews/JOBS_SCREENS_UX013_BUILD_2026-09-28.md)) |
| 7 | Cài đặt, Người dùng, Đăng nhập/Thiết lập | — | Áp design system; kiểm tra biểu mẫu và thông báo lỗi. `CCMAI-UX-014`: canvas `1790607141-5a73`, [SPEC](../specs/SETTINGS_USERS_AUTH_UX014_2026-09-28.md), [work order](../work_orders/CCMAI_UX_014.md); BUILD xong, `REVIEW_PENDING` ([evidence](../reviews/SETTINGS_USERS_AUTH_UX014_BUILD_2026-09-28.md)) |
| 8 | Nhật ký hệ thống, Nhật ký chi phí, Lịch sử thông báo, Kết nối MCP | UX-15 | Bảng dài trên mobile, định dạng số/tiền. `CCMAI-UX-015`: canvas `1790608303-0cb6`, [SPEC](../specs/LOGS_COST_NOTIFY_MCP_UX015_2026-09-28.md), [work order](../work_orders/CCMAI_UX_015.md); BUILD xong, `REVIEW_PENDING` ([evidence](../reviews/LOGS_COST_NOTIFY_MCP_UX015_BUILD_2026-09-28.md)) |

**Hoàn thành mỗi màn khi:** thiết kế đạt các nguyên tắc ở trên và quyết định hướng thiết kế, Claude chốt, và version id canvas được ghi vào SPEC của màn đó.

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
- **Nghiệm thu:** ảnh chụp sau triển khai khớp thiết kế đã chốt và không còn phát hiện liên quan trong baseline. Sau đó mới FREEZE.

Khi có tranche runtime đang mở trên cùng file (ví dụ R008 đụng luồng đồng bộ và trang Kênh), tranche UX của màn đó chờ tranche runtime xong trước để tránh xung đột.

### Giai đoạn 4 — Kiểm chứng tổng thể

- Chạy lại toàn bộ bộ ảnh chụp và lập bảng so sánh với baseline cho từng phát hiện UX-01 đến UX-20.
- Cập nhật tài liệu hướng dẫn người dùng có ảnh màn hình cũ (`docs/usage/`, `docs/public/screenshots/`).
- Báo owner kết quả kèm link ảnh so sánh; ghi baseline mới cho các đợt sau.

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

1. [Đã dựng bản đầu, 2026-09-28] Claude dựng canvas Giai đoạn 0: bảng màu light/dark, thang chữ, component dùng chung (chip kết luận, chip trạng thái nguồn, khung trạng thái nguồn, thẻ số liệu, thẻ kết quả mobile, hàng bảng, hộp thoại, menu ⋯) và bảng thuật ngữ.
2. [UX-000: Codex re-review `REVIEW_PASS` UX000-R1 ngày 2026-09-28 — [review](../reviews/UI_FOUNDATION_UX000_R1_INDEPENDENT_REREVIEW_2026-09-28.md)] Ba avatar đạt tương phản; UX-000, UX-002 và UX-001a đều REVIEW PASS, FREEZE vẫn mở. Claude có thể viết work order UX-010 rồi BUILD theo SPEC đã chốt, trả `REVIEW_PENDING` cho Codex.
2a. [UX-010: Codex review `CHANGES_REQUIRED` ngày 2026-09-28 — [review](../reviews/CCMAI_UX_010_INDEPENDENT_REVIEW_2026-09-28.md)] Giữ thiết kế phạm vi lần chạy mới nhất; sửa UX010-R1 trong cùng work order: thẻ “đã đánh giá” phải lọc đúng hội thoại không SKIP, lịch sử phân loại phải dùng ngữ nghĩa phân loại. Claude trả `REVIEW_PENDING` để Codex review lại; chưa FREEZE.
2b. [UX-010: Codex re-review `REVIEW_PASS` UX010-R1 ngày 2026-09-28 — [review](../reviews/CCMAI_UX_010_REPAIR_R1_REREVIEW_2026-09-28.md)] Hai finding đã đóng; FREEZE còn mở. Orchestrator có thể điều phối tranche màn hình tiếp theo theo roadmap.
2c. [UX-011 Results: INTAKE hoàn tất ngày 2026-09-28 — [brief](../specs/RESULTS_SCREEN_UX011_INTAKE_2026-09-28.md)] Claude được tiến hành DESIGN → SPEC → WORK_ORDER → BUILD **chỉ** trong phạm vi giao diện độc lập; mọi giao điểm backend/API/runtime giữ ngoài tranche, trả `REVIEW_PENDING` cho Codex.
3. `CCMAI-UX-001a/b/c` viết SPEC sau khi Codex xong tranche runtime đang mở trên cùng file (R008), để tránh xung đột.

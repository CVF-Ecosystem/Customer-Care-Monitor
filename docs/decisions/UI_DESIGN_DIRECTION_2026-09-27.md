# Quyết định hướng thiết kế UI/UX

**Ngày:** 2026-09-27 · **Người quyết định:** Claude, theo ủy quyền của owner ("Tôi không tham gia sâu vào việc redesign, đó là việc của bạn, thiết kế theo chuẩn hiện đại và chuyên nghiệp") · **Thuộc:** [roadmap redesign UI/UX](../roadmaps/UI_UX_REDESIGN_ROADMAP_2026-09-27.md), Giai đoạn 0 · **Phạm vi:** quyết định thiết kế, không cấp quyền BUILD.

## 1. Tham chiếu thị trường

Customer Care Monitor AI thuộc nhóm **chấm chất lượng hội thoại bằng AI (Auto QA / conversation intelligence)**, gần với Zendesk QA (trước đây là Klaus), MaestroQA, Level AI, Scorebuddy, Observe.AI. Phần xem hội thoại gần với hộp thư hỗ trợ của Intercom, Help Scout, Front. Mẫu giao diện chung rút ra:

| Mẫu | Ở công cụ tham chiếu | Áp dụng cho CCMA |
|---|---|---|
| **Chấm 100% hội thoại, người chỉ xem phần đáng xem** | Zendesk QA "Spotlight" tự nêu hội thoại rủi ro, bất thường, leo thang; MaestroQA có hàng đợi review theo luật | Bộ lọc mặc định **"Cần xem lại"** = không đạt, hoặc nguồn đã đổi, hoặc không xác minh được. Tính ở frontend từ trường API đang có. |
| **Màn review: hội thoại cạnh bảng chấm, bằng chứng được đánh dấu** | Các công cụ QA đặt transcript cạnh scorecard, đánh dấu khoảnh khắc quan trọng để review nhanh hơn | Hộp thoại/trang chi tiết hai cột: diễn biến chat bên trái, kết luận và từng vấn đề bên phải. Bấm một vấn đề thì cuộn tới và tô trích dẫn trong chat (dùng `evidence_refs` sẵn có). |
| **AI phải giải thích được và người có quyền sửa** | Các hướng dẫn chọn công cụ 2026 yêu cầu chấm theo tiêu chí minh bạch, ghi nhận khi người sửa điểm AI, theo dõi tỉ lệ khiếu nại | Luôn ghi rõ "Nhận xét do AI tạo". Confidence chỉ hiện kèm nhãn "mô hình tự ước lượng, chưa hiệu chuẩn" (R005). Sửa điểm và khiếu nại **chưa có ở backend**: ghi vào nhu cầu tương lai (S3 human disposition), không vẽ như đã có. |
| **Dashboard dẫn xuống tận hội thoại** | Dashboard QA hiện xu hướng theo nhóm, nhân viên, kênh, và cho đi thẳng tới hội thoại cụ thể | Mỗi thẻ số liệu và điểm biểu đồ là liên kết tới danh sách đã lọc tương ứng. |
| **Hộp thư ba cột** | Intercom/Help Scout: điều hướng, luồng hội thoại, thông tin khách; các khối ngữ cảnh gập được | Màn Tin nhắn trên desktop: danh sách hội thoại, nội dung chat, ngữ cảnh (kênh, kết quả chấm, trạng thái nguồn). Trên mobile, ba cột thành ba bước. |
| **Mobile kiểu danh sách email** | Help Scout được đánh giá có ứng dụng mobile tốt nhất nhóm | Danh sách mobile là thẻ một cột: tên, thời gian, kết luận, một dòng tóm tắt, cảnh báo nếu có. |
| **Giao diện nhiều chữ, ít trang trí, một màu nhấn** | Công cụ chuyên nghiệp dùng nền trung tính, một màu thương hiệu, màu trạng thái có tiết chế | Nền xám trung tính lạnh, một màu primary, màu trạng thái chỉ dùng cho trạng thái. |

## 2. Quyết định

### 2.1 Màu

Bỏ màu và chú thích kế thừa trong `frontend/src/plugins/vuetify.ts`. Bảng màu riêng, đã tính tương phản trên nền trắng:

| Token | Light | Dark | Dùng cho |
|---|---|---|---|
| primary | `#3342A8` (chữ trắng 8.45:1) | `#9AA6F0` | Nút chính, liên kết, lựa chọn |
| background | `#F5F6F8` | `#0F1217` | Nền trang |
| surface | `#FFFFFF` | `#171B22` | Thẻ, bảng, hộp thoại |
| border | `#E3E6EB` | `#2A303A` | Viền, đường kẻ |
| text | `#1B1F24` | `#E7EAF0` | Chữ chính |
| text-muted | `#5B6470` (6.0:1 trên trắng, 5.55:1 trên nền) | `#A3ABB8` | Chữ phụ, nhãn |
| pass | chữ `#1E6B34` / nền `#E4F4EA` | `#7FD39A` / `#15301F` | Đạt |
| fail | chữ `#A3261B` / nền `#FDECEA` | `#F2998F` / `#3A1B18` | Không đạt, đồng bộ lỗi |
| skip | chữ `#4F5661` / nền `#ECEEF1` | `#B4BAC4` / `#242A33` | Bỏ qua |
| source-changed | chữ `#7A4100` / nền `#FFF1DC` | `#F5BE7A` / `#3A2A12` | Nguồn đã đổi sau khi chấm, đồng bộ một phần |
| source-unavailable | chữ `#5B3A99` / nền `#F1EBFB` | `#C3A9F2` / `#2A2140` | Không xác minh được nguồn |
| source-legacy | chữ `#5B6470`, không nền | `#A3ABB8` | Kết quả cũ, chưa xác minh |
| danger | `#B42318` (chữ trắng 6.57:1) | `#C9372C` (chữ trắng 5.16:1) | Nút xác nhận thao tác phá hủy |

Mọi cặp chữ/nền trạng thái đạt ≥ 5.7:1 ở light và ≥ 7.2:1 ở dark (đã tính theo WCAG). "Nguồn đã đổi" (cam) và "không đạt" (đỏ) có độ sáng gần như bằng nhau, nên **không** phân biệt bằng màu: chip kết luận là kiểu nền đặc, còn chip trạng thái nguồn là kiểu viền kèm biểu tượng tam giác cảnh báo. Mọi chip trạng thái đều có biểu tượng và chữ.

### 2.2 Chữ

- **Font: Be Vietnam Pro** (400/500/600/700), thiết kế cho tiếng Việt, dấu rõ ở cỡ nhỏ. **Tự host trong frontend**, không tải từ Google Fonts, để chạy được khi triển khai nội bộ không có internet. Thêm gói font vào frontend là một thay đổi dependency và phải ghi trong SPEC `CCMAI-UX-000`.
- **Thang chữ:** 12 (chú thích) · 13 (bảng, chip) · 14 (thân) · 16 (tiêu đề khối) · 20 (tiêu đề phụ) · 24 (tiêu đề trang mobile) · 30 (tiêu đề trang desktop). Số liệu dùng chữ số đều bề (`tabular-nums`).

### 2.3 Dark mode

**Hỗ trợ ngang hàng light.** Ứng dụng đã có nút chuyển. Mọi token có giá trị dark ở bảng trên, và mỗi màn triển khai phải chụp ảnh cả hai chế độ.

### 2.4 Bố cục và thành phần

- Lưới 8px; bo góc 8 (nút, chip), 12 (thẻ), 16 (khung lớn); bóng rất nhẹ, ưu tiên viền.
- Desktop: thanh bên có thể thu gọn, vùng nội dung tối đa 1440px. Mobile: thanh trên + menu trượt, mọi danh sách là thẻ một cột, vùng bấm ≥ 44px.
- Thao tác phá hủy (xóa kết quả, xóa kênh) nằm trong menu ⋯ và luôn có hộp xác nhận.
- Hộp thoại luôn có nút đóng ở góc, đóng được bằng Esc.
- Biểu tượng: giữ Material Design Icons (đã có trong Vuetify), kiểu outline, cỡ 16/20/24.
- Giữ Vuetify 4: hiện thực bằng theme, `defaults` và component dùng chung, không viết lại framework.

### 2.5 Mẫu tương tác mới (chỉ dùng dữ liệu API hiện có)

1. Bộ lọc **"Cần xem lại"** trên Kết quả và Job Detail.
2. Chi tiết hội thoại hai cột, bấm vấn đề thì tô trích dẫn trong chat.
3. Thẻ số liệu và biểu đồ ở Trang chủ, Job Detail dẫn tới danh sách đã lọc.
4. Tin nhắn ba cột trên desktop.
5. Nhãn "Nhận xét do AI tạo" trên mọi nhận xét và kết luận của AI.

### 2.6 Chưa làm (cần backend, thuộc roadmap runtime)

Người sửa điểm AI, khiếu nại điểm, hiệu chỉnh giữa người chấm, giao việc xem lại, bảng xếp hạng/huấn luyện theo nhân viên. Thiết kế **không** vẽ các tính năng này như thể đã có.

## 3. Canvas nền tảng

Bảng màu, thang chữ, bảng thuật ngữ và bộ component dùng chung (light và dark) được vẽ trên canvas claude.ai: https://claude.ai/artifact/DsSy7rS8zkAr4Gk6DB6vxt (riêng tư, của owner). Canvas và văn bản này là một cặp: đổi một bên phải cập nhật bên kia.

## 4. Hệ quả

- `CCMAI-UX-000` hiện thực bảng token (light/dark), font tự host, component dùng chung và bảng thuật ngữ theo quyết định này.
- Canvas thiết kế các màn sau dùng đúng bảng màu và chữ ở trên. Bản thử Job Detail (nền ấm) sẽ chuyển sang nền trung tính lạnh.
- Quyết định có thể đổi, nhưng phải cập nhật văn bản này và ghi lý do.

## Nguồn tham khảo

- [Zendesk — 10 best AI quality assurance software for customer service in 2026](https://www.zendesk.com/service/quality-assurance/customer-service-quality-assurance-software/)
- [Intryc — Best AI QA Software for Customer Support (2026 Buyer's Guide)](https://www.intryc.com/blog/best-ai-qa-software-for-customer-support-2026-buyers-guide)
- [Kaizo — Best Call Center QA Software 2026](https://kaizo.com/blog/customer-service-quality-assurance-software/)
- [Level AI — Quality assurance for contact centers](https://thelevel.ai/product/quality-assurance-contact-center)
- [Observe.AI — Auto QA](https://www.observe.ai/post-interaction/auto-qa)
- [UXSnaps — 5 UX lessons from Intercom's support inbox UI](https://www.uxsnaps.com/customer-support-inbox)
- [Help Scout vs Front vs Hiver 2026](https://www.aicofounderstack.com/2026/08/23/help-scout-vs-front-vs-hiver-2026/)

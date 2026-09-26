# Roadmap kiểm soát AI runtime và chi phí

**Trạng thái:** DRAFT / REVIEW_PENDING. **Ngày:** 2026-09-27. **Work order lập kế hoạch:** `CCMAI-ROADMAP-001`. **Rủi ro dự kiến khi triển khai:** tối thiểu R2 cho dữ liệu khách hàng, provider/network và governance runtime. Roadmap này không cấp quyền BUILD các tranche sản phẩm, gọi provider, đưa dữ liệu ra ngoài, triển khai production hoặc tuyên bố CVF đã kiểm soát AI runtime.

## Mục tiêu

Chỉ dùng AI bên ngoài khi hội thoại cần phán đoán ngữ nghĩa hoặc lời nhận xét mà quy tắc xác định không xử lý được. Mỗi kết quả phải giữ được nguồn, phiên bản quy tắc, quyết định điều phối, chi phí và trạng thái kiểm tra của con người. Tối ưu **tổng chi phí cho một kết quả đúng và hữu ích**, không chỉ số lần gọi LLM.

Luồng đích:

```text
adapter kênh -> hội thoại/tin nhắn đã lưu
  -> kiểm dữ liệu, quyền, phạm vi và chính sách riêng tư
  -> gate xác định tại máy (NO_AI / RULES_ONLY / cần đánh giá tiếp)
  -> [tùy chọn: phân loại ngữ nghĩa kiểu Jev, chỉ sau pilot]
  -> ngân sách + chọn provider/model -> LLM khi cần
  -> kiểm schema, liên kết bằng chứng và lưu proposal
  -> người có quyền review/confirm/correct -> báo cáo/thông báo
```

`NO_AI` và `RULES_ONLY` là **chế độ xử lý**, không phải tên một model. Gate xác định có thể dùng quy tắc, từ điển, trạng thái đồng bộ và dữ liệu có cấu trúc mà không cần API AI. Jev là một **dịch vụ model bên ngoài** nếu được chọn; dù không tạo văn xuôi, nó vẫn có input, độ trễ, chi phí, sai số và ranh giới dữ liệu riêng.

## Hiện trạng phải giữ đúng

| Bề mặt | Có trong mã hiện tại | Khoảng trống |
|---|---|---|
| Kênh đầu vào | Zalo OA, Facebook và Pancake qua `ChannelAdapter` (`backend/channels/adapter.go`, `registry.go`). | Không quyết định chính sách AI; đồng bộ lỗi từng hội thoại vẫn có thể bị bỏ qua rồi ghi kênh là success (`backend/engine/sync.go`). |
| Công việc AI | `Job` chọn kênh, QC hoặc classification, quy tắc, provider/model (`backend/db/models/job.go`). | `skip_conditions` được chèn vào prompt QC (`backend/ai/prompts.go`), chưa phải gate xác định trước khi gọi AI. |
| Gọi provider | `Analyzer` tạo transcript/prompt rồi gọi `AnalyzeChat` hoặc batch; hỗ trợ Claude, Gemini, OpenAI, xAI (`backend/engine/analyzer.go`, `backend/ai/provider.go`). | Chưa có điểm admission chung để kiểm quyền, dữ liệu nhạy cảm, quyết định NO_AI/RULES_ONLY và ngân sách trước mọi lượt gọi. |
| Kiểm đầu ra | Có kiểm JSON/giá trị và liên kết batch với conversation ID (`backend/engine/result_validation.go`). | `evidence` mới được kiểm có chữ, chưa đối chiếu span/ID tin nhắn và version nguồn; chưa có lifecycle proposal → human confirmed. |
| Chi phí | Có log token/model và tính chi phí (`backend/engine/analyzer.go`). | Chưa có reservation/budget gate; giá không biết được lưu thành `CostUSD=0`, gây hiểu nhầm trong tổng UI. |
| CVF và Shift | CVF quản lý thay đổi repo; Shift có nền tảng `NO_AI`/`RULES_ONLY`/`EXTERNAL_AI` trong `packages/ai-providers`. | Nền tảng đó chưa được nối vào caller ứng dụng này; chưa có bằng chứng provider thật rằng CVF chặn/điều phối luồng AI của ứng dụng. |

Nguồn hiện trạng: `docs/PRODUCT_DIRECTION.md`, `IMPLEMENTATION_STATUS.json`, mã được nêu trong bảng và `../shift-operations-workspace/packages/ai-providers/README.md`. Đây là rà soát source, không phải phép đo E2E.

## Nguyên tắc thiết kế và giới hạn

1. **Máy giữ quyền quyết định cuối:** chính sách quyền, dữ liệu nhạy cảm, ngân sách, nhánh xử lý và ngưỡng phê duyệt được version hóa trong code/config có review. Điểm số model chỉ là đầu vào cho nhánh đó; không tự cấp quyền hoặc xác nhận kết quả.
2. **Không bỏ sót ca quan trọng để tiết kiệm:** khi thiếu dữ liệu, không chắc nhãn, có khiếu nại, nguy cơ pháp lý/tài chính, hoặc nghi vấn ảnh hưởng đến nhân viên/khách hàng, không tự động đánh dấu “không cần xử lý”. Chuyển người phụ trách hoặc LLM theo chính sách đã duyệt; chặn gửi dữ liệu ngoài khi chưa được phép.
3. **Đo cả hai phía của gate:** ghi số hội thoại đi mỗi nhánh, false negative/false positive trên tập gán nhãn, token, phí của **gate + LLM**, độ trễ p50/p95, thời gian review của người, lỗi/retry và tỷ lệ proposal được chấp nhận. Giá chưa biết phải mang trạng thái `UNKNOWN`, không cộng như 0 thật.
4. **Bằng chứng nguồn:** kết quả AI phải trỏ đến `conversation_id`, message ID/span, source version/digest, rule/prompt version, model/provider, quyết định route và receipt chi phí. Không coi chuỗi trích dẫn do model viết là bằng chứng đã đối chiếu.
5. **Giữ dữ liệu tối thiểu:** xác định trường được đưa sang từng provider, thời gian lưu, xử lý PII và quyền xem receipt; không gửi toàn bộ hội thoại cho gate ngữ nghĩa nếu câu hỏi chỉ cần một đoạn đã được phép dùng.

## Trình tự tranche đề xuất

### S0 — Baseline và tập đánh giá được phép dùng

Lập inventory luồng `sync -> job -> provider -> result -> notification`; chốt nguồn dữ liệu pilot, quyền dùng dữ liệu, nhãn QC/classification, ca âm và ca rủi ro cao. Đo baseline số call, token, phí thật/UNKNOWN, latency và thời gian người review. Khóa tiêu chí chấp nhận, ngưỡng false negative, phương pháp so sánh và rollback **trước** pilot. Nếu chưa có dữ liệu được phép dùng, chỉ làm thiết kế/synthetic; không gửi dữ liệu khách hàng tới Jev hoặc LLM.

**Exit:** có corpus/version và danh sách lỗi cần bắt, kể cả chat trùng, chat rỗng, thiếu lịch sử, khiếu nại, prompt injection, PII và trường hợp nguồn bị sửa sau phân tích. Chưa claim cải thiện chi phí hay độ chính xác.

### S1 — Độ tin cậy dữ liệu và hợp đồng kết quả

Sửa checkpoint đồng bộ: lỗi từng hội thoại/tin nhắn không thể nâng mốc thành công toàn kênh; retry/replay có idempotency và trạng thái `partial`. Định nghĩa hợp đồng QC/classification version hóa, liên kết từng finding với message ID/span và xác minh trích dẫn trên snapshot nguồn trước khi lưu proposal. Run/item lưu trạng thái lỗi trung thực; transactional write không làm mất kết quả khác.

**Exit:** test lỗi từng bước, replay, duplicate, stale source, batch thiếu/thừa/sai ID và kết quả thiếu evidence. Từng thất bại có trạng thái quan sát được; không có false success.

### S2 — Gate xác định tại máy trước AI

Tạo một entry point dùng chung cho mọi job path (đơn/batch, thủ công/theo lịch, chạy lại). Trước provider call: kiểm quyền và phạm vi, dữ liệu đủ/đúng, trùng lặp, điều kiện loại trừ chắc chắn, chính sách PII và trạng thái nguồn. Trả một quyết định có version: `NO_AI`, `RULES_ONLY`, `NEEDS_SEMANTIC_REVIEW`, `NEEDS_LLM`, `HUMAN_REVIEW`, hoặc `DENY`. `NO_AI` dành cho item không đủ điều kiện chạy; `RULES_ONLY` chỉ cho kết quả được hợp đồng xác định cho phép. Điều kiện bỏ qua đang nằm trong prompt chỉ chuyển vào gate nếu có thể định nghĩa và kiểm thử xác định; phần mơ hồ vẫn không được tự bỏ.

**Exit:** shadow run trước; đối chiếu mọi quyết định với baseline và tập gán nhãn. Không bật auto-skip cho ca rủi ro cao. Receipt ghi policy/version/reason an toàn, số provider call quan sát được và quyền override của người.

### S3 — Admission provider, ngân sách và audit

Đặt một điểm dispatch duy nhất sau gate cho Claude/Gemini/OpenAI/xAI: allowlist provider/model, hạn mức theo job/run/workspace, reserve trước call và settle sau usage, idempotency/reconciliation khi timeout hoặc crash, `KNOWN/UNKNOWN/PENDING` cho giá. Ghi audit bất biến cho rule, route, provider, version, human override và correction; giữ receipt đã làm sạch secret/PII. Không cho fallback từ provider lỗi sang provider khác nếu chưa có chính sách và ngân sách rõ.

**Exit:** refusal trước dispatch quan sát zero provider call; nhánh được chấp nhận có một call thật và request/response sanitized trong evidence artifact. Rà soát độc lập R2 và real-provider proof là điều kiện cho mọi claim CVF governance runtime.

### S4 — Pilot Jev như tầng phân loại ngữ nghĩa tùy chọn

Thử các câu hỏi hẹp dạng **Choice** (loại yêu cầu), **Score** (mức độ cần phân tích) và **Noul** (xác suất một điều kiện), trên dữ liệu đã được phép xử lý. Code áp chính sách route; Jev không tạo nhận xét QC tự do, không xác nhận vi phạm và không thay review. So sánh ba nhánh trên cùng corpus: (A) gate xác định + LLM hiện tại, (B) gate xác định + Jev + LLM khi cần, (C) gate xác định + bộ phân loại/rule local khác nếu phù hợp. Threshold chọn từ dữ liệu thực tế, không sao chép từ ví dụ vendor. Shadow mode lưu nhãn và chi phí nhưng không thay kết quả production.

**Exit:** chứng minh trên holdout tiếng Việt và từng nhóm rủi ro rằng phương án B đạt tiêu chí chất lượng đã khóa ở S0, giảm **tổng** chi phí/độ trễ theo mục tiêu đã khóa, không tạo đường bypass quyền/PII và có kế hoạch khi Jev lỗi hoặc confidence thấp. Nếu không đạt, giữ Jev ở trạng thái DEFER/REJECT; không tích hợp chỉ vì model không sinh văn bản.

### S5 — Review, correction và đầu ra có trách nhiệm

Màn hình proposal có nguồn, trích dẫn, model, phiên bản rule, quyết định gate, chi phí và độ chắc chắn nếu có. Người được phân quyền xác nhận/từ chối/sửa, ghi actor/time/reason và before/after; dashboard và notification phân biệt proposal với confirmed. Gửi ra ngoài qua outbox theo từng người nhận, có retry/idempotency và không gửi dữ liệu nhạy cảm mặc định.

**Exit:** không có AI output tự biến thành quyết định chính thức; sửa nguồn sau proposal làm evidence stale; thử quyền xem, phê duyệt, correction và notification lỗi từng đích.

### S6 — Pilot và quyết định mở rộng

Chạy shadow rồi pilot giới hạn theo kênh/job với người review; đối chiếu số liệu trước/sau trên cùng phân bố dữ liệu. Tách lỗi gate, lỗi model, lỗi nguồn và lỗi lưu/truyền; kiểm chứng rollback về baseline. Release/production có work order, review và live evidence riêng.

**Exit:** báo cáo lợi ích ròng, false negative theo nhóm, chi phí `gate + LLM + retry + human`, latency, chất lượng dẫn chứng và quyết định GO/REVISE/STOP của owner. Không dùng số liệu demo hoặc mock làm bằng chứng governance hay tiết kiệm thực.

### S7 — Vận hành và phát hành có kiểm chứng

Sau khi pilot được chấp nhận, rà soát toàn bộ hướng dẫn CQA kế thừa theo luồng thật; kiểm tra restore backup, retention, quyền người dùng, audit truy xuất, secret rotation, dependency vulnerabilities, độ bền dữ liệu khi nâng cấp và giới hạn tải. Thiết kế release pipeline/image của sản phẩm riêng; không tái dùng đường phát hành CQA cũ. Mọi cam kết production, khả năng mở rộng hoặc chi phí phải dựa trên phép đo của bản này.

**Exit:** independent review của tài liệu cài đặt/vận hành, thử backup-restore và rollback trên môi trường tách biệt, kiểm tra bảo mật và release gate có bằng chứng provider thật cho các claim governance runtime. Việc chấp nhận S6 không tự động cấp quyền triển khai S7.

## Jev: nguồn tham khảo và giới hạn claim

- [TypeSafe Introduction](https://docs.typesafe.ai/introduction) mô tả Jev nhận state và câu hỏi có kiểu, trả `Choice`, `Score`, `Noul` cùng xác suất thay vì tạo văn xuôi. Điều đó **không** chứng minh câu trả lời luôn đúng hoặc chi phí luôn thấp hơn mọi LLM.
- [Intent routing](https://docs.typesafe.ai/patterns/intent-routing) dùng ví dụ CSKH để điều phối giữa logic xác định, LLM và con người. [Classifying RAG passages](https://docs.typesafe.ai/cookbooks/classifying_rag_passages) minh họa lọc bằng quyết định có kiểu trước model tạo nội dung. Đây là mẫu thiết kế, không là benchmark cho dữ liệu hội thoại của dự án.
- [Jev jaggedness](https://docs.typesafe.ai/model-jaggedness/jev-1.13) và [confidence guidance](https://docs.typesafe.ai/confidence) nhắc phải đánh giá lỗi, độ chắc chắn và dữ liệu đích. Pilot cần kiểm quyền dữ liệu, điều khoản API, giới hạn hiện hành và giá tại thời điểm mở tranche; roadmap không khóa một mức giá hoặc phiên bản model.

## Cổng quản trị

Mỗi S0–S7 mở bằng INTAKE → DESIGN → SPEC → WORK_ORDER → BUILD → REVIEW → FREEZE hoặc kế thừa bằng chứng được ghi rõ ở giai đoạn sớm nhất còn mở. R2 cần reviewer độc lập, scope/path/effect/credential cụ thể và rollback. Bất kỳ claim nào rằng CVF phân loại rủi ro, lọc dữ liệu, chặn call, route provider, validate output hoặc audit AI tại runtime đều cần gọi provider API thật và lưu request/response đã làm sạch theo `AGENTS.md`; mock chỉ dùng cho UI structure. Roadmap hiện tại và static checks không đáp ứng cổng đó.

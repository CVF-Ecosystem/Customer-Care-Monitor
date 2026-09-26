# Roadmap Source of Truth, machine gate và chi phí AI runtime

**Trạng thái:** DRAFT / REVIEW_PENDING. **Ngày:** 2026-09-27. **Work order lập kế hoạch:** `CCMAI-ROADMAP-001`. **Rủi ro dự kiến khi triển khai:** tối thiểu R2 cho dữ liệu khách hàng, provider/network và governance runtime. Roadmap này không cấp quyền BUILD các tranche sản phẩm, gọi provider, đưa dữ liệu ra ngoài, triển khai production hoặc tuyên bố CVF đã kiểm soát AI runtime.

## Mục tiêu

Source of Truth (SoT) và gate tại máy phải quyết định trước khi gọi bất kỳ Agent/AI/LLM nào. Chỉ dùng Claude/Gemini/OpenAI/xAI khi dữ liệu nguồn đáng tin, quyền và ngân sách cho phép, và tác vụ thực sự cần phân tích ngữ nghĩa sâu hoặc tạo nhận xét mà quy tắc tại máy không giải quyết được. Mỗi kết quả phải giữ được nguồn, phiên bản quy tắc, quyết định điều phối, chi phí và trạng thái kiểm tra của con người. Tối ưu **tổng chi phí cho một kết quả đúng và hữu ích**, không chỉ số lần gọi LLM.

Luồng đích:

```text
adapter kênh -> raw evidence -> hội thoại/tin nhắn chuẩn hóa, có phiên bản
  -> SoT gate: nguồn, provenance, trạng thái, quyền, phạm vi, dữ liệu đủ và chính sách riêng tư
  -> machine gate tại máy: quy tắc xác định, chống trùng, phân loại có kiểu
  -> NO_AI / RULES_ONLY / HUMAN_REVIEW / NEEDS_LLM / DENY
  -> chỉ nhánh NEEDS_LLM: ngân sách + chọn provider/model -> LLM
  -> kiểm schema, liên kết bằng chứng và lưu proposal
  -> người có quyền review/confirm/correct -> báo cáo/thông báo
```

`NO_AI` và `RULES_ONLY` là **chế độ xử lý**, không phải tên một model. Gate xác định dùng dữ liệu SoT, quy tắc, từ điển, trạng thái đồng bộ và phép kiểm có cấu trúc tại máy; nó không gọi Jev, Agent, AI hay LLM. Từ skill TypeSafe/Jev, roadmap chỉ học cách chia bài toán thành quyết định hẹp, kết quả có kiểu, nhánh `unknown` và cách ghép quyết định bằng code. Không thêm Jev API/SDK, credential hoặc dịch vụ trung gian vào kiến trúc đích.

## Ranh giới Source of Truth

- Payload từ kênh là **raw evidence**; bản chuẩn hóa đã lưu kèm nguồn, timestamp, version/digest và lịch sử đồng bộ là nguồn để đánh giá. Dữ liệu thiếu, lỗi đồng bộ hoặc provenance không rõ không được biến thành kết luận chắc chắn.
- Chính sách và quy tắc đã duyệt có version là nguồn thẩm quyền cho quyền truy cập, loại trừ, phân loại xác định, ngân sách và mức cần người duyệt. Một nhãn do AI suy ra không thể ghi đè chính sách.
- Kết quả `RULES_ONLY` phải nêu rule ID/version, điều kiện khớp và message/source refs. Phản hồi LLM là **proposal**; chỉ human disposition có thẩm quyền mới tạo trạng thái confirmed. Khi nguồn hoặc rule đổi, kết quả phụ thuộc phải được đánh dấu stale để xét lại.
- Gate được phép trả `UNKNOWN` hoặc `HUMAN_REVIEW`. Không tự tạo xác suất hoặc nhãn “độ tin cậy” từ quy tắc xác định; chỉ dùng xác suất khi có một bộ phân loại đã được hiệu chuẩn và đo trên dữ liệu đích.

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

1. **SoT và máy giữ quyền điều phối:** chính sách quyền, dữ liệu nhạy cảm, ngân sách, nhánh xử lý và ngưỡng phê duyệt được version hóa trong code/config có review. Phản hồi model ở nhánh sau chỉ là đầu vào; không tự cấp quyền hoặc xác nhận kết quả.
2. **Không bỏ sót ca quan trọng để tiết kiệm:** khi thiếu dữ liệu, không chắc nhãn, có khiếu nại, nguy cơ pháp lý/tài chính, hoặc nghi vấn ảnh hưởng đến nhân viên/khách hàng, không tự động đánh dấu “không cần xử lý”. Chuyển người phụ trách hoặc LLM theo chính sách đã duyệt; chặn gửi dữ liệu ngoài khi chưa được phép.
3. **Đo cả hai phía của gate:** ghi số hội thoại đi mỗi nhánh, false negative/false positive trên tập gán nhãn, CPU/thời gian gate tại máy, token và phí của **những LLM thật sự được gọi**, độ trễ p50/p95, thời gian review của người, lỗi/retry và tỷ lệ proposal được chấp nhận. Giá chưa biết phải mang trạng thái `UNKNOWN`, không cộng như 0 thật.
4. **Bằng chứng nguồn:** kết quả AI phải trỏ đến `conversation_id`, message ID/span, source version/digest, rule/prompt version, model/provider, quyết định route và receipt chi phí. Không coi chuỗi trích dẫn do model viết là bằng chứng đã đối chiếu.
5. **Giữ dữ liệu tối thiểu:** machine gate xử lý trong ranh giới ứng dụng. Chỉ nhánh LLM được phép mới chuẩn bị phần hội thoại cần thiết, sau kiểm PII, thời gian lưu và quyền xem receipt.

## Trình tự tranche đề xuất

### S0 — Baseline và tập đánh giá được phép dùng

Lập inventory luồng `sync -> job -> provider -> result -> notification`, các SoT hiện có và nơi nguồn có thể bị sai/stale; chốt dữ liệu pilot được phép dùng, nhãn QC/classification, ca âm và ca rủi ro cao. Đo baseline số call Agent/AI/LLM, token, phí thật/UNKNOWN, latency và thời gian người review. Khóa tiêu chí chấp nhận, ngưỡng false negative, phương pháp so sánh và rollback **trước** pilot. Nếu chưa có dữ liệu được phép dùng, chỉ làm thiết kế/synthetic; không gửi dữ liệu khách hàng tới bất kỳ provider nào.

**Exit:** có corpus/version và danh sách lỗi cần bắt, kể cả chat trùng, chat rỗng, thiếu lịch sử, khiếu nại, prompt injection, PII và trường hợp nguồn bị sửa sau phân tích. Chưa claim cải thiện chi phí hay độ chính xác.

### S1 — Độ tin cậy dữ liệu và hợp đồng kết quả

Sửa checkpoint đồng bộ: lỗi từng hội thoại/tin nhắn không thể nâng mốc thành công toàn kênh; retry/replay có idempotency và trạng thái `partial`. Định nghĩa hợp đồng QC/classification version hóa, liên kết từng finding với message ID/span và xác minh trích dẫn trên snapshot nguồn trước khi lưu proposal. Run/item lưu trạng thái lỗi trung thực; transactional write không làm mất kết quả khác.

**Exit:** test lỗi từng bước, replay, duplicate, stale source, batch thiếu/thừa/sai ID và kết quả thiếu evidence. Từng thất bại có trạng thái quan sát được; không có false success.

### S2 — Gate xác định tại máy trước AI

Tạo một entry point dùng chung cho mọi job path (đơn/batch, thủ công/theo lịch, chạy lại). Đọc SoT đã version hóa rồi kiểm nguồn/provenance, quyền và phạm vi, dữ liệu đủ/đúng, trùng lặp, điều kiện loại trừ chắc chắn, PII và trạng thái đồng bộ. Thiết kế các câu hỏi nội bộ hẹp có đáp án hữu hạn như `có_tin_mới?`, `dữ_liệu_đủ?`, `thuộc_loại_quy_tắc_nào?`, `cần_người_xem?`; trả kết quả có kiểu, lý do và source refs. Nhánh cuối có version: `NO_AI`, `RULES_ONLY`, `NEEDS_LLM`, `HUMAN_REVIEW`, hoặc `DENY`. `NO_AI` dành cho item không đủ điều kiện chạy; `RULES_ONLY` chỉ cho kết quả mà quy tắc xác định và SoT đủ chứng minh. Không biến nhãn `UNKNOWN` thành auto-skip. Điều kiện bỏ qua đang nằm trong prompt chỉ chuyển vào gate nếu có thể định nghĩa và kiểm thử xác định; phần mơ hồ vẫn không được tự bỏ.

**Exit:** shadow run trước; đối chiếu mọi quyết định với SoT và tập gán nhãn. Không bật auto-skip cho ca rủi ro cao. Receipt ghi source digest, policy/rule version, reason an toàn, nhánh quyết định và quyền override của người. Đo zero external call ở các nhánh local; không suy từ code đơn lẻ rằng mọi đường chạy đều đã được chặn.

### S3 — Admission provider, ngân sách và audit

Đặt một điểm dispatch duy nhất sau gate cho Claude/Gemini/OpenAI/xAI: allowlist provider/model, hạn mức theo job/run/workspace, reserve trước call và settle sau usage, idempotency/reconciliation khi timeout hoặc crash, `KNOWN/UNKNOWN/PENDING` cho giá. Ghi audit bất biến cho rule, route, provider, version, human override và correction; giữ receipt đã làm sạch secret/PII. Không cho fallback từ provider lỗi sang provider khác nếu chưa có chính sách và ngân sách rõ.

**Exit:** refusal trước dispatch quan sát zero provider call; nhánh được chấp nhận có một call thật và request/response sanitized trong evidence artifact. Rà soát độc lập R2 và real-provider proof là điều kiện cho mọi claim CVF governance runtime.

### S4 — Học mẫu thiết kế quyết định có kiểu và tối ưu gate tại máy

Rà [skill TypeSafe/Jev](https://github.com/typesafe-ai/skills/blob/main/skills/typesafe-ai/SKILL.md) như nguồn **phương pháp**: tách câu hỏi hẹp, định nghĩa lựa chọn đầy đủ với `unknown/no-match`, giữ quy tắc và phép tính trong code, nối các kết quả có kiểu thành nhánh xử lý, giữ provenance và chuyển ca không chắc cho người. Thiết kế lại thành `DecisionQuestion`, `DecisionAnswer`, `DecisionTrace` nội bộ; có thể dùng enum, boolean ba trạng thái và mức ưu tiên xác định từ SoT/rule. Không sao chép giả định rằng kết quả local có xác suất đã hiệu chuẩn. So sánh trên cùng corpus: (A) luồng hiện tại gọi LLM, (B) SoT + gate local + LLM chỉ khi cần, (C) biến thể local rule/index nếu B chưa đủ tốt. Chạy shadow trước, không thêm Jev API/SDK hoặc provider phân loại trung gian.

**Exit:** chứng minh trên holdout tiếng Việt và từng nhóm rủi ro rằng phương án B đạt tiêu chí chất lượng đã khóa ở S0, giảm số call Agent/AI/LLM và tổng chi phí theo mục tiêu đã khóa, không tạo đường bypass quyền/PII. Item `UNKNOWN`, xung đột nguồn hoặc ngoài coverage có nhánh người/LLM được kiểm soát; nếu gate local không đạt, sửa rule/SoT hoặc giữ luồng cũ thay vì âm thầm bỏ item.

### S5 — Review, correction và đầu ra có trách nhiệm

Màn hình proposal có nguồn, trích dẫn, model, phiên bản rule, quyết định gate, chi phí và độ chắc chắn nếu có. Người được phân quyền xác nhận/từ chối/sửa, ghi actor/time/reason và before/after; dashboard và notification phân biệt proposal với confirmed. Gửi ra ngoài qua outbox theo từng người nhận, có retry/idempotency và không gửi dữ liệu nhạy cảm mặc định.

**Exit:** không có AI output tự biến thành quyết định chính thức; sửa nguồn sau proposal làm evidence stale; thử quyền xem, phê duyệt, correction và notification lỗi từng đích.

### S6 — Pilot và quyết định mở rộng

Chạy shadow rồi pilot giới hạn theo kênh/job với người review; đối chiếu số liệu trước/sau trên cùng phân bố dữ liệu. Tách lỗi gate, lỗi model, lỗi nguồn và lỗi lưu/truyền; kiểm chứng rollback về baseline. Release/production có work order, review và live evidence riêng.

**Exit:** báo cáo lợi ích ròng, false negative theo nhóm, chi phí `local gate + LLM đã gọi + retry + human`, latency, chất lượng dẫn chứng và quyết định GO/REVISE/STOP của owner. Không dùng số liệu demo hoặc mock làm bằng chứng governance hay tiết kiệm thực.

### S7 — Vận hành và phát hành có kiểm chứng

Sau khi pilot được chấp nhận, rà soát toàn bộ hướng dẫn CQA kế thừa theo luồng thật; kiểm tra restore backup, retention, quyền người dùng, audit truy xuất, secret rotation, dependency vulnerabilities, độ bền dữ liệu khi nâng cấp và giới hạn tải. Thiết kế release pipeline/image của sản phẩm riêng; không tái dùng đường phát hành CQA cũ. Mọi cam kết production, khả năng mở rộng hoặc chi phí phải dựa trên phép đo của bản này.

**Exit:** independent review của tài liệu cài đặt/vận hành, thử backup-restore và rollback trên môi trường tách biệt, kiểm tra bảo mật và release gate có bằng chứng provider thật cho các claim governance runtime. Việc chấp nhận S6 không tự động cấp quyền triển khai S7.

## TypeSafe/Jev: học kỹ năng thiết kế, không dùng dịch vụ

- [Skill chính thức của TypeSafe](https://github.com/typesafe-ai/skills/blob/main/skills/typesafe-ai/SKILL.md) nêu “code owns the workflow”, giữ quy tắc/phép tính/lookups xác định trong code, chia câu hỏi thành phần nhỏ, có lựa chọn `no-match`, rồi kết hợp và xác minh kết quả. Đây là nguyên tắc tham khảo để thiết kế **module nội bộ** của dự án.
- [Intent routing](https://docs.typesafe.ai/patterns/intent-routing) cho thấy một quyết định có thể dẫn tới code xác định, LLM hoặc người. Ở dự án này SoT và quy tắc local phải chạy trước; không dùng TypeSafe/Jev để thực hiện bước phân loại đầu.
- [Confidence guidance](https://docs.typesafe.ai/confidence) là tài liệu cho output xác suất của model; không được gắn số confidence giả cho quyết định quy tắc. Nếu sau này có đề xuất dùng model phân loại, đó là thay đổi ranh giới dịch vụ mới cần INTAKE/work order và quyết định của owner riêng, không nằm trong roadmap đã duyệt ở đây.

## Cổng quản trị

Mỗi S0–S7 mở bằng INTAKE → DESIGN → SPEC → WORK_ORDER → BUILD → REVIEW → FREEZE hoặc kế thừa bằng chứng được ghi rõ ở giai đoạn sớm nhất còn mở. R2 cần reviewer độc lập, scope/path/effect/credential cụ thể và rollback. Bất kỳ claim nào rằng CVF phân loại rủi ro, lọc dữ liệu, chặn call, route provider, validate output hoặc audit AI tại runtime đều cần gọi provider API thật và lưu request/response đã làm sạch theo `AGENTS.md`; mock chỉ dùng cho UI structure. Roadmap hiện tại và static checks không đáp ứng cổng đó.

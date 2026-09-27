# Roadmap CSKH: dữ liệu đáng tin, bộ lọc hỗ trợ AI và xử lý có trách nhiệm

**Trạng thái:** DRAFT / REVIEW_PENDING. **Ngày:** 2026-09-27. **Work order lập kế hoạch:** `CCMAI-ROADMAP-001`. **Rủi ro dự kiến khi triển khai:** tối thiểu R2 cho dữ liệu khách hàng, provider/network và governance runtime. Roadmap này không cấp quyền BUILD các tranche sản phẩm, gọi provider, đưa dữ liệu ra ngoài, triển khai production hoặc tuyên bố CVF đã kiểm soát AI runtime.

**Quyết định thiết kế đi kèm:** [mẫu lọc dữ liệu SoT-first có thể tái sử dụng](../decisions/SOT_FIRST_DATA_FILTERING_PATTERN_2026-09-27.md). Mẫu nêu hợp đồng chung; roadmap này ánh xạ vào ứng dụng CSKH và chưa chứng minh một thư viện đa dự án đã vận hành.

## Mục tiêu

Giúp người phụ trách trả lời: **hội thoại nào cần can thiệp, vì sao, ai xử lý và đã giải quyết đến đâu?** Hoàn thiện một luồng CSKH từ nguồn tới hành động có bằng chứng là ưu tiên đầu. Khi dự án vận hành và được chấp nhận, kết quả mới trở thành use case để nhân rộng sang dự án khác, rồi cân nhắc đóng góp nâng nền CVF theo work order riêng.

SoT và gate tại máy kiểm nguồn, quyền, phạm vi và nhu cầu xử lý trước provider. Bộ lọc học từ skill/cách xử lý Jev chuẩn bị ngữ cảnh, câu hỏi hẹp và quyết định có kiểu; AI/LLM tiếp tục phân tích ngữ nghĩa và tạo nhận xét/phản hồi trên phần việc cần năng lực đó. Tối ưu **tổng chi phí cho một kết quả đúng và hữu ích**, với chất lượng và rủi ro khách hàng là điều kiện bắt buộc trước khi chấp nhận tiết kiệm. Không đặt mục tiêu cắt call bằng mọi giá hoặc coi LLM là dấu hiệu bộ lọc thất bại.

Luồng đích:

```text
adapter -> evidence -> snapshot có nguồn, vai trò, thời gian và trạng thái đầy đủ
  -> SoT/policy gate + bộ lọc nội bộ học từ Jev
  -> đủ điều kiện? / xử lý bằng gì? / cần ai duyệt?
     -> chờ dữ liệu, chặn theo policy, hoặc local có căn cứ
     -> cần ngữ nghĩa/nhận xét/phản hồi: admission -> AI/LLM
  -> kiểm output, liên kết evidence -> proposal
  -> hàng đợi ưu tiên -> người review/giao xử lý -> ghi kết quả/correction
```

`NO_AI` và `RULES_ONLY` chỉ áp dụng cho phần việc có thể kết thúc hợp lệ bằng local rules hoặc tái dùng kết quả còn hiệu lực. Chúng không thay thế nhu cầu AI/LLM của toàn sản phẩm. Jev đóng góp phương pháp thiết kế bộ lọc và xử lý quyết định; định hướng hiện tại vẫn là thiết kế lại trong ứng dụng, không thêm Jev API/SDK hay dịch vụ trung gian. Học skill không đồng nghĩa đã tái tạo model ngữ nghĩa Jev hoặc đạt hiệu năng của nó.

“Phản hồi” trong đợt thiết kế này là nhận xét/gợi ý cho người phụ trách; tự gửi câu trả lời tới khách hàng cần SPEC và quyền phát hành riêng. Gate, AI và người duyệt phối hợp theo policy, không ghi đè kết luận của nhau âm thầm.

## Ranh giới Source of Truth

- Payload từ kênh là **raw evidence**; bản chuẩn hóa đã lưu kèm nguồn, timestamp, version/digest và lịch sử đồng bộ là nguồn để đánh giá. Dữ liệu thiếu, lỗi đồng bộ hoặc provenance không rõ không được biến thành kết luận chắc chắn.
- Chính sách và quy tắc đã duyệt có version là nguồn thẩm quyền cho quyền truy cập, loại trừ, phân loại xác định, ngân sách và mức cần người duyệt. Một nhãn do AI suy ra không thể ghi đè chính sách.
- Phân biệt sự kiện quan sát được, suy luận và đánh giá/quyết định được tổ chức chấp nhận. Tin nhắn “đã hoàn tiền” chứng minh lời nói đã được ghi nhận; xác nhận hoàn tiền cần nguồn giao dịch có thẩm quyền. Human confirmation ghi người chấp nhận đánh giá, không tự biến suy luận thành sự thật khách quan.
- Kết quả `RULES_ONLY` phải nêu rule ID/version, điều kiện khớp và message/source refs. Phản hồi LLM là **proposal**. Khi nguồn hoặc rule đổi, kết quả phụ thuộc phải được đánh dấu stale để xét lại. Dữ kiện quan sát được có thể lưu tự động; quyết định tác động khách hàng/nhân viên cần quyền duyệt tương ứng.
- Tách eligibility (`ELIGIBLE`, `WAIT_DATA`, `DENY`), execution (`NONE`, `RULES_ONLY`, `LLM`) và disposition (`PENDING`, `REVIEW_REQUIRED`, `ACCEPTED`, `REJECTED`, `CORRECTED`) trong SPEC. Đây là tên định hướng, chưa phải API. `UNKNOWN` là trạng thái hiểu biết, không phải auto-skip; rule local cũng có thể cần người duyệt. Chờ phải có lý do, owner, hạn xem lại và cảnh báo quá hạn. Không gán confidence giả hoặc mặc định 1.0 cho kết luận chưa đo.

## Hiện trạng phải giữ đúng

| Bề mặt | Có trong mã hiện tại | Khoảng trống |
|---|---|---|
| Database | MySQL 8 + GORM lưu channel, conversation, message, job/run/result, usage và `ccma.snapshot.v1` digest/coverage. Fresh install mặc định schema `CCMA`; bản cài cũ có thể giữ `DB_NAME=cqa`. | Tên schema không tạo SoT semantics; chưa có decision receipt cho gate. |
| Kênh đầu vào | Zalo OA, Facebook và Pancake qua `ChannelAdapter` (`backend/channels/adapter.go`, `registry.go`). Lỗi từng item tạo `partial` và giữ checkpoint thành công; replay cùng external message ID cập nhật các trường được cung cấp, có thay đổi. | Không quyết định chính sách AI; giới hạn lịch sử và tín hiệu xóa/sửa tùy adapter, chưa có real-channel validation cho replay. |
| Công việc AI | `Job` chọn kênh, QC hoặc classification, quy tắc, provider/model (`backend/db/models/job.go`). | `skip_conditions` được chèn vào prompt QC (`backend/ai/prompts.go`), chưa phải gate xác định trước khi gọi AI. |
| Gọi provider | `Analyzer` tạo transcript/prompt rồi gọi `AnalyzeChat` hoặc batch; hỗ trợ Claude, Gemini, OpenAI, xAI (`backend/engine/analyzer.go`, `backend/ai/provider.go`). | Chưa có điểm admission chung để kiểm quyền, dữ liệu nhạy cảm, quyết định NO_AI/RULES_ONLY và ngân sách trước mọi lượt gọi. |
| Kiểm đầu ra | Có kiểm JSON/giá trị, batch conversation ID, evidence refs đối chiếu message/span và snapshot digest (`backend/engine/result_validation.go`, `snapshot.go`). | Chưa có lifecycle proposal → human confirmed; chưa chứng minh độ đầy đủ của replay bằng real-channel evidence. |
| Chi phí | Có log token/model và tính chi phí (`backend/engine/analyzer.go`). | Chưa có reservation/budget gate; giá không biết được lưu thành `CostUSD=0`, gây hiểu nhầm trong tổng UI. |
| CVF và Shift | CVF quản lý thay đổi repo; Shift có nền tảng `NO_AI`/`RULES_ONLY`/`EXTERNAL_AI` trong `packages/ai-providers`. | Nền tảng đó chưa được nối vào caller ứng dụng này; chưa có bằng chứng provider thật rằng CVF chặn/điều phối luồng AI của ứng dụng. |

Rà soát source sau sáu tranche S1 đã qua REVIEW: `partial/error` không còn cập nhật `last_sync_at`; transcript snapshot có message ID, vai trò và timestamp RFC3339 kèm timezone. `CCMAI-RUNTIME-003` đã qua independent REVIEW cho replay cùng ID, bao gồm nội dung, vai trò, thời gian, raw data và attachment identity; không suy tín hiệu xóa thành delete. `CCMAI-RUNTIME-005` đã bỏ confidence QC giả định 1.0 cho kết quả mới và gắn nhãn số do model báo là chưa hiệu chuẩn; chưa có phép đo calibration. Giới hạn coverage của từng adapter cần được ghi rõ.

Nguồn hiện trạng: `docs/PRODUCT_DIRECTION.md`, `IMPLEMENTATION_STATUS.json`, mã được nêu trong bảng và `../shift-operations-workspace/packages/ai-providers/README.md`. Đây là rà soát source, không phải phép đo E2E.

## Nguyên tắc thiết kế và giới hạn

1. **SoT và máy giữ quyền điều phối:** chính sách quyền, dữ liệu nhạy cảm, ngân sách, nhánh xử lý và ngưỡng phê duyệt được version hóa trong code/config có review. Phản hồi model ở nhánh sau chỉ là đầu vào; không tự cấp quyền hoặc xác nhận kết quả.
2. **Không bỏ sót ca quan trọng để tiết kiệm:** khi thiếu dữ liệu, không chắc nhãn, có khiếu nại, nguy cơ pháp lý/tài chính, hoặc nghi vấn ảnh hưởng đến nhân viên/khách hàng, không tự động đánh dấu “không cần xử lý”. Chuyển người phụ trách hoặc LLM theo chính sách đã duyệt; chặn gửi dữ liệu ngoài khi chưa được phép.
3. **Đo cả hai phía của gate:** ghi số hội thoại đi mỗi nhánh, false negative/false positive trên tập gán nhãn, CPU/thời gian gate tại máy, token và phí của **những LLM thật sự được gọi**, độ trễ p50/p95, thời gian review của người, lỗi/retry và tỷ lệ proposal được chấp nhận. Giá chưa biết phải mang trạng thái `UNKNOWN`, không cộng như 0 thật.
4. **Bằng chứng nguồn:** kết quả AI phải trỏ đến `conversation_id`, message ID/span, source version/digest, rule/prompt version, model/provider, quyết định route và receipt chi phí. Không coi chuỗi trích dẫn do model viết là bằng chứng đã đối chiếu.
5. **Giữ dữ liệu tối thiểu:** machine gate xử lý trong ranh giới ứng dụng. Chỉ nhánh LLM được phép mới chuẩn bị phần hội thoại cần thiết, sau kiểm PII, thời gian lưu và quyền xem receipt.
6. **Tái dùng có ranh giới:** trạng thái evidence, provenance, kết quả gate có kiểu, trace, admission và receipt là ứng viên hợp đồng chung. Adapter, taxonomy QC, SoT owner, risk/PII policy, quyền duyệt và rule pack thuộc từng dự án. Chỉ tách thành thư viện chung sau khi có SPEC và bằng chứng trên một dự án thứ hai; không ghi vào CVF core từ work order này.
7. **Tiếng Việt và chất lượng là cổng bắt buộc:** kiểm phủ định, nói giảm/nói tránh, mỉa mai, đại từ xưng hô, không dấu, viết tắt, tiếng địa phương, xen ngôn ngữ và ngữ cảnh nhiều lượt. Ví dụ “chăm sóc tốt quá, nhắn ba hôm chưa ai trả lời” không được rule từ khóa “tốt” tự kết luận tích cực. Mẫu này là ca thử thiết kế; corpus thực tế quyết định coverage, không mặc định tiếng Việt luôn khó hơn tiếng Anh.
8. **Giữ đủ ngữ cảnh:** giảm payload phải giữ lượt liên quan, vai trò, ngày giờ, quan hệ trả lời và dấu hiệu thiếu ảnh/file/lịch sử. Khi không chắc, mở rộng ngữ cảnh, dùng LLM đủ năng lực theo policy hoặc chuyển người; không cắt cụt rồi báo PASS. Hết ngân sách thì giữ việc ở trạng thái chờ/escalate có người chịu trách nhiệm, không tự hạ chất lượng hay vượt ngân sách.
9. **Database giữ cấu trúc, ứng dụng giữ quyết định:** MySQL lọc tenant/channel/time/status và snapshot đã xử lý. Go dựng projection tối thiểu, kiểm provenance/policy/rule/admission rồi mới resolve provider. Không đặt semantic network call trong database hoặc phụ thuộc vào tên schema để suy authority.

## Trình tự tranche đề xuất

### Tiến độ triển khai

| Stage | Trạng thái | Bằng chứng |
|---|---|---|
| S0 | REVIEW_PASS_WITH_REPAIRS / FREEZE_OPEN | Corpus synthetic `s0-vi-intervention-v1`, baseline database và Gate A review tại `docs/reviews/CCMAI_RUNTIME_001_INDEPENDENT_REVIEW_2026-09-27.md`. Chưa có dữ liệu/call thật để đo chất lượng hoặc chi phí. |
| S1 | IN_PROGRESS | `CCMAI-RUNTIME-001` sync truth, `CCMAI-RUNTIME-002` snapshot/evidence, `CCMAI-RUNTIME-003` replay integrity, `CCMAI-RUNTIME-004` tín hiệu nguồn trên trang Kết quả/export, `CCMAI-RUNTIME-005` confidence trung thực, `CCMAI-RUNTIME-006` tín hiệu nguồn theo job, `CCMAI-RUNTIME-007` phản hồi bắt đầu đồng bộ thủ công và `CCMAI-RUNTIME-008` kiểm cấu hình trước đồng bộ thủ công đều đã qua independent REVIEW; cả tám còn FREEZE open. R006-R1 đã đưa giới hạn so sánh cục bộ lên bảng/thẻ Job Detail. R007 chỉ trả 202 sau khi ghi được trạng thái `syncing` theo tenant; R008 chỉ chạy khi cấu hình hợp lệ và truyền đúng cấu hình đó cho worker. `CCMAI-RUNTIME-009` đã BUILD xong (REVIEW_PENDING) cho lỗi `config.Load` trong job trigger/test-run: hai endpoint chỉ chạy khi cấu hình hợp lệ và truyền đúng cấu hình đó cho worker; OAuth/credential và agent vẫn cần scope riêng. Đồng bộ chồng lấn giữa các path và trạng thái `syncing` sau crash vẫn là residual S1. Các kiểm thử hiện dùng disposable MySQL và adapter fixtures, chưa có real-channel proof. |
| S2–S7 | NOT_STARTED | Chưa có runtime gate/provider admission/human disposition hoặc claim governance live. |

Giữ ID S0–S7 để truy vết, nhưng **ID không còn là thứ tự tuyến tính**: S0 → S1 → phần tối thiểu của S2 + S3 + S5 tạo một luồng hoàn chỉnh → S4 tối ưu trên phản hồi thực → S6 pilot → S7 hoàn thiện vận hành. S2/S3/S5 đều có scope/acceptance riêng; review UI và audit tối thiểu phải sẵn sàng trong luồng đầu, không đợi tối ưu xong. Kiểm quyền, bảo mật và bảo vệ dữ liệu cần thiết cho pilot phải hoàn tất trước S6; S7 mở rộng kiểm vận hành trước phát hành. Nhân rộng/CVF uplift đứng sau nghiệm thu sản phẩm.

### S0 — Baseline và tập đánh giá được phép dùng

Lập inventory luồng `sync -> job -> provider -> result -> notification`, các SoT hiện có và nơi nguồn có thể bị sai/stale; chốt dữ liệu pilot được phép dùng, nhãn QC/classification, ca âm và ca rủi ro cao. Đo baseline số call Agent/AI/LLM, token, phí thật/UNKNOWN, latency và thời gian người review. Khóa tiêu chí chấp nhận, ngưỡng false negative, phương pháp so sánh và rollback **trước** pilot. Nếu chưa có dữ liệu được phép dùng, chỉ làm thiết kế/synthetic; không gửi dữ liệu khách hàng tới bất kỳ provider nào.

Chọn một kênh và một use case can thiệp cụ thể cùng người dùng pilot; đo thêm thời gian tới hành động, số ca quan trọng được giải quyết, cảnh báo sai và backlog người review. Corpus tiếng Việt cần ca mơ hồ và nhãn có phân xử bất đồng; không lấy output model hoặc một lượt người duyệt làm ground truth mặc định. Mọi baseline call thật cần admission phù hợp trong work order đánh giá, kể cả trước khi S3 hoàn chỉnh.

Chốt bảng ánh xạ từ hợp đồng chung trong [quyết định SoT-first](../decisions/SOT_FIRST_DATA_FILTERING_PATTERN_2026-09-27.md) sang SoT owner, adapter, rule pack và quyền xác nhận riêng của Customer-Care-Monitor-AI; ghi phần nào cần thay đổi nếu áp dụng cho một dự án khác.

**Exit:** có corpus/version và danh sách lỗi cần bắt, kể cả chat trùng, chat rỗng, thiếu lịch sử, khiếu nại, prompt injection, PII và trường hợp nguồn bị sửa sau phân tích. Chưa claim cải thiện chi phí hay độ chính xác.

### S1 — Độ tin cậy dữ liệu và hợp đồng kết quả

Sửa checkpoint đồng bộ: lỗi từng hội thoại/tin nhắn không thể nâng mốc thành công toàn kênh; retry/replay có idempotency và trạng thái `partial`. Định nghĩa hợp đồng QC/classification version hóa, liên kết từng finding với message ID/span và xác minh trích dẫn trên snapshot nguồn trước khi lưu proposal. Run/item lưu trạng thái lỗi trung thực; transactional write không làm mất kết quả khác.

Bảo toàn timestamp đầy đủ, vai trò người gửi, tin sửa/xóa theo khả năng adapter và coverage của ảnh/file/lịch sử; snapshot không đủ phải hiển thị thiếu. Tách confidence chưa đo khỏi xác suất đã hiệu chuẩn. Kiểm mốc sync khi lỗi và ghi chú rõ khả năng nào upstream không cung cấp; không suy adapter thành nguồn hoàn chỉnh mặc định.

**Exit:** test lỗi từng bước, replay, duplicate, stale source, batch thiếu/thừa/sai ID và kết quả thiếu evidence. Từng thất bại có trạng thái quan sát được; không có false success.

### S2 — Gate xác định tại máy trước AI

Tạo một entry point dùng chung cho mọi job path (đơn/batch, thủ công/theo lịch, chạy lại). Kiểm nguồn, quyền, dữ liệu đủ, trùng lặp, PII và trạng thái đồng bộ; học cách chia câu hỏi hẹp, có kiểu và `unknown/no-match` từ Jev ngay tại S2. SPEC tách eligibility, execution, disposition; `WAIT_DATA` có owner/deadline/retry. Điều kiện bỏ qua từ prompt chỉ chuyển thành rule khi chứng minh được trên dữ liệu đích. Bộ lọc giữ ngữ cảnh cho AI/LLM xử lý nghĩa và tạo phản hồi; trường hợp không rõ không được tự skip.

Thứ tự kỹ thuật là SQL predicate rẻ → snapshot/projection có digest → gate/rule → provider admission. Di chuyển việc resolve provider/API key xuống sau quyết định cần LLM. Áp dụng bài học có chọn lọc từ [pg-jev](https://github.com/Blackbird081/pg-jev): chỉ gửi trường cần thiết, batch/concurrency có trần, cache theo content + question/rule/model version, max rows/chars và usage stats. Triển khai tại Go/MySQL; không đổi sang PostgreSQL hay nhúng Jev extension vào database.

SPEC của S2 phải tách hợp đồng gate và trace có thể tái dùng khỏi rule CSKH cụ thể. `source_ref`, snapshot/version, policy/rule version, reason codes và outcome cần được kiểm trên từng path trước khi cân nhắc chia sẻ module cho Shift hay dự án khác.

**Exit:** shadow run trước; đối chiếu với SoT và tập gán nhãn. Không bật auto-skip cho ca rủi ro cao. Receipt ghi source digest, rule/policy, lý do, cả ba trục quyết định và quyền override. Đo zero external call cho dispatch local/chặn trong suite có provider thật cho nhánh được phép; giữ audit sampling thành luồng đánh giá riêng có quyền/ngân sách. Lấy mẫu ngẫu nhiên phân tầng ở cả nhánh bị bỏ qua để đo false negative; không suy coverage từ riêng các ca được gate chọn.

### S3 — Admission provider, ngân sách và audit

Đặt một điểm dispatch duy nhất sau gate cho Claude/Gemini/OpenAI/xAI: allowlist provider/model, hạn mức theo job/run/workspace, reserve trước call và settle sau usage, idempotency/reconciliation khi timeout hoặc crash, `KNOWN/UNKNOWN/PENDING` cho giá. Ghi audit bất biến cho rule, route, provider, version, human override và correction; giữ receipt đã làm sạch secret/PII. Không cho fallback từ provider lỗi sang provider khác nếu chưa có chính sách và ngân sách rõ.

Luồng đầu chọn một provider/model phù hợp để kiểm chứng, giữ interface cho các provider khác. Hết ngân sách hoặc provider lỗi phải tạo pending/escalation có deadline và owner; không đánh dấu việc đã xử lý hoặc tự thay model yếu hơn. Khi rule và LLM bất đồng, giữ cả hai kết quả cùng căn cứ rồi áp dụng policy review; output model không ghi đè quyền/policy.

**Exit:** refusal trước dispatch quan sát zero provider call; nhánh được chấp nhận có một call thật và request/response sanitized trong evidence artifact. Rà soát độc lập R2 và real-provider proof là điều kiện cho mọi claim CVF governance runtime.

### S4 — Học mẫu thiết kế quyết định có kiểu và tối ưu gate tại máy

Kế thừa cách chia câu hỏi, lựa chọn `unknown/no-match` và trace đã thiết kế tại S2 từ [skill TypeSafe/Jev](https://github.com/typesafe-ai/skills/blob/main/skills/typesafe-ai/SKILL.md); dùng phản hồi S5 để tối ưu `DecisionQuestion`, `DecisionAnswer`, `DecisionTrace` nội bộ. Không gán xác suất đã hiệu chuẩn cho enum/rule local. So sánh trên cùng corpus: (A) baseline LLM, (B) SoT + bộ lọc học từ Jev + LLM cho phần cần ngữ nghĩa/phản hồi, (C) biến thể local rule/index nếu có căn cứ cải thiện. Chạy shadow trước, giữ quyền/dữ liệu/ngân sách nhất quán giữa các phương án; không thêm dịch vụ phân loại trung gian.

Trước khi bật rule mới, preview những quyết định thay đổi trên corpus, số ca cần review và chi phí ước tính có trạng thái bất định. Tái dùng kết quả theo snapshot, rule/policy, task/prompt và cấu hình model liên quan; thay ngưỡng hiển thị/trọng số không cần inference mới nếu ý nghĩa câu hỏi và evidence giữ nguyên. Nguồn mới/sửa hoặc policy đổi cần invalidation; phân tích phần thay đổi vẫn phải giữ ngữ cảnh đủ.

**Exit:** so sánh trên holdout tiếng Việt theo từng nhóm rủi ro, có audit mẫu ca bị bỏ qua và độ bất định của phép đo. Chất lượng/evidence và giới hạn rủi ro khóa ở S0 phải đạt trước khi xét lợi ích chi phí. Không bắt buộc giảm số call nếu thêm call giúp xử lý đúng ca cần thiết; có thể chấp nhận chi phí tăng có giải trình và ngân sách được duyệt. Không dùng tỷ lệ tiết kiệm bù cho vượt ngưỡng bỏ sót; khi thất bại, rollback tối ưu và giữ việc cần xử lý.

### S5 — Review, correction và đầu ra có trách nhiệm

Phần tối thiểu triển khai cùng S2/S3: hàng đợi can thiệp có nguồn, trích dẫn, ưu tiên, người phụ trách, hạn xử lý, model/rule/gate và chi phí. Người được phân quyền xác nhận/từ chối/sửa đánh giá, giao xử lý và ghi kết quả; giữ actor/time/reason và before/after. Đánh giá được chấp nhận khác với vụ việc đã giải quyết. Dashboard/notification phân biệt proposal, accepted assessment và action status. Feedback phục vụ đánh giá rule; không tự trở thành rule production. Phần outbox mở rộng có retry/idempotency theo từng người nhận, nội dung tối thiểu và quyền gửi rõ.

**Exit:** không có AI output tự biến thành quyết định chính thức; sửa nguồn sau proposal làm evidence stale; thử quyền xem, phê duyệt, correction và notification lỗi từng đích.

### S6 — Pilot và quyết định mở rộng

Chạy shadow rồi pilot giới hạn theo kênh/job với người review; đối chiếu số liệu trước/sau trên cùng phân bố dữ liệu. Tách lỗi gate, lỗi model, lỗi nguồn và lỗi lưu/truyền; kiểm chứng rollback về baseline. Release/production có work order, review và live evidence riêng.

**Exit:** báo cáo lợi ích ròng, false negative theo nhóm, chi phí `local gate + LLM đã gọi + retry + human`, latency, chất lượng dẫn chứng và quyết định GO/REVISE/STOP của owner. Không dùng số liệu demo hoặc mock làm bằng chứng governance hay tiết kiệm thực.

### S7 — Vận hành và phát hành có kiểm chứng

Sau khi pilot được chấp nhận, rà soát toàn bộ hướng dẫn CQA kế thừa theo luồng thật; kiểm tra restore backup, retention, quyền người dùng, audit truy xuất, secret rotation, dependency vulnerabilities, độ bền dữ liệu khi nâng cấp và giới hạn tải. Thiết kế release pipeline/image của sản phẩm riêng; không tái dùng đường phát hành CQA cũ. Mọi cam kết production, khả năng mở rộng hoặc chi phí phải dựa trên phép đo của bản này.

**Exit:** independent review của tài liệu cài đặt/vận hành, thử backup-restore và rollback trên môi trường tách biệt, kiểm tra bảo mật và release gate có bằng chứng provider thật cho các claim governance runtime. Việc chấp nhận S6 không tự động cấp quyền triển khai S7.

### Sau khi hoàn thiện CSKH — nhân rộng và đề xuất nâng nền CVF

Chỉ mở khi use case CSKH đạt nghiệm thu và yêu cầu vận hành của dự án. Rút hợp đồng đã chứng minh, thử trên một dự án thứ hai với policy/corpus riêng, rồi mới quyết định tách module/rule pack và đề xuất nâng nền CVF. Giữ trace nguồn và bài học từ Jev/CVF/Shift; phần dùng chung không được kéo chậm các yêu cầu hoàn thiện CSKH. Tranche này có work order riêng ngoài S0–S7.

## TypeSafe/Jev kết hợp với AI/LLM

- [Skill chính thức của TypeSafe](https://github.com/typesafe-ai/skills/blob/main/skills/typesafe-ai/SKILL.md) nêu “code owns the workflow”, giữ quy tắc/phép tính/lookups xác định trong code, chia câu hỏi thành phần nhỏ, có lựa chọn `no-match`, rồi kết hợp và xác minh kết quả. Đây là nguyên tắc tham khảo để thiết kế **module nội bộ** của dự án.
- [Intent routing](https://docs.typesafe.ai/patterns/intent-routing) là nguồn học cách nối quyết định tới code, LLM hoặc người. Trong dự án, bộ lọc nội bộ học phương pháp ấy giữ dữ kiện và điều phối; LLM tiếp tục thực hiện phân tích ngữ nghĩa, nhận xét và phản hồi cần thiết. Đây là phối hợp giữa các lớp, không phải loại Jev khỏi nguồn học hoặc thay toàn bộ AI bằng rule.
- [Confidence guidance](https://docs.typesafe.ai/confidence) cần được phân biệt với rule local: không gắn số confidence giả, không suy output có kiểu là output đúng. Kế hoạch hiện dùng phương pháp từ skill; đưa model Jev hoặc một model phân loại thành dependency là thay đổi kiến trúc riêng, không mặc nhiên phát sinh từ việc học skill.

## Cổng quản trị

Mỗi S0–S7 mở bằng INTAKE → DESIGN → SPEC → WORK_ORDER → BUILD → REVIEW → FREEZE hoặc kế thừa bằng chứng được ghi rõ ở giai đoạn sớm nhất còn mở. R2 cần reviewer độc lập, scope/path/effect/credential cụ thể và rollback. Bất kỳ claim nào rằng CVF phân loại rủi ro, lọc dữ liệu, chặn call, route provider, validate output hoặc audit AI tại runtime đều cần gọi provider API thật và lưu request/response đã làm sạch theo `AGENTS.md`; mock chỉ dùng cho UI structure. Roadmap hiện tại và static checks không đáp ứng cổng đó.

Khi phát triển ứng dụng, work order được cấp quyền, mã/test và evidence là thẩm quyền cho trạng thái triển khai; roadmap chỉ là ý định. Role transition, rủi ro, reviewer độc lập, claim boundary và continuity phải được ghi ở từng tranche. Áp dụng nguyên tắc CVF về tách chuẩn bị dữ liệu khỏi quyền kết luận: core SOT3 là nguồn tham khảo có phạm vi, chưa được tích hợp vào runtime của ứng dụng này.

# Quyết định đề xuất: mẫu lọc dữ liệu SoT-first có thể tái sử dụng

**Trạng thái:** PROPOSED / REVIEW_PENDING · **Ngày:** 2026-09-27 · **Work order:** `CCMAI-ROADMAP-001` · **Rủi ro:** R2 ở bước thiết kế, tối thiểu R2 khi xử lý dữ liệu thật hoặc gọi provider.

## Bối cảnh và quyết định

Quyết định hiện hành ưu tiên hoàn thiện Customer-Care-Monitor-AI: giúp người phụ trách xác định ca cần can thiệp, xem bằng chứng, giao xử lý và ghi kết quả. Khi CSKH đạt nghiệm thu/vận hành, mẫu đã chứng minh mới là use case để nhân rộng và đề xuất nâng nền CVF. Phần dùng chung không trở thành điều kiện làm chậm việc hoàn thiện ứng dụng.

Mẫu thiết kế là **nguồn theo từng loại phát biểu → bộ lọc và quyết định có kiểu học từ Jev → AI/LLM phân tích ngữ nghĩa và tạo nhận xét/phản hồi → kiểm chứng, người duyệt và xử lý**. Local rules giải quyết phần xác định; LLM tiếp tục xử lý phần cần ngữ nghĩa. Chất lượng và rủi ro khách hàng là điều kiện trước tối ưu chi phí. Đây là thiết kế đề xuất, chưa phải module dùng chung hoặc runtime đã chứng minh.

```text
source adapters → raw evidence → bản chuẩn hóa có provenance/version
  → kiểm nguồn, quyền, trạng thái, dữ liệu đủ và policy đã duyệt
  → quy tắc và phân loại xác định tại máy
  → eligibility / execution / disposition theo policy
  → local có căn cứ, chờ có owner/deadline, hoặc AI/LLM qua admission
  → output được kiểm và lưu như proposal → human disposition
  → đánh giá được chấp nhận → giao xử lý → kết quả hành động/correction
```

### Ranh giới thẩm quyền

| Lớp | Vai trò | Không được tự làm |
|---|---|---|
| Raw evidence | Giữ payload gốc, nguồn, thời điểm và trace đồng bộ. | Trở thành fact đã xác nhận. |
| Bản chuẩn hóa | Tạo snapshot có version/digest và liên kết ngược tới evidence. | Che mất lỗi nguồn, xung đột hoặc dữ liệu stale. |
| Policy/SoT authority | Quy định quyền, phạm vi dữ liệu, mức rủi ro, phân loại chắc chắn và ngân sách; có owner, version và hiệu lực. | Bị nhãn từ provider hoặc prompt ghi đè. |
| Gate tại máy | Áp dụng kiểm tra và quy tắc xác định, xuất quyết định có lý do. | Đoán nhãn chắc chắn khi thiếu dữ liệu; gọi AI để quyết định những trường hợp local đã đủ. |
| Provider output | Đưa ra nhận xét hoặc phân loại ngữ nghĩa theo quyền đã cấp. | Tự xác nhận fact, cấp quyền hoặc thay policy. |
| Human disposition | Xác nhận, bác bỏ, sửa kết quả theo vai trò; lưu before/after. | Xóa dấu vết nguồn và quyết định trước đó. |

SoT gắn với từng loại phát biểu: hội thoại chứng minh ai đã nói gì; trạng thái giao dịch cần nguồn giao dịch. Phân biệt observed fact, inference và accepted assessment/decision. Người duyệt chấp nhận một đánh giá không biến nội dung suy luận thành sự thật khách quan; trạng thái “đã duyệt” cũng chưa có nghĩa vụ việc đã giải quyết. Ghi nhận dữ kiện quan sát được không bắt buộc người duyệt mọi bản ghi; tác động tới khách hàng/nhân viên theo quyền và policy riêng.

Nguồn thiếu/stale dẫn tới chờ bổ sung hoặc escalation có owner/deadline. Dữ liệu đủ nhưng nghĩa chưa rõ cần LLM phù hợp hoặc người theo policy; LLM không bù được evidence đang thiếu. Không suy `NO_AI` từ vắng bằng chứng. `RULES_ONLY` cần rule ID/version và source refs, vẫn có thể yêu cầu người duyệt. Bất đồng giữa rule và LLM được lưu với cả hai căn cứ để xét lại, không tự ghi đè policy hoặc che mất bất đồng.

### Hợp đồng quyết định dự kiến

Mỗi item có `source_ref`, snapshot digest/version, trạng thái/coverage đồng bộ, policy/rule version, scope và data classification. Thiết kế mới thay outcome đơn bằng ba trục có trace/reason codes:

| Trục | Giá trị định hướng | Ý nghĩa |
|---|---|---|
| Eligibility | `ELIGIBLE`, `WAIT_DATA`, `DENY` | Đủ dữ liệu và được phép xử lý hay cần bổ sung/chặn. |
| Execution | `NONE`, `RULES_ONLY`, `LLM` | Phần việc đã có kết quả hợp lệ, dùng rule, hoặc cần model. |
| Disposition | `PENDING`, `REVIEW_REQUIRED`, `ACCEPTED`, `REJECTED`, `CORRECTED` | Trạng thái đánh giá và quyền chấp nhận. |

SPEC phải quy định tổ hợp hợp lệ: `WAIT_DATA`/`DENY` không được dispatch LLM; execution là kế hoạch, chỉ chạy khi admission cho phép. `UNKNOWN` là trạng thái hiểu biết, không đồng nghĩa NONE/PASS. Chờ do ngân sách/provider có reason, owner, deadline, retry/escalation; không xóa việc hoặc tự hạ chất lượng. Action status (chưa giao/đang xử lý/đã giải quyết) độc lập với trạng thái đánh giá.

Nhánh LLM có admission/dispatch receipt và cost status (`KNOWN`, `PENDING`, `UNKNOWN`). Nhánh local/chặn phải đo zero external call trong suite có nhánh provider thật theo AGENTS; luồng lấy mẫu audit có quyền/ngân sách và receipt riêng. Output AI có model/prompt version, evidence span/ID và snapshot. Thay nguồn, ý nghĩa rule/câu hỏi hoặc policy cần invalidation; đổi bộ lọc hiển thị/trọng số có thể tái dùng kết quả nếu căn cứ vẫn giữ nguyên.

Schema, reason codes, policy precedence và quyền human override cần SPEC riêng trước khi BUILD; tên trường trên là hợp đồng định hướng, chưa phải API ổn định. Mọi nhánh phải quan sát được để đo false negative, false positive, chi phí local + provider + retry + human và latency.

## Jev, LLM và ngữ cảnh tiếng Việt

Giữ Jev/TypeSafe là nguồn học cách chia câu hỏi hẹp, chọn dữ kiện, trả kết quả có kiểu, `unknown/no-match` và kiểm lại. Bộ lọc nội bộ chuẩn bị/điều phối để AI/LLM phân tích và tạo phản hồi tốt hơn; không mặc định mọi việc phải kết thúc ở rule. Học skill không tái tạo khả năng ngữ nghĩa của model Jev. Hướng hiện hành vẫn không thêm Jev như một dịch vụ API/SDK trung gian. Phản hồi phục vụ người phụ trách; gửi trực tiếp tới khách cần thiết kế/quyền riêng.

Đánh giá trên corpus tiếng Việt có phủ định, hàm ý, nói giảm, mỉa mai, xưng hô, không dấu, viết tắt, phương ngữ, xen ngôn ngữ và ngữ cảnh nhiều lượt. Giữ vai trò, ngày giờ và dữ kiện ảnh/file/lịch sử liên quan; không ép nhãn khi bộ lọc không hiểu. Chất lượng và ngưỡng bỏ sót theo rủi ro phải đạt trước khi chấp nhận tối ưu. Có thể giữ/tăng call khi cần để bảo vệ chất lượng trong ngân sách được duyệt. Lấy mẫu ngẫu nhiên phân tầng ở nhánh bỏ qua để kiểm false negative; preview rule trước khi bật; feedback cần review trước khi thành rule mới.

[pg-jev](https://github.com/Blackbird081/pg-jev) cho thấy semantic judgment có thể ghép với predicate rẻ, projection nhỏ, batching, content cache và spend guard. Dự án hấp thụ các pattern đó trong Go để giữ MySQL, provider-neutral admission và audit ở application boundary. Không dùng PostgreSQL extension, `plpython3u`, superuser hoặc TypeSafe API trong database; chúng tạo dependency và external disclosure khác với kiến trúc đã chọn. Authority chi tiết: `docs/specs/DATABASE_FILTER_PIPELINE_2026-09-27.md`.

## Cách tái dùng giữa các dự án

Phần có thể chuẩn hóa là trạng thái dữ liệu, cấu trúc provenance, câu trả lời có kiểu, quy tắc fail-safe, decision trace, admission boundary, chi phí và bằng chứng review. Mỗi dự án tự khai báo adapter nguồn, thẩm quyền SoT, taxonomy nghiệp vụ, ngưỡng rủi ro, chính sách PII/retention, rule pack, quyền xác nhận và provider được phép. Không đưa Pancake, Zalo, Facebook, CQA hoặc nhãn QC của ứng dụng này thành giả định trong lõi dùng lại.

Sau nghiệm thu/vận hành CSKH, muốn chuyển mẫu sang một dự án khác cần inventory nguồn và owner, DESIGN/SPEC riêng, work order có scope/risk, corpus của dự án đó và review độc lập khi R2. Chỉ sau bằng chứng tái dùng mới đề xuất tách thư viện hoặc rule pack chia sẻ; nâng nền CVF cần work order và quyết định ở CVF core. Kiến trúc hiện giữ các ranh giới rõ để thuận lợi cho bước sau, không mở thêm dự án nền tảng trong giai đoạn hoàn thiện CSKH.

## Áp dụng CVF vào phát triển ứng dụng này

- Dùng chuỗi `INTAKE → DESIGN → SPEC → WORK_ORDER → BUILD → REVIEW → FREEZE`; mỗi tranche có acceptance, đường dẫn/effect được phép, điều kiện dừng, bằng chứng và next move. Kế thừa bằng chứng chỉ khi ghi rõ nguồn và phạm vi.
- Phân loại rủi ro trước BUILD. Dữ liệu khách hàng, API ngoài, bảo mật và hành vi governance runtime tối thiểu R2; REVIEWER độc lập với người triển khai. Ghi role transition và đồng bộ handoff, implementation truth, index/catalog theo thay đổi.
- SoT cho *quá trình phát triển* là manifest/policy, work order đã được cấp quyền, mã đang chạy, test và evidence; roadmap/spec chỉ mô tả ý định. Một claim chỉ rộng bằng bằng chứng của nó. Static check/doctor không chứng minh CVF kiểm soát provider tại runtime.
- Đối với claim CVF phân loại rủi ro, lọc DLP, chặn/route provider, validate output hoặc ghi audit tại runtime, phải có API call provider thật và request/response được lưu an toàn theo `AGENTS.md`. Mock chỉ dùng cho kiểm tra cấu trúc UI. Không dùng dữ liệu khách hàng hoặc gọi provider trong work order tài liệu này.

CVF core `docs/CVF_ARCHITECTURE_DECISIONS.md` (ADR-053) nêu Refinery chuẩn bị dữ liệu có nguồn nhưng không giữ quyền truth; Truth Kernel giữ quyền đánh giá/quyết định/receipt; output provider ở hạ lưu. Đây là nguyên tắc tham khảo cho ranh giới trên. SOT3 trong core có phạm vi kích hoạt giới hạn; ứng dụng này chưa tích hợp SOT3 và không thừa hưởng claim `LIVE_GOVERNANCE_PROVEN_BOUNDED` của core. Shift `docs/cvf/EVIDENCE_AND_TRUTH.md` là ví dụ domain khác về raw → proposed → confirmed, correction và audit; trạng thái nghiệp vụ của Shift không mặc nhiên trở thành policy CSKH.

## Điều kiện chấp nhận để tiến tới triển khai

Review độc lập phải kiểm tính đúng của ranh giới SoT, trường hợp thiếu/xung đột nguồn, ca rủi ro cao và đường bypass. Tranche S0 cần khóa baseline/corpus và tiêu chí đo; tranche sau mới định nghĩa schema, triển khai, shadow run và provider proof theo work order riêng. Cho đến lúc đó, đây là quyết định đề xuất trong REVIEW, không phải bằng chứng tiết kiệm chi phí hay gate đã vận hành.

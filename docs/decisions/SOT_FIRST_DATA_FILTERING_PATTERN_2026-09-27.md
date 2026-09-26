# Quyết định đề xuất: mẫu lọc dữ liệu SoT-first có thể tái sử dụng

**Trạng thái:** PROPOSED / REVIEW_PENDING · **Ngày:** 2026-09-27 · **Work order:** `CCMAI-ROADMAP-001` · **Rủi ro:** R2 ở bước thiết kế, tối thiểu R2 khi xử lý dữ liệu thật hoặc gọi provider.

## Bối cảnh và quyết định

Nhiều ứng dụng nhận dữ liệu từ kênh ngoài, phải phân loại trước khi dùng Agent/AI/LLM. Nếu đưa mọi bản ghi tới model, chi phí tăng và kết luận có thể vượt quá độ tin cậy của nguồn. Mẫu dùng lại là **nguồn có thẩm quyền → chuẩn bị xác định → gate có kiểu → chỉ gọi AI khi cần và được phép → kiểm chứng và người xác nhận**. Đây là hợp đồng thiết kế cho các dự án downstream; chưa phải module đã triển khai hoặc một dịch vụ chung.

```text
source adapters → raw evidence → bản chuẩn hóa có provenance/version
  → kiểm nguồn, quyền, trạng thái, dữ liệu đủ và policy đã duyệt
  → quy tắc và phân loại xác định tại máy
  → NO_AI | RULES_ONLY | HUMAN_REVIEW | NEEDS_LLM | DENY
  → chỉ NEEDS_LLM qua admission quyền, dữ liệu và ngân sách → provider
  → output được kiểm và lưu như proposal → human disposition → fact được xác nhận
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

Khi nguồn thiếu, stale, xung đột hoặc rule không bao phủ, gate trả trạng thái chưa xác định và route theo policy tới `HUMAN_REVIEW` hoặc `DENY`; chỉ route `NEEDS_LLM` nếu quyền, dữ liệu và ngân sách cho phép. Không suy `NO_AI` từ sự vắng mặt của bằng chứng. `RULES_ONLY` cần rule ID/version và evidence refs đủ để kiểm lại.

### Hợp đồng quyết định dự kiến

Mỗi item có `source_ref`, snapshot digest/version, trạng thái đồng bộ, policy/rule version, scope và data classification. Gate trả một outcome đóng (`NO_AI`, `RULES_ONLY`, `HUMAN_REVIEW`, `NEEDS_LLM`, `DENY`), `reason_codes`, evidence refs và trace của các câu hỏi hẹp. Nhánh `NEEDS_LLM` bổ sung admission/dispatch receipt và cost status (`KNOWN`, `PENDING`, `UNKNOWN`); nhánh local ghi nhận **zero external call** khi đã đo trên mọi đường chạy. Output AI lưu cùng model/prompt version và liên kết span/ID tới snapshot để kiểm trước khi đưa ra proposal. Nguồn hoặc policy đổi phải đánh dấu các kết quả phụ thuộc là stale.

Schema, reason codes, policy precedence và quyền human override cần SPEC riêng trước khi BUILD; tên trường trên là hợp đồng định hướng, chưa phải API ổn định. Mọi nhánh phải quan sát được để đo false negative, false positive, chi phí local + provider + retry + human và latency.

## Cách tái dùng giữa các dự án

Phần có thể chuẩn hóa là trạng thái dữ liệu, cấu trúc provenance, câu trả lời có kiểu, quy tắc fail-safe, decision trace, admission boundary, chi phí và bằng chứng review. Mỗi dự án tự khai báo adapter nguồn, thẩm quyền SoT, taxonomy nghiệp vụ, ngưỡng rủi ro, chính sách PII/retention, rule pack, quyền xác nhận và provider được phép. Không đưa Pancake, Zalo, Facebook, CQA hoặc nhãn QC của ứng dụng này thành giả định trong lõi dùng lại.

Muốn chuyển mẫu sang một dự án khác cần inventory nguồn và owner của nguồn, DESIGN/SPEC riêng, work order có scope/risk, kiểm trên corpus của dự án đó và review độc lập khi R2. Chỉ sau bằng chứng tái dùng mới đề xuất tách thư viện hoặc rule pack chia sẻ; việc sửa CVF core cần work order và quyết định ở CVF core. Mẫu này không thêm Jev/TypeSafe API, SDK hay dịch vụ phân loại trung gian; học từ skill của họ cách chia quyết định hẹp, kết quả có kiểu, nhánh `unknown/no-match` và giữ workflow trong code.

## Áp dụng CVF vào phát triển ứng dụng này

- Dùng chuỗi `INTAKE → DESIGN → SPEC → WORK_ORDER → BUILD → REVIEW → FREEZE`; mỗi tranche có acceptance, đường dẫn/effect được phép, điều kiện dừng, bằng chứng và next move. Kế thừa bằng chứng chỉ khi ghi rõ nguồn và phạm vi.
- Phân loại rủi ro trước BUILD. Dữ liệu khách hàng, API ngoài, bảo mật và hành vi governance runtime tối thiểu R2; REVIEWER độc lập với người triển khai. Ghi role transition và đồng bộ handoff, implementation truth, index/catalog theo thay đổi.
- SoT cho *quá trình phát triển* là manifest/policy, work order đã được cấp quyền, mã đang chạy, test và evidence; roadmap/spec chỉ mô tả ý định. Một claim chỉ rộng bằng bằng chứng của nó. Static check/doctor không chứng minh CVF kiểm soát provider tại runtime.
- Đối với claim CVF phân loại rủi ro, lọc DLP, chặn/route provider, validate output hoặc ghi audit tại runtime, phải có API call provider thật và request/response được lưu an toàn theo `AGENTS.md`. Mock chỉ dùng cho kiểm tra cấu trúc UI. Không dùng dữ liệu khách hàng hoặc gọi provider trong work order tài liệu này.

CVF core `docs/CVF_ARCHITECTURE_DECISIONS.md` (ADR-053) nêu Refinery chuẩn bị dữ liệu có nguồn nhưng không giữ quyền truth; Truth Kernel giữ quyền đánh giá/quyết định/receipt; output provider ở hạ lưu. Đây là nguyên tắc tham khảo cho ranh giới trên. SOT3 trong core có phạm vi kích hoạt giới hạn; ứng dụng này chưa tích hợp SOT3 và không thừa hưởng claim `LIVE_GOVERNANCE_PROVEN_BOUNDED` của core. Shift `docs/cvf/EVIDENCE_AND_TRUTH.md` là ví dụ domain khác về raw → proposed → confirmed, correction và audit; trạng thái nghiệp vụ của Shift không mặc nhiên trở thành policy CSKH.

## Điều kiện chấp nhận để tiến tới triển khai

Review độc lập phải kiểm tính đúng của ranh giới SoT, trường hợp thiếu/xung đột nguồn, ca rủi ro cao và đường bypass. Tranche S0 cần khóa baseline/corpus và tiêu chí đo; tranche sau mới định nghĩa schema, triển khai, shadow run và provider proof theo work order riêng. Cho đến lúc đó, đây là quyết định đề xuất trong REVIEW, không phải bằng chứng tiết kiệm chi phí hay gate đã vận hành.

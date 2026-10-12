# Roadmap kế thừa mẫu kiểm soát CVF vào runtime CCMA

Memory class: PLANNING_RECORD
docType: roadmap
Status: DRAFT
Risk: R1
Author: Codex /root
Date: 2026-10-12
Execution disposition: HOLD_SEPARATE_RUNTIME_SPEC_AND_WORK_ORDER

## Mục tiêu và authority

Owner yêu cầu chuẩn bị roadmap trước, chờ Claude phản hồi rồi tiến hành phần kế thừa. Lượt này chỉ cho phép lập kế hoạch/tài liệu; không cấp quyền BUILD runtime, fixture DB, Go campaign, SMTP/channel/provider/network, secret/customer data, commit/push hoặc deploy. Worker vẫn là Claude/Sonnet do owner copy giao; Local là orchestrator/reviewer và quyết định kỹ thuật cuối cùng. Không tự spawn worker.

Kết quả hữu hạn dự kiến: đóng bảy finding đã xác định trên các đường auth, dữ liệu dashboard, hiển thị hội thoại và outbound; người dùng không xem nhầm hội thoại, người thiếu quyền không lấy dữ liệu, thất bại gửi không biến thành thành công và job không bị giữ vô hạn bởi SMTP. Đây là hỗ trợ an toàn cho S1/S3/S5/S7, không phải nghiệm thu toàn bộ các stage hoặc luồng CSKH human review/assignment/outcome.

Nguồn chương trình: [điểm kiểm S0–S7 và P0–P4](../../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [learning điều phối](../learnings/feedback_orchestrator_roadmap_control.md), [learning repair](../learnings/feedback_cvf_repair_workflow.md). Giữ route sản phẩm một-job gate/proposal rồi human workflow. Không chuyển roadmap sản phẩm thành chuỗi cải thiện framework vô hạn.

## Current disposition và điều kiện chuyển bước

GOV002-C1 đã được Local chấp nhận ở phạm vi static portable package, material d0ebc890d043b5f94b647c1b228e8f1660f83292 đã được xuất bản trên nhánh project. Trạng thái REVIEW / REVIEW_PASS, FREEZE sản phẩm OPEN. Roadmap này vẫn DRAFT, chưa có work order runtime; bước tiếp theo là SPEC/work order riêng. Doctor PASS27/27 chỉ chứng minh workspace.

1. DONE: Local đã hoàn tất independent review GOV002 từ evidence đã trả, giữ nguyên implementation/proof của worker và sửa metadata có căn cứ. Không tái tạo implementation hoặc chạy lại toàn campaign mặc định.
2. Chỉ khi disposition GOV002 được giải quyết và có baseline candidate được xác định rõ mới lựa chọn package runtime đầu tiên. Publication GOV002 thuộc authority sẵn có và Local; roadmap này không thay publication contract.
3. Local chuẩn bị SPEC/work order và immutable seed đúng phạm vi trước runtime BUILD; xác nhận fixture và effect budget trước bất kỳ DB/native/provider effect. Không dispatch chỉ dựa vào roadmap DRAFT này.

Runtime reviewed source anchor: project HEAD ca1c3aa29a6088916bbd5d31bef37e67277f8b9c; backend/frontend sạch khi review. Nhận bàn giao runtime phải tái xác nhận hash/path và finding còn tồn tại. GOV002 có thay đổi governance chưa commit; không coi chúng là runtime acceptance.

## Source authority và mức kế thừa

Public Core dùng qua manifest tại e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba, read-only. Đường dưới đây tương đối với Core public, không phải project code để sửa. Đây là đối chiếu source có giới hạn, không claim complete corpus scan hoặc full-read mọi module.

| ID | Nguồn public tham khảo | Phạm vi dùng lại / giới hạn |
| --- | --- | --- |
| CVF-PERM | EXTENSIONS/CVF_v1.6_AGENT_PLATFORM/cvf-web/src/lib/execute-role-permission-gate.ts; src/app/api/admin/impersonate/route.ts trong cùng cvf-web | Server-side actor/action/output và bảo vệ target cấp cao; role CVF không tự ánh xạ thành owner/admin/member CCMA. Không copy mock-enterprise identity vào CCMA. |
| CVF-CAS | EXTENSIONS/CVF_v1.6_AGENT_PLATFORM/cvf-web/src/lib/pending-agent-execution-sqlite-store.ts, compareAndSwap | Conditional version/status update và đúng một dòng thay đổi. SQLite implementation là mẫu tham khảo; CCMA triển khai theo MySQL, không thay DB. |
| CVF-ATTEMPT | EXTENSIONS/CVF_v1.6_AGENT_PLATFORM/cvf-web/src/lib/provider-attempt-admission.ts; src/lib/canonical-web-gateway-execution.ts trong cùng cvf-web | Phân biệt admission, call-start, retry và outcome; không tự thêm CVF provider gate/budget đầy đủ vào scope bugfix backoff. |
| CVF-DEADLINE | EXTENSIONS/CVF_v1.6_AGENT_PLATFORM/cvf-web/src/lib/ai/providers.ts, AbortSignal.timeout | Deadline tại biên outbound. Cần chuyển sang deadline/cancellation của Go SMTP; không suy timeout HTTP thành SMTP đã được sửa. |
| UI-LIFETIME | Mẫu lifecycle/identity A–B–A trong phiên review; source lỗi CCMA tại Messages.vue và conversations.ts | Triển khai độc lập bằng Vue/Pinia từ hành vi và test contract. ResultViewer public ở pin hiện tại khác bản sửa mới trong private parent; không sao chép file/code/evidence private vào repo public. |
| DELIVERY-TRUTH | Principle: attempted, succeeded và persisted là các fact khác nhau | Chưa xác nhận reusable SMTP delivery module ở Core. Thiết kế receipt/retry theo output CCMA, không claim có thư viện ready-made. |

Nếu cần nâng Core, chỉ mở intake riêng khi phát hiện lỗi nguồn có tính dùng chung và public-safe; không sửa cha chỉ để hoàn thành bảy finding. Provider-specific memory không là nguồn authority.

## Finding ledger và acceptance dự kiến

P1/P2 là mức ưu tiên sửa lỗi, không phải CVF risk class. Mọi implementation auth/persistence/outbound sẽ được phân loại ít nhất R2 ở work order tương ứng.

| ID | Finding / source project | Mức bằng chứng hiện tại | Kế thừa | Acceptance phải chứng minh | Trạng thái |
| --- | --- | --- | --- | --- | --- |
| F01 P1 | backend/api/router.go reset-password route; backend/api/handlers/users.go ResetUserPassword không chặn admin nhắm owner | Static path review; chưa thử exploit/DB trong lượt này | CVF-PERM | Admin không reset owner hoặc target bị policy cấm; authorized controls vẫn thành công; denial không đổi password/version; ràng buộc target kiểm trong transaction hoặc cách tránh race tương đương | OPEN_NOT_DISPATCHED |
| F02 P1 | backend/api/router.go dashboard không RequirePermission; dashboard.go trả QC evidence/classification/cost | Static path/payload review, không claim live data leak | CVF-PERM | Member thiếu jobs/settings permission không nhận payload tương ứng qua dashboard hoặc đường thay thế notification logs; authorized users vẫn nhận đúng dữ liệu. Policy payload/route được chốt trước BUILD, không chỉ ẩn UI | OPEN_NOT_DISPATCHED |
| F03 P2 | backend/api/handlers/auth.go RefreshTokenHandler đọc version rồi update không condition/error check | Static race/error-path review; DB reproduction pending | CVF-CAS | Hai refresh cùng token chỉ một request thắng; failure/zero-row không mint successful successor; version không bị refresh cũ ghi lùi qua password-reset revocation. Không sửa login/session semantics khác ngoài contract đã chốt | OPEN_NOT_DISPATCHED |
| F04 P2 | backend/notifications/dispatcher.go ghi notified_at sau sends bất kể kết quả | Static control-flow review; transport/DB proof pending | DELIVERY-TRUTH / outcome separation | All-failed không đánh dấu delivered; mixed success có outcome/retry theo output, retry không gửi lại output đã xác nhận thành công; query/log/persistence failure không bị báo thành công. Chốt semantics notified_at và crash ambiguity trước BUILD | OPEN_NOT_DISPATCHED |
| F05 P2 | backend/notifications/email.go Send/HealthCheck bỏ context; analyzer giữ owner đến notification return | Static caller/callee review; chưa chạy SMTP | CVF-DEADLINE | Server synthetic giữ kết nối hoặc không trả reply vẫn kết thúc theo deadline; cancellation được tôn trọng, tài nguyên được đóng và job owner được release. Không gọi SMTP/customer thật | OPEN_NOT_DISPATCHED |
| F06 P2 | frontend/src/views/Messages.vue selectConversation; frontend/src/stores/conversations.ts fetchMessages dùng shared state | Đã tái hiện bằng hai hàm extracted thật và UI synthetic responses: selected B nhưng displayed messages/evaluation A khi A về sau. Không phải browser/full integration proof | UI-LIFETIME | A→B đảo response, A→B→A, late error/finally, reset/unmount và list/filter race không ghi nhầm message/evaluation/loading/error/selection. Chặn ở component và store; current healthy selection vẫn hiển thị đầy đủ | OPEN_NOT_DISPATCHED |
| F07 P2 | backend/engine/analyzer.go runBatchMode provider-error continue trước progress/backoff | Static loop review; adapter có retry riêng và vẫn giữ nguyên | CVF-ATTEMPT / bounded retry | Provider-error batch vẫn qua bookkeeping/backoff chung; streak/delay/reset/progress đúng, cancel ngắt chờ và không phát thêm call. Test invalid-response/save-error và healthy batches để tránh regression. Không tuyên bố CVF đã gate AI | OPEN_NOT_DISPATCHED |

Mỗi finding phải có named failing assertion trên source trước sửa, healthy control và pass sau sửa khi phù hợp. Source-derived static findings chưa có runtime reproduction được ghi đúng mức; không đổi thành VERIFIED chỉ vì viết roadmap. Không coi malformed fixture, skip hoặc lỗi setup là detector credit.

## Ba package đề xuất, chưa là work orders

| Thứ tự | Package / outcome | Findings | Điều kiện vào và kết thúc |
| --- | --- | --- | --- |
| A | Auth và quyền dữ liệu: role thấp không điều khiển owner/lấy evidence trái quyền; refresh có một winner | F01/F02/F03 | Sau GOV002 disposition; chốt permission matrix/target hierarchy và fixture scope. Kết thúc khi source/DB/route positive-negative evidence được Local nhận, không chỉ test helper |
| B | UI đúng lựa chọn: chat và đánh giá luôn thuộc hội thoại người dùng đang xem | F06 | Baseline clean và identity contract được chốt; có thể dùng UI synthetic thuần, không cần account/provider. Kết thúc khi deferred-response regressions ở store và component đạt contract |
| C | Outbound thất bại trung thực, hữu hạn: có thể retry đúng và không giữ job vô hạn hoặc bỏ qua delay | F04/F05/F07 | Chốt delivery semantics và isolated transport/DB prerequisites trước effects. Kết thúc khi failure/timeout/mixed-output/backoff regressions và caller integration đạt contract |

Đây là ba package dự kiến, không phải bảy tranche bắt buộc hoặc ETA. Local có thể gộp/tách đúng dependency ở SPEC, phải ghi lý do; không tự thêm successor hay campaign. Chỉ một package active tại một thời điểm, worker implementation và Local review độc lập. Các sửa nhỏ cùng phạm vi dùng repair hiện hữu.

## Evidence, budget và stop conditions

- Work order mới khai báo exact paths, actor/reviewer/commit owner, actual base HEAD, source map và effect budget; project-only, không sửa Core, seed cũ, evidence cũ hoặc gate yếu đi.
- Consume valid evidence; chỉ rerun khi có named contradiction/information gain. Một campaign cần thiết và một consolidated review mỗi package; stop và reassess ở repair round three khi không có independent new root cause. Không đặt vòng test full-stack vô hạn.
- Fixture phải được xác minh riêng trước DB/SMTP tests. Native fixture R085 fail/R086 NOT_RUN không tự thành PASS; thiếu prerequisite trả một consolidated blocker. Không mặc định hoàn thiện toàn durable-fence/recovery mới được sửa các bug độc lập này.
- UI structure/state tests synthetic chỉ chứng minh UI; source/hermetic transport/DB tests chứng minh application contracts trong scope. Mọi claim CVF kiểm soát AI/provider runtime phải có real-provider proof dưới authority riêng; roadmap này không authorize claim hoặc call đó.
- Source/unit, native integration, live và deployment là các mức khác nhau. Test exit0 với mandatory SKIP không phải acceptance. Không dùng count test/commit/tranche làm phần trăm sản phẩm.
- Known docs55 là closure/release debt riêng; roadmap này không sửa deadlinks và không xóa blocker. Facebook/Zalo OA accounts parked; full S2/S3/S5, pilot/S7, native fence/recovery và deployed facts vẫn OPEN/UNKNOWN.

## Checker Source Read-Ahead Block

Local đọc legacy gate ALWAYS_ALLOWED và active-seed path/effect checks; profile role/claim declarations; catalog schema/generator và progressive continuity trước khi soạn. Roadmap được lưu trong docs/reviews/planning và đăng ký catalog để không nới immutable C1 seed hoặc tạo executable work order trong lúc GOV002 review pending. Runtime BUILD cần separate SPEC/order/seed, không dùng exemption metadata làm source authority. Đây là planning artifact, không completion_review và không formal source acceptance.

## Public Export Disposition

EXPORTED. Project remote https://github.com/CVF-Ecosystem/Customer-Care-Monitor.git; branch codex/cvf-portable-inheritance; actual material d0ebc890d043b5f94b647c1b228e8f1660f83292; artifact docs/reviews/planning/CCMA_RUNTIME_CVF_INHERITANCE_ROADMAP_2026-10-12.md. Receipt ../probes/gov002_reviewed_publication_2026-10-12.json. Public-safe planning only; no private Core source/evidence export, runtime dispatch or product-completion claim.

## GOV002 prerequisite disposition

Local independently accepted GOV002-C1 bounded static portable material d0ebc890d043b5f94b647c1b228e8f1660f83292; project branch publication verified in ../probes/gov002_reviewed_publication_2026-10-12.json. The original prerequisite HOLD is satisfied. This roadmap remains DRAFT: no executable order or runtime effect, seven findings unchanged, next step is a separate SPEC/work order/seed before manual worker dispatch.

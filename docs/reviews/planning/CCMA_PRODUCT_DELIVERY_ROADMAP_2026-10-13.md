# CCMA — Roadmap phát triển sản phẩm và tiếp quản giữa các agent

Status: ACTIVE_PLANNING

Ngày: 2026-10-13. Plan ID: `CCMAI-PRODUCT-DELIVERY-20261013`. Tác giả: Local ORCHESTRATOR / SPEC_AUTHOR. Phạm vi đợt này: tài liệu R1 theo yêu cầu trực tiếp của owner; implementation tương lai tối thiểu R2 theo AGENTS.md. Baseline source: `49160128df5109ed46054babb6942457a3ed85a9`; worktree sạch trước authoring. Các mốc M1–M5 dưới đây đều chưa được nghiệm thu theo hợp đồng roadmap này.

## 1. Đọc và áp dụng roadmap này

Đây là kế hoạch thực thi sản phẩm hiện hành, chuyển đánh giá độc lập ngày 2026-10-13 thành mốc hữu hạn và điều kiện bàn giao. Agent mới đọc bootstrap → memory → active handoff → tài liệu này → SPEC/work order của mốc được giao. Không đọc toàn bộ lịch sử mặc định. Roadmap cho phép chuẩn bị kế hoạch trong phạm vi owner đã giao; không phải work order cấp BUILD, provider, dữ liệu khách hàng, account, push hoặc deploy.

- [Định hướng sản phẩm](../../PRODUCT_DIRECTION.md) giữ mục tiêu và ranh giới nghiệp vụ.
- [Roadmap S0–S7](../../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md) giữ kiến trúc đích, hợp đồng exit toàn stage và lịch sử bằng chứng. M1–M5 là cách tổ chức delivery xuyên stage; hoàn tất một mốc không tự đóng cả stage.
- Tài liệu này cập nhật thứ tự planning P0–P4 thành M1–M5; đưa quyết định dữ liệu/tiêu chí S0 lên trước implementation và đưa an toàn dữ liệu tối thiểu lên trước pilot. Không hạ các exit S0–S7 đã có.
- [Roadmap kế thừa CVF](CCMA_RUNTIME_CVF_INHERITANCE_ROADMAP_2026-10-12.md) đã đóng bảy finding trong phạm vi riêng. Không mở lại nếu không có regression hoặc mâu thuẫn bằng chứng cụ thể. ID F01–F07 của kế thừa khác ID finding sản phẩm.
- [Implementation truth](../../../IMPLEMENTATION_STATUS.json) và review gắn đúng source revision quyết định việc gì đã làm. Các câu “next” trong checkpoint lịch sử không phải lệnh hiện hành.
- Nếu có xung đột với AGENTS.md, policy, quyết định owner hoặc SPEC đã khóa: ghi rõ xung đột, dừng phần phụ thuộc và giao Local quyết định; không âm thầm chọn cách hiểu thuận tiện.

## 2. Kết quả sản phẩm phải đạt

Người phụ trách xem được hội thoại cần can thiệp, lý do và bằng chứng; xác nhận hoặc sửa đề xuất; giao người xử lý; biết vụ việc đang chờ ai và đã giải quyết thế nào. Luồng đầu tiên phải dùng được xuyên dữ liệu → quyết định gọi AI → đề xuất → con người → kết quả, thay vì chỉ thêm log hoặc màn đọc kết quả.

Phạm vi khởi đầu kế thừa [baseline S0](../RUNTIME_BASELINE_S0_2026-09-27.md): Pancake; phát hiện khiếu nại hoặc khách chờ phản hồi; corpus synthetic `s0-vi-intervention-v1`. M1 phải xác nhận phạm vi pilot, dữ liệu thực được phép dùng và người nghiệm thu; không coi baseline synthetic là quyền truy cập Pancake thật.

Ranh giới giữ cố định:

1. Một bản cài cho một công ty/cá nhân; không mở rộng thành SaaS nhiều khách hàng độc lập.
2. MySQL 8 và Go giữ dữ liệu, policy, admission; AI phân tích ngữ nghĩa. Không đổi database engine hoặc đưa lời gọi semantic network vào SQL.
3. Lời khách nói là evidence về lời nói; giao dịch/hoàn tiền cần nguồn nghiệp vụ có thẩm quyền. AI và người duyệt không biến suy luận thành sự kiện nguồn.
4. Đề xuất AI không tự thành kết luận chính thức, chế tài nhân viên hoặc câu trả lời đã gửi khách.
5. Chưa làm auto-send trả lời khách, đa kênh đồng loạt, nhiều provider/model, framework dùng chung hoặc nâng Core trong đường tới M3.
6. Chất lượng và rủi ro được xét trước tiết kiệm call/token; giữ UNKNOWN khi chưa đo. Không biến confidence do model tự báo thành xác suất đã hiệu chuẩn.

## 3. Điểm xuất phát và phần còn thiếu

| Năng lực | Bằng chứng hiện có | Khoảng trống ảnh hưởng delivery |
|---|---|---|
| Đồng bộ, replay, snapshot, dẫn chứng | Các acceptance S1 được tổng hợp trong roadmap S0–S7 | Real-channel coverage, dữ liệu thiếu/sửa/xóa và vận hành chưa được chứng minh toàn diện |
| Chuẩn bị nguồn, khởi tạo provider muộn | R050–R064; analyzer chuẩn bị candidate trước resolve provider | Quan sát nguồn/rule chưa là gate quyền/PII/đủ dữ liệu/WAIT_DATA hoàn chỉnh |
| Usage/cost và UI đọc receipt | R065/R068/R071/R067/R073 | Chưa có admission ngân sách, reserve/settle/reconcile và workflow người duyệt đầy đủ |
| Auth, quyền, UI lifetime, outbound/backoff | R088/R090/R091 được Local chấp nhận bounded | Không suy thành toàn bộ RBAC, recovery, real SMTP/provider, browser hay production đã đạt |
| Vòng xử lý vụ việc | JobResult và API đọc/lọc/export hiện có | Thiếu contract review/correction/assignment/deadline/outcome với concurrency và audit |
| CI và phát hành | Có workflow, có lịch sử hosted CI tại revision cũ | Cần proof gắn revision được chọn để pilot/phát hành; push thành công không chứng minh CI |

Review nguồn: [R088](../R088_FINAL_INDEPENDENT_LOCAL_REVIEW_2026-10-13.md), [R090](../R090_FINAL_INDEPENDENT_LOCAL_REVIEW_2026-10-13.md), [R091](../R091_FINAL_INDEPENDENT_LOCAL_REVIEW_2026-10-13.md). Các proof synthetic/local đã chấp nhận được tái sử dụng đúng phạm vi; không diễn giải chúng thành live governance proof.

Sai lệch tài liệu cần xử lý trong M1: spec database còn mô tả provider trước candidate; `knownLimitations` chứa lệnh dispatch R065 cũ; PRODUCT_DIRECTION có câu blanket “chưa triển khai” và readiness/provider chưa gắn ngày; các phần lịch sử UI có “next” cũ. Đối chiếu source rồi sửa mô tả hiện hành, đánh dấu lịch sử; không sửa bằng chứng gốc. Audit 8 mục `--omit=dev` và 17 mục full audit có scope khác nhau, không gọi chênh số là regression nếu chưa so sánh lock/command/date.

## 4. Thứ tự delivery và trạng thái

| Mốc | Kết quả nghiệm thu | Stage liên quan | Dependency | Trạng thái lúc tạo |
|---|---|---|---|---|
| M1 | Hợp đồng bản đầu tiên và route an toàn hữu hạn | S0, một phần S1/S7 | Nguồn/review hiện hành | NEXT — chưa khóa quyết định/acceptance |
| M2 | Một đường phân tích có quyết định và đề xuất gắn nguồn | S1/S2/S3 | M1 contract; prerequisite thật sự chặn | PLANNED — chưa dispatch |
| M3 | Người dùng hoàn tất review → giao việc → kết quả | S5 và an toàn S7 tối thiểu | M2 contract/data; một phần UI/API có thể chuẩn bị song song sau khóa contract | PLANNED — chưa dispatch |
| M4 | Chất lượng và lợi ích được đo trong shadow/pilot giới hạn | S0/S6; S4 nếu tối ưu | M2/M3, live readiness và an toàn dữ liệu | PLANNED — chưa dispatch |
| M5 | Bản phát hành có thể cài, vận hành và tiếp quản | S7 | M4 GO và proof release | PLANNED — chưa dispatch |

Không gán phần trăm tiến độ từ số tranche/test/commit. Mỗi mốc có thể gồm backend và frontend work order phối hợp nhưng chỉ báo hoàn thành khi khả năng sản phẩm được nghiệm thu. Không bắt buộc một work order cho từng helper/receipt. Nếu chưa có account hoặc dữ liệu thật, tiếp tục phần source/UI hợp lệ; giữ M4 phần live mở.

## 5. M1 — Khóa bản đầu tiên và các dependency

**Đầu vào:** baseline S0, code/review hiện tại, phạm vi owner đã cho phép, giới hạn account. **Đầu ra:** một SPEC sản phẩm tối thiểu có decision register, bảng acceptance, route dependency và kế hoạch evidence; dùng chính SPEC đó cho các work order M2/M3.

| ID | Việc phải chốt | Điều kiện hoàn thành | Chủ quyết định |
|---|---|---|---|
| M1-01 | Giữ hoặc thu hẹp use case Pancake khiếu nại/khách chờ | Định nghĩa đối tượng, trigger, ngoại lệ, thời gian chờ theo giờ làm việc và kết quả hữu ích; không lẫn QC toàn diện với một use case | Local đề xuất; owner chốt thay đổi nghiệp vụ đáng kể |
| M1-02 | Người review và người xử lý | Vai trò/quyền cụ thể; ai nhận WAIT_DATA, ai xử lý quá hạn, ai nghiệm thu pilot | Owner nghiệp vụ; Local ánh xạ quyền |
| M1-03 | Dữ liệu/corpus và nhãn | Nguồn, quyền sử dụng, PII, retention, phiên bản corpus, tách phát triển/holdout, cách phân xử nhãn bất đồng | Owner dữ liệu + reviewer |
| M1-04 | Chỉ số và ngưỡng | Chốt ngưỡng bỏ sót theo mức rủi ro, cảnh báo sai, thời gian xử lý, budget/call cap, độ trễ; ghi đơn vị/mẫu số/cỡ mẫu và điều kiện chưa đủ kết luận | Owner chấp nhận trade-off; Local thiết kế phép đo |
| M1-05 | Provider/model | Một lựa chọn được phép, revision/cấu hình nếu có, giá/provenance, timeout/retry cap, đường xử lý unknown usage/price | Local trong authority hiện có; owner cho quyền/budget còn thiếu |
| M1-06 | Topology và recovery | Một process hay nhiều process; ownership/cancel/stale-running behavior; dependency fence bắt buộc hoặc hoãn với chứng cứ; không tự giả định single-process an toàn | Local reviewer |
| M1-07 | Tài liệu và route | Xử lý bounded docs55 và drift nêu ở mục 3; lập danh sách prerequisite hữu hạn gắn acceptance M2/M3 | Session steward / Local |
| M1-08 | Work-order readiness | SPEC khóa contract; mỗi packet có allowed paths, role, ngân sách test/repair, pre-effect checks, evidence và stop condition | Dispatcher + reviewer |

M1 không đòi chạy toàn bộ runtime để lập kế hoạch. Các lựa chọn chưa đủ dữ kiện ghi `OPEN` kèm người quyết định, hành động giải quyết và phần bị chặn. Không điền một số ngưỡng/budget tùy ý để cho đủ bảng. Dữ liệu tổng hợp đủ cho thiết kế; thiếu quyền dữ liệu thật chỉ chặn hoạt động cần quyền đó.

**Exit M1:** M1-01…08 có quyết định hoặc disposition hoãn được chứng minh, không còn UNKNOWN chặn packet M2 đầu tiên; có một đường hữu hạn đến demo M3. Đánh dấu M1 chưa đạt nếu thiếu tiêu chí nghiệm thu hoặc dùng giả định thay quyền dữ liệu/budget.

## 6. M2 — Nguồn → gate → AI → đề xuất

**Phạm vi:** một job/use case, một provider/model đã chọn. Tái sử dụng snapshot, validation, ownership, lazy initialization và receipt đã được review. Không viết lại analyzer tổng thể chỉ để có cấu trúc mới.

Luồng bắt buộc: chọn candidate trong SQL → snapshot có provenance → đánh giá quyền/nguồn/PII/đủ dữ liệu → eligibility → admission ngân sách/provider → dispatch nếu được phép → validate output/evidence → lưu proposal/audit → đưa vào hàng đợi M3. Mọi tác dụng phải gắn tenant, job/run, conversation và snapshot hiện hành.

| ID | Acceptance bắt buộc | Case âm và lỗi cần chứng minh |
|---|---|---|
| M2-01 | Quyền và dữ liệu được kiểm trước disclosure; policy/rule có version | Sai tenant/quyền bị thu hồi/PII không được phép → không gọi provider; log không lộ secrets/payload nhạy cảm |
| M2-02 | Thiếu nguồn trả WAIT_DATA với reason, owner, deadline, hành động tiếp | Chat rỗng, partial sync, nguồn giao dịch thiếu, history không đủ; không biến UNKNOWN thành pass hoặc bỏ qua vĩnh viễn |
| M2-03 | Phân tách eligibility, execution và disposition | ELIGIBLE/WAIT_DATA/DENY; NONE/RULES_ONLY/LLM; PENDING/REVIEW_REQUIRED/ACCEPTED/REJECTED/CORRECTED không bị gom thành một status “success” |
| M2-04 | Một admission chung cho các entry path được bật | Lập inventory manual/test-run/scheduled/after-sync/agent/MCP/rerun theo source thật; đường ngoài phạm vi phải được chặn hoặc định tuyến qua cùng boundary, không để bypass rồi tuyên bố gate hoàn chỉnh |
| M2-05 | Budget check và reservation bền vững trước gọi; settle/reconcile sau | Hai run cạnh tranh, provider timeout, crash sau dispatch, usage ghi lỗi, token/giá UNKNOWN; không tính UNKNOWN là 0 hoặc tự giải phóng reservation khi chưa rõ đã bị tính phí |
| M2-06 | Proposal gắn snapshot/message evidence và model/prompt/rule/policy revision | Output sai schema, dẫn chứng không tồn tại, batch nhầm hội thoại, nguồn đổi trước publish; không lưu thành kết quả hợp lệ |
| M2-07 | Retry/cancel giữ định danh và trạng thái trung thực | Retry có trần/backoff, cancellation không tiếp tục publish/gửi; timeout không giả định provider chưa nhận; idempotency không đồng nghĩa exactly-once external call |
| M2-08 | Receipt đủ đọc lại quyết định và chi phí | Lưu reason, identity, thời gian, input digest, provider/model, usage KNOWN/UNKNOWN/PENDING; phân biệt accepted request với delivered/read/resolved |
| M2-09 | Có proof đường được gọi và đường không gọi | Local contract proof trước; live governance claim chỉ sau provider thật được phép, kiểm branch allow và quan sát zero-call deny/wait; không suy từ mock |

**Exit M2:** M2-01…08 được source/integration review trong phạm vi packet; M2-09 ghi đúng tầng evidence. Nếu live chưa chạy, chỉ nhận `LOCAL_CONTRACT_ACCEPTED_LIVE_OPEN`, không đóng toàn S2/S3 và không cho pilot qua M4. Queue/proposal có thể được nối M3 trong môi trường synthetic khi hợp lệ.

## 7. M3 — Người duyệt, giao việc và kết quả xử lý

**Kết quả nhìn thấy:** mở danh sách việc cần can thiệp → đọc hội thoại và lý do → xác nhận/bác bỏ/sửa → giao người phụ trách và hạn → ghi kết quả → truy lại lịch sử. Bắt buộc tích hợp model/API/UI; thêm nút UI hoặc bảng database riêng lẻ chưa là milestone.

Mô hình tối thiểu cần khóa trong SPEC: proposal/version gắn evidence; review event có actor/time/reason và before/after; case/action liên kết proposal, assignee, deadline, outcome; audit append-only theo hợp đồng đã chọn. Giữ vòng đời đánh giá tách vòng đời xử lý. Các tên bảng/enum cụ thể do SPEC quyết định; roadmap không áp đặt migration chưa được review.

| ID | Acceptance bắt buộc | Bằng chứng cần có |
|---|---|---|
| M3-01 | Queue ưu tiên rõ, evidence/missing/stale/unknown hiển thị trung thực | UI và API cùng contract filter/pagination/count; người dùng hiểu vì sao ca xuất hiện |
| M3-02 | Confirm/reject/correct có quyền và lý do | Member không được quyền bị từ chối ở backend; owner/admin đúng contract; giữ output gốc và lịch sử sửa |
| M3-03 | Review kiểm snapshot/version hiện hành | Hai người review cạnh tranh, source sửa/xóa, tab cũ; trả conflict/stale thay vì ghi đè im lặng; refresh không tự áp lại action cũ |
| M3-04 | Assignment, deadline và outcome | Giao người hợp lệ; đổi người/đổi hạn có audit; accepted assessment chưa là resolved case; reopen không xóa kết quả trước |
| M3-05 | WAIT_DATA và quá hạn có đường xử lý | Có người nhận, lý do, cách bổ sung/rerun/escalate; không tự đánh dấu bỏ qua/hoàn tất để sạch queue |
| M3-06 | Async UI gắn lifetime của route/tenant/conversation | Response cũ, double-click, navigation, logout/permission change không cập nhật sai đối tượng; giải quyết channels-store race nếu luồng này phụ thuộc nó |
| M3-07 | Notification không quyết định trạng thái nghiệp vụ | Ghi nhận hoặc từ chối thông báo tách khỏi review/case; thất bại thông báo không làm mất action; retry không tạo case/review trùng |
| M3-08 | Luồng thật qua browser và backend trên dữ liệu kiểm thử | Demo happy path và denied/stale/failure; có ảnh hoặc trace gắn revision, kiểm reload/persistence, ngôn ngữ và viewport được chọn |

**Exit M3:** người review hoàn tất một ca từ nguồn đến outcome qua trình duyệt thật; các case âm trọng yếu pass; Local độc lập nghiệm thu. Demo synthetic được ghi rõ phạm vi; không dùng để khẳng định AI governance, chất lượng tiếng Việt hoặc live channel. Giữ thiết kế UI hiện có; polish chỉ khi ảnh hưởng thao tác, đọc hiểu hoặc accessibility của luồng này.

## 8. M4 — Đánh giá chất lượng, shadow và pilot

**Điều kiện vào:** M2/M3 đủ khả năng cho phạm vi pilot; corpus/nhãn và metric M1 khóa; account/data/provider authority rõ; các yêu cầu an toàn trước dữ liệu thật ở mục 10 đạt. Facebook/Zalo OA vẫn parked cho tới khi owner mở lại. Pancake cần readiness riêng; không suy quyền Pancake từ tên baseline.

| ID | Công việc | Điều kiện nghiệm thu |
|---|---|---|
| M4-01 | Baseline và holdout | Tách development/holdout, không tune bằng holdout; version input/labels/config; baseline không dùng giả định “0 call tiết kiệm” |
| M4-02 | Live provider và nguồn kênh đã chọn | Receipt sanitized gắn source revision/provider/model; sự kiện allow/deny/wait; coverage kênh thật ghi rõ supported/unsupported, không suy toàn global F02 |
| M4-03 | Shadow trước tác động nghiệp vụ | So AI/gate với nhãn và người review; phân nhóm lỗi source/gate/model/persistence; audit mẫu phân tầng các ca bị bỏ qua |
| M4-04 | Pilot có người chịu trách nhiệm | Giới hạn channel/job/người dùng/thời gian/chi phí; có kill switch/rollback đã thử; không auto-send khách |
| M4-05 | Báo cáo quyết định | False negative theo rủi ro, false positive, evidence correctness, backlog, thời gian tới hành động, human time, tổng chi phí gồm retry/human; cỡ mẫu và bất định rõ |
| M4-06 | Owner GO/REVISE/STOP | GO chỉ khi tiêu chí khóa trước đạt; REVISE chỉ ra lỗi và phạm vi sửa; STOP dừng tác động và giữ bằng chứng; không sửa ngưỡng hậu nghiệm để thành pass |

S4 tối ưu là nhánh có điều kiện: chỉ thử sau khi có vòng phản hồi và baseline so sánh. Một internal shadow/usability run có thể cung cấp phản hồi trước formal pilot. Rule/cache/batch mới phải qua holdout và kiểm mẫu bị bỏ qua; rollback tối ưu khi vượt ngưỡng rủi ro. Không bắt buộc tiết kiệm chi phí để được pilot nếu chất lượng đạt và budget được phép. Không bắt buộc hoàn thành mọi tối ưu trước M5.

## 9. M5 — Release và tiếp quản máy khác

| ID | Acceptance | Proof phải gắn bản phát hành |
|---|---|---|
| M5-01 | Clone/bootstrap/build ở môi trường sạch | Revision/lock/runtime versions, public Core pin/doctor, cài dependency, cấu hình mẫu không có secrets; không phụ thuộc ổ D hay folder tạm của máy tác giả |
| M5-02 | CI đúng revision | URL run + SHA + workflow + conclusion, DB tests thực sự chạy; không dùng hosted receipt của revision cũ hoặc chỉ local preflight |
| M5-03 | Migration và bảo toàn dữ liệu | Fresh install và upgrade từ bản hỗ trợ; backup/restore đọc lại được; phân biệt rollback ứng dụng với migration có khả năng đảo |
| M5-04 | Vận hành | Giới hạn topology/tải, job owner/recovery contract, log/alert, secrets rotation, retention/deletion, dependency disposition và hướng dẫn xử lý lỗi |
| M5-05 | Release/deploy | Artifact/image/version và cấu hình deploy đúng scope được phép; smoke check, rollback drill; không merge/release/deploy ngoài authority |
| M5-06 | Bàn giao | Runbook người vận hành, known limitations, cửa vào agent, evidence/index cập nhật; người tiếp quản tái tạo đường cài và một luồng nghiệp vụ |

Owner cho phép đồng bộ repo không đồng nghĩa bật production. Nếu chỉ chuyển máy để code, có thể clone và tiếp tục trước M5 theo [AGENTS.md](../../../AGENTS.md); phải giữ nhãn development và các gap. M5 là nghiệm thu bản phát hành/vận hành, không phải prerequisite để mở IDE trên máy mới.

## 10. Dependency và backlog: cái gì chặn, cái gì để sau

| Hạng mục | Mốc bị chặn | Cách xử lý hữu hạn / điều kiện hoãn |
|---|---|---|
| Docs55 và chỉ dẫn hiện trạng lệch | Document closure/fresh-agent routing; không tự biến thành lỗi runtime | Một scope sửa các link/claim xác định rồi docs build/catalog; baseline lỗi gốc giữ lại; không mở chiến dịch viết lại docs |
| R086 native fixture, fence/recovery | M2 nếu source/contract thực sự cần; M4/M5 theo topology | Local kiểm dependency, chọn supported contract; muốn hoãn phải chứng minh đường an toàn và ghi giới hạn; fresh MySQL của R090/R091 không đóng R086 |
| Global channels-store stale race | M3 nếu queue/filter dùng store này | Reproduce theo tenant/request lifetime, sửa/test cùng scope UI liên quan; không coi conversation fix R088 đã tự phủ channels |
| Dependency warnings | Trước exposure/data thật theo mức ảnh hưởng; trước release | Ghi lock/command/date, phân biệt runtime/dev, exploitability/mitigation; sửa có mục tiêu, không chạy audit fix --force mù |
| Quyền, PII/disclosure, secrets, retention | Trước provider/customer data/pilot tương ứng | Backend denial, redaction/minimization, danh tính người truy cập, consent/authority và thời hạn dữ liệu phải kiểm chứng trước effect |
| Backup/restore và rollback | Trước nhập dữ liệu thật không thể thay thế; release | Disposable proof trước; không dùng DB đang vận hành làm fixture hoặc diễn tập phá hủy |
| Facebook/Zalo OA accounts | Chỉ live coverage các kênh đó | PARKED theo owner; source/synthetic work độc lập vẫn có thể đi tiếp; không tự xin token hoặc gọi API |
| Real browser/provider/current CI | M3/M2 live/M4/M5 theo acceptance cụ thể | Ba tầng proof độc lập; không lấy layer này thay layer kia; reuse receipt nếu source binding còn hợp lệ |
| Tối ưu chi phí, đa provider, đa kênh, polish, CVF uplift | Không chặn M3 nếu ngoài use case đã khóa | Đưa backlog sau acceptance; muốn kéo vào phải chỉ rõ acceptance nào không thể đạt nếu thiếu |

## 11. Evidence và review: làm đủ, tránh chạy lại toàn bộ

| Tầng | Chứng minh được | Không chứng minh được |
|---|---|---|
| Source/static/catalog | Contract trong code/tài liệu và consistency được checker kiểm | Hành vi runtime, live governance, CI hosted |
| Local unit/integration với DB/loopback kiểm soát | Lỗi, transaction, concurrency, cancellation trong điều kiện đã chạy | Provider/kênh thật, production parity |
| Mounted synthetic UI | Rendering và state/lifetime trong harness | Browser thật và toàn đường API |
| Browser + backend | Luồng tương tác, persistence và quyền trong môi trường được ghi | Chất lượng model hoặc nguồn kênh thật nếu input giả lập |
| Real provider/channel | Hành vi đã quan sát trên API/dữ liệu/phạm vi được phép | Mọi model/kênh, toàn corpus, production readiness |
| Hosted CI/deploy/pilot | Revision/workflow/môi trường/đối tượng đã ghi | Revision tương lai hoặc scope chưa chạy |

Claim CVF governance runtime phải tuân thủ AGENTS.md: cần provider API thật. Các probe synthetic kế thừa chỉ là bằng chứng hành vi phần mềm trong phạm vi đã ghi; không dùng mock để claim CVF kiểm soát AI. Trước live retry lỗi/timeout/ambiguous, lưu diagnostic secret-safe, phân loại nguyên nhân và quyền retry; không lặp mù.

Mỗi acceptance có ID, source revision/hash, test selector hoặc thao tác, expected/observed, exit code, raw artifact và giới hạn. Evidence reuse cần chứng minh dependency chưa đổi; reviewer chỉ rerun khi nêu contradiction, information gain và cost. Giữ output fail gốc; số test/subtest/file/call có đơn vị riêng, không cộng lẫn. Không log keys, signed headers, payload khách hàng chưa redacted. Token/bill/effort không đọc được ghi UNKNOWN với lý do.

## 12. Quy tắc dispatch và kiểm soát đi lệch

1. ORCHESTRATOR chọn một mốc và acceptance cụ thể; nêu “người dùng làm thêm được việc gì” trước danh sách file. Tái dùng review đã nhận, không giao worker khảo sát lại toàn repo.
2. SPEC_AUTHOR khóa input/output, state transitions, permission, error/retry/concurrency, compatibility/migration và out-of-scope. Schema/API chưa khóa thì chỉ cho phép chuẩn bị độc lập.
3. WORK_ORDER_AUTHOR ghi allowed paths/artifact class, source base, độc lập reviewer R2, explicit effects, test/call/cost/repair budget, protected surfaces, stop conditions. Scope implementation chỉ active sau lifecycle/authority đúng repo.
4. Worker shared workspace là INTERNAL_AGENT dù dùng model nào. Worker không tự mở successor, chỉnh seed/gate, mở account, lấy secret hoặc publish. Nếu provider/model routing đổi phải theo authority hiện hành; roadmap không khóa vào Claude/Luna hay một private canvas.
5. Để các nhóm API/UI dùng chung SPEC, phân định file ownership trước làm song song. Không hai worker sửa cùng migration/model contract khi chưa có một người integration owner.
6. Same-scope repair tiếp tục khi authority/budget không đổi. Chỉ tạo successor ở ranh giới thật; không chia mỗi lỗi thành một tranche hoặc đặt zero-repair mặc định.
7. Sau hai scope hỗ trợ chưa đạt acceptance sản phẩm, hoặc ngay khi prerequisite thất bại, Local rà lại route: gộp/thu hẹp/hoãn/tiếp tục có lý do. Repair round three không có root cause mới → `REVIEW_COST_ESCALATION_REQUIRED` theo AGENTS.md.
8. Báo tiến độ bằng capability, acceptance còn thiếu, blocker, số bước dự kiến và điều kiện xong. Không lấy số test, commit, tên model hoặc số file làm tỷ lệ hoàn thành.
9. SESSION_SYNC_STEWARD cập nhật roadmap tracker, implementation truth, bootstrap/memory/handoff và catalog khi route/status đổi. Không sửa generated index bằng tay hoặc xóa lịch sử để trạng thái đẹp hơn.

## 13. Mẫu packet và bàn giao cho agent tiếp theo

Packet thực thi phải mang đủ các trường sau; có thể nằm trong một SPEC/work order/return hiện hữu, không cần tạo thêm file chỉ để lặp nội dung:

```text
Milestone + acceptance IDs:
Product capability / user's observable outcome:
Source base + current dirty paths + accepted evidence reused:
Locked contract + decisions still OPEN + dependent actions paused:
Allowed paths / effects / explicitly excluded effects:
Implementation worker / independent reviewer / integration owner / commit steward:
Prerequisite -> why it blocks this acceptance -> exit -> stop condition:
Tests / fixture ownership / live authority / call-cost-time-repair budget:
Positive + negative + stale/concurrent/failure cases:
Evidence paths + revision/hash + actual exit + known limits:
Return status / reviewer disposition / remaining milestone gap:
Next allowed move / parked checkpoint:
```

Mẫu báo owner: “Dùng được X trong môi trường Y. Mốc M còn thiếu A/B. Bị chặn bởi Z, người giải quyết là R. Còn N bước đã lập kế hoạch; hoàn tất khi C. ETA chưa xác lập nếu chưa có dữ liệu.” Không ghi COMPLETE khi chỉ worker return; review phải phân biệt accepted local scope với full milestone, live và release.

## 14. Decision register khi bắt đầu

| Quyết định | Hiện trạng | Hành động kế tiếp / người xử lý | Chặn phần nào |
|---|---|---|---|
| D01 use case/kênh | Pancake + khiếu nại/khách chờ từ baseline; chưa khóa pilot hiện tại | Local chuẩn bị M1-01 từ baseline, chỉ hỏi owner khi thay mục tiêu hoặc thiếu quyết định nghiệp vụ | Khóa SPEC và live pilot |
| D02 người review/assignee, SLA giờ làm việc | OPEN | Local ánh xạ roles; owner chỉ định người/quy tắc nghiệp vụ còn thiếu | Workflow/pilot acceptance |
| D03 corpus/PII/holdout | Synthetic baseline có; real-data scope OPEN | Local lập corpus contract; owner cho quyền dữ liệu cần thiết | Disclosure và M4 |
| D04 ngưỡng chất lượng và budget | Chưa khóa trong roadmap này | Local đề xuất phép đo/căn cứ; owner chốt trade-off; không bịa số | M1 exit/pilot GO |
| D05 provider/model/key readiness | Chưa kiểm tra ở lượt planning này | Tái sử dụng authorization trước nếu còn đúng scope; kiểm readiness secret-safe, không yêu cầu gửi key vào chat | Live branch của M2/M4 |
| D06 topology/recovery/fence | Source/native/production boundaries còn mở | Local quyết định dependency từ source/contract, ghi safe route hoặc blocker cụ thể | Packet M2 có liên quan |
| D07 Facebook/Zalo OA | OWNER_DEFERRED | Chỉ owner mở lại; không đưa vào prerequisite của Pancake synthetic | Live riêng hai kênh |
| D08 release target và vận hành | OPEN | Chốt môi trường/người vận hành trước M5; không tự deploy | M5 release |

**Next allowed planning action:** chuẩn bị M1 contract và dependency decision trong phạm vi owner đã cho phép; chưa dispatch M2. Không yêu cầu owner trả lời lại các quyền đã có. Nếu quyền hoặc quyết định mới thật sự cần, hỏi đúng phần còn thiếu và tiếp tục công việc độc lập.

## 15. Source Authority và Evidence Comparison

Nguồn được đọc có chọn lọc tại baseline, không phải full-corpus audit: AGENTS.md, manifest/policy, bootstrap/memory/handoff R091, implementation current projections, roadmap S0–S7, PRODUCT_DIRECTION, UI roadmap current checkpoint, baseline S0, database filter spec, shared learning điều phối; source job model, result routes/handlers, analyzer provider boundary và workflow backend. Review R088/R090/R091 cung cấp acceptance kế thừa, không thay live proof.

Evidence Comparison: định hướng và current stage table khớp việc chưa có full gate/admission/human workflow. Analyzer đã có lazy provider initialization, trái câu hiện trạng cũ trong database spec. R065 đã có closure trong tracker nhưng còn lệnh dispatch cũ trong knownLimitations. Contradiction or Gap Disposition: dùng source/review gắn revision để lập M1 documentary reconciliation; giữ gaps trong mục 3/14, không coi phần chưa thấy là toàn repo không tồn tại. Claim Update: có roadmap delivery và route agent rõ hơn; chưa có capability runtime mới, product exit, live, hosted CI hay deploy mới.

## 16. Checker Source Read-Ahead Block

Downstream applicability: đọc `scripts/cvf_downstream_gate.py` (ALWAYS_ALLOWED, phase/status grammar, continuity, tranche scope, claims) và entrypoint/profile runner `scripts/cvf_gates/cvf_downstream_gate_runner.py`; đọc catalog source/render entrypoint `scripts/manage_cvf_downstream_catalog.ps1`. Áp dụng legacy preflight, installed profile qua trusted workspace doctor, catalog generation/check và diff/relative-link checks cho tài liệu. Không chạy private-Core checker trên project hoặc sửa checker để tài liệu pass. Work-order APPL/closeability không được coi là CHECKED nếu packet này chỉ là roadmap planning.

## 17. Dual Agent Surface Matrix

| Surface | Role/interface | Authority/risk boundary | Evidence | Adapter boundary | Disposition |
|---|---|---|---|---|---|
| INTERNAL_AGENT | Local author/reviewer/steward, shared repository | Tài liệu R1 hiện tại; R2 worker/reviewer độc lập trong packet tương lai | Repo/source/review được pin; kế hoạch này | Không đổi Core hoặc gate | Planning trong scope owner; worker chưa dispatch |
| EXTERNAL_AGENT_CLI_MCP | Remote advisory nếu được giao sau | Không có quyền private repo, source BUILD hoặc closure qua lời nhận xét | Chưa sử dụng trong đợt này | External research kết thúc trước internal implementation | NOT_APPLICABLE_WITH_REASON: không có remote research/absorption |

## 18. Public Export Disposition

Disposition: `EXPORTED`

Public-sync remote: `https://github.com/CVF-Ecosystem/Customer-Care-Monitor.git`
Public-sync branch: `codex/cvf-portable-inheritance`
Public-sync commit: `d72de76984e4aad9c208cc82e6093578797b5340`
Public artifact paths: `docs/reviews/planning/CCMA_PRODUCT_DELIVERY_ROADMAP_2026-10-13.md`; `docs/roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md`
Public catalog paths: `docs/catalog/ARTIFACT_REGISTRY.json`; `docs/INDEX.md`
Verification: `git ls-remote origin refs/heads/codex/cvf-portable-inheritance` returned `d72de76984e4aad9c208cc82e6093578797b5340` after the material push; receipt `docs/reviews/probes/ccma_product_delivery_publication_2026-10-13.json`. This is a CCMA project publication, not a CVF Core public-sync export. Product stage exits and live readiness remain OPEN.

## 19. Lịch sử cập nhật

- 2026-10-13: Owner yêu cầu lưu đánh giá thành roadmap chi tiết để agent khác tiếp tục. Local tạo M1–M5, acceptance/dependency/evidence/decision/dispatch contracts; không thay source runtime hoặc nghiệm thu thêm stage. Cập nhật sau phải giữ mốc, decision và lý do thay route, không ghi đè failures gốc.

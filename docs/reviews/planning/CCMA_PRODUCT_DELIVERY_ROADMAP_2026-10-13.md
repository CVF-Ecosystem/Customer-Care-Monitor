# CCMA — Roadmap phát triển sản phẩm và tiếp quản giữa các agent

## R136 — đã chốt hợp đồng màn hình đánh giá lại, chưa triển khai mã, 2026-10-20

Review độc lập chấp nhận thiết kế: hai API chỉ đọc về điều kiện chạy/trạng thái, dùng POST thực thi hiện có và giấy phép máy chủ cấp riêng; không nhập mã giấy phép hoặc cấu hình provider trên UI. Lưu marker trước khi gửi, khóa gửi lại khi mất kết nối/reload, đọc trạng thái thay cho suy đoán; giữ lịch sử khi ca chuyển sang OPEN. Có13hash nguồn đối chiếu khớp và danh sách file/tiêu chí cụ thể cho một work order R2 tích hợp. [Review](../R136_DESIGN_REVIEW_2026-10-20.md).

Đây là intended contract, không phải tính năng đã làm hoặc bằng chứng AI/CVF runtime. Mã/API/UI chưa thay đổi, provider/grant/execute0; E01..E06 chưa chạy. M3 vẫn PARTIAL_IMPLEMENTATION_OPEN. Bước tới là một gói R2 triển khai API đọc/UI và kiểm chứng có giới hạn; mọi cuộc gọi thật/quyền/secret cần authority mới trước khi dùng, không tái dùng R1171/1. Các mốc dưới là lịch sử.

## R135 — hàng đợi nhiều trang và từ chối thành công đã có bằng chứng local, 2026-10-20

Lượt thực tế exit0:13API GET kiểm tra phân trang/lọc/thứ tự/quyền;10trạng thái DOM và10phản hồi queue khớp chính xác. Mặc định20->26, chờ duyệt20->24; đổi bộ lọc thay đúng danh sách. REJECT200 có lý do/lịch sử sau reload; tổng mặc định26->25, chờ duyệt24->23, từ chối1->2; reader không có nút ghi. Nguồn/đề xuất/snapshot và ca khác giữ nguyên; chỉ1sự kiện người duyệt, provider0. Server/MySQL thử dừng,28container cũ không đổi. [Review](../R135_RUNTIME_REVIEW_2026-10-20.md).

Hai gap queue và REJECT đã đóng trong phạm vi synthetic/thứ tự thời gian. M3 vẫn PARTIAL: cần chốt yêu cầu/quyền cho UI thực thi đánh giá lại; giữ qualification mobile/raster/offline chưa hỗ trợ, business/channel/pilot/hosted/release mở. Tiếp theo chỉ INTAKE/DESIGN đối chiếu acceptance, không tự mở thêm fixture/tooling hay gọi provider. Các mục dưới là lịch sử.

## R134 — luồng bổ sung thông tin và lưu yêu cầu đã nghiệm thu local, 2026-10-20

Browser hoạt động trở lại. Lượt thử thực tế: lỗi503 có thông báo dễ hiểu/không ghi dữ liệu, tải lại thủ công rồi201 ghi nhận,201 lưu yêu cầu giữ thông báo đúng/lần cập nhật3. Tải lại và tài khoản chỉ đọc vẫn thấy hai lịch sử, không có nút ghi; Go exit0, đúng1bản bổ sung/1yêu cầu/2sự kiện, chưa chạy AI. Server/MySQL thử đã dừng;27container cũ giữ nguyên, provider0. [Review](../R134_RUNTIME_REVIEW_2026-10-20.md).

M3 vẫn PARTIAL: còn queue nhiều trang/filter/count/order và REJECT thành công; UI chạy đánh giá thực cần hợp đồng/quyền riêng. Khoảng trống hiển thị lỗi backend thực đã có bằng chứng503; qualification mobile/raster và offline NOT_RUN giữ nguyên. Bước tới chỉ INTAKE/SPEC cho bằng chứng còn thiếu, chưa dispatch/runtime; Facebook/ZaloOA, business/pilot/hosted/release vẫn mở. R132/R133 lỗi lịch sử và ngân sách cũ giữ nguyên. Các mục dưới là lịch sử.

## R133 — sửa mã xong, kiểm chứng browser cần chuyển agent

Đã sửa refresh cùng ca và điều kiện dữ liệu synthetic;30 kiểm tra UI, typecheck/build và offline Go compile PASS. Browser hiện không khả dụng nên chưa kiểm chứng luồng mới; lượt fixture kết thúc exit1 do không chạy browser, không nhận PASS. Native quyền403/403,26 container cũ không đổi; server/MySQL thử dừng và giữ nguyên, provider0. Theo chỉ thị owner, dừng và bàn giao agent có browser, không đổi stack/tự chạy lại. M3 vẫn PARTIAL. [Gói bàn giao](../R133_RUNTIME_TRANSFER_PACKET_2026-10-19.md). Các mục dưới là lịch sử.

## R132 — lưu thông tin được, yêu cầu đánh giá lại chưa nghiệm thu

Lượt browser synthetic thực tế ghi nhận503 rồi201, lịch sử bản ghi còn sau reload và reader không có nút ghi. Chưa nghiệm thu: lỗi refresh sau lưu và fixture thiếu nội dung khiếu nại khiến request409; test exit1 giữ nguyên. Server/MySQL thử đã dừng,25 container cũ không đổi, provider0. R133 chỉ sửa hai nguyên nhân mới này, có seed và proof riêng; M3 vẫn PARTIAL. Xem [review R132](../R132_RUNTIME_REVIEW_2026-10-19.md). Các mục dưới là lịch sử.

## R131 — đối chiếu acceptance M3, 2026-10-19

Đã chốt bảng 8 tiêu chí bằng bằng chứng hiện có và review độc lập. Cạnh tranh hai người duyệt đã có PASS trong package handlers kế thừa R126, không phải gap cần chạy lại; full run R125 exit1 vẫn được giữ. Source R128 hiện tại khớp nội dung 414 bản ghi (411 byte/hash nguyên gốc, 3 chỉ khác CRLF/LF).

M3 vẫn PARTIAL: còn bằng chứng queue nhiều trang, REJECT thành công, UI ghi nhận dữ liệu bổ sung/yêu cầu đánh giá lại và một lỗi thực hiển thị qua browser. B09 offline không được hỗ trợ vẫn NOT_RUN; không coi đó là blocker duy nhất. Bước sản phẩm kế tiếp R132: UI request-only trên API revision/request đã có; không tự chạy AI hoặc mở lại ngân sách cũ. M1 nghiệp vụ, M2 live, M4 pilot, hosted/release và tài khoản Facebook/ZaloOA giữ nguyên. Xem [review R131](../R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md). Các mục dưới là lịch sử.


## R128 — tên người tham gia ca cho tài khoản chỉ đọc, 2026-10-19

Đã sửa và nghiệm thu local: reader thấy tên người xử lý, người bổ sung dữ liệu và người ghi lịch sử sau reload; API chỉ trả user_id/name của người cùng workspace được ca/lịch sử ca tham chiếu. Không trả người ngoài ca/khác workspace; danh sách chọn người xử lý yêu cầu quyền đọc và ghi. Reader vẫn không có thao tác ghi hoặc yêu cầu danh bạ đầy đủ.

Native MySQL7test, UI/API21test, build và lượt browser hiện tại exit0 PASS; luồng sửa/giao/kết quả/mở lại và WAIT_DATA không hồi quy, không gọi provider. Không nhận full Go/frontend suite mới. Xem [review](../R128_CASE_PARTICIPANT_NAMES_REVIEW_2026-10-19.md). Tiếp theo chốt bảng acceptance M3 từ bằng chứng hiện có; B09 offline chưa có điều khiển hỗ trợ, full M3/business/pilot/hosted/release còn mở. Không mở chuỗi tooling/polish mới nếu không có gap sản phẩm cụ thể. Các checkpoint dưới là lịch sử.


## R127-A — luồng thao tác M3 trên trình duyệt, 2026-10-19

Đã nghiệm thu phạm vi synthetic hiện tại: sửa đánh giá → giao việc → ghi kết quả → mở lại, giữ lịch sử qua tải lại; giao người bổ sung và chuyển cấp vẫn giữ ca chờ dữ liệu; tab cũ409; backend chặn reader403 và UI reader không có nút ghi. Bản chạy lại exit0, nguồn/đề xuất giữ nguyên, đúng6event, không gọi provider. Lượt đầu exit1 do baseline fixture chưa seed con trỏ đề xuất hiện tại được giữ nguyên; không sửa lịch sử lỗi.

CSS390x844 được quan sát thực tế; ảnh gốc có raster nội dung375px, không nhận exact-raster390. Offline B09 chưa chạy vì thiếu điều khiển offline. Còn lỗi đọc hiểu cụ thể: reader thấy fallback tên người xử lý/lịch sử dù người dùng có tên; bước tiếp theo chỉ bổ sung nhãn người tham gia ca, giữ quyền ghi/danh bạ. Full M3/pilot/release còn mở. Xem [review](../R127_M3_BROWSER_ACTIONS_REVIEW_2026-10-19.md). Các checkpoint dưới đây là lịch sử.


## R126 — chốt sửa góp ý local, 2026-10-18

Các mục N01/N03/N04/N05 đã sửa và được kiểm tra local; N02 có đường chạy synthetic qua binary thông thường, setup/CLI/hạn mức/grant, không gọi provider. N03 giữ lịch sử khi người dùng xác nhận nguồn đã đổi; UI mới có mã thoát0 và fingerprint trước/sau bằng nhau. Frontend45file/474test và build PASS; Python231PASS.

Go: lượt full R125 vẫn là exit1. Sau sửa vòng đời pool test và assertion MCP, đã chạy lại trọn engine/MCP; kết hợp kết quả nguyên package của các phần nguồn không đổi cho current-package coverage PASS, gate DB đủ5sentinel và không có DB-unavailable skip. Đây là kết quả tổng hợp có kiểm tra hash/provenance, không phải một lượt full mới sau sửa. Hosted CI chưa chạy. Xem [review R126](../R126_FINAL_CURRENT_PACKAGE_REVIEW_2026-10-18.md).

Phạm vi sửa góp ý đã được nghiệm thu local; tiếp theo đối chiếu các điều kiện còn thiếu của M3 hiện có. Full M3/pilot/release và tài khoản Facebook/ZaloOA còn mở; không tự mở thêm tooling hay gọi AI trả phí. Các checkpoint phía dưới là lịch sử.


## R124 — hoàn thiện đường chạy synthetic và kiểm tra cuối, 2026-10-18

N02 đã triển khai startup mặc định tắt, cấu hình synthetic rõ ràng và CLI tạo fixture, bổ sung dữ liệu giả, tăng trần hạn mức không đặt lại phần đã dùng, cấp grant đúng yêu cầu/người dùng/nguồn/cấu hình/hạn dùng. Smoke bằng binary thông thường trên cài đặt mới đã tìm và sửa lỗi tenant `single-workspace`; luồng từ setup đến cấp grant PASS, attempts/executions/consumed đều0, không gọi provider. Kiểm tra thực thi dùng stub, không suy ra chất lượng AI thật. Xem [runbook](../../reference/INTERVENTION_SYNTHETIC_RUNBOOK.md).

N04 đã có lượt UI mới hoàn tất: đúng active proposal/lịch sử tiếng Việt, logout, exit0, fingerprint và effect counts trước/sau bằng nhau, server đã dừng. Không sửa lại các biên bản R120/R121 thiếu bằng chứng trước đây. Fixture Dashboard đã sửa đúng quyền đọc; hai lỗi Go kế thừa được sửa ở test context/trạng thái sending, không hạ kiểm tra quyền hay bỏ cơ chế bền vững của thông báo.

Full Go/frontend lượt cuối đang được kiểm tra trên nguồn cố định; chưa ghi PASS trước mã thoát cuối. Sau kết quả, cập nhật review `docs/reviews/R124_CLEAN_INSTALL_REVIEW_2026-10-18.md` và các điều kiện còn mở. Không tự mở dữ liệu thật, gọi AI trả phí, pilot hoặc release. Các mục R122 và kế hoạch bên dưới là lịch sử; checkpoint hiện hành theo mục này.


## R122 — kết quả sửa local, 2026-10-18

N01 đã sửa test fixture độc lập ledger/history, Python discovery231PASS; N05 sửa parser và72liên kết, `npm ci` + docs build PASS; N03 đã tái hiện nguồn đổi sau giao việc và thêm xác nhận phiên bản nguồn mới giữ nguyên đề xuất/quyết định/lịch sử, native MySQL + component tests PASS; N04 sửa encoding/ready ordering và đã quan sát đúng active proposal/lịch sử/đăng xuất trên UI synthetic. Các lượt kiểm thử bị ngắt khi môi trường phiên đổi phải giữ nhãn INCOMPLETE, không thay bằng PASS.

N02 chỉ hoàn tất trạng thái capability trên UI/API và [contract kích hoạt synthetic](../../reference/INTERVENTION_SYNTHETIC_ACTIVATION_CONTRACT.md); ordinary activation/budget/grant/clean-install chưa triển khai. Full Go suite chưa hoàn tất; frontend full suite470PASS/3lỗi Dashboard kế thừa, build PASS. Chưa có full M3/pilot/release hoặc runtime governance acceptance. Chi tiết: `docs/reviews/R122_LOCAL_FIX_REVIEW_2026-10-18.md`. Tiếp theo là chốt review/commit phần sửa có bằng chứng, rồi xử lý các gap trên đúng phạm vi; không tự gọi provider/bật dữ liệu thật. Các phần kế hoạch bên dưới là checkpoint trước sửa, disposition hiện hành theo mục này.


## Kế hoạch sửa sau thẩm định local — 2026-10-18

Đây là kế hoạch hiện hành, được owner yêu cầu cập nhật sau khi local đối chiếu bản review bên ngoài `CCM_Independent_Review_d7a6f76_2026-10-18.md`. Báo cáo bên ngoài là đầu vào; source, test và log được local kiểm tra mới quyết định tiếp thu. Chưa mục nào dưới đây được coi là đã sửa hoặc đã nghiệm thu. M1–M5 giữ nguyên; trọng tâm sản phẩm vẫn là M3.

Local kiểm tra tại `1e3db8b12d8576b33429fa3ece158786ee0b4f7e`, working tree sạch trước cập nhật; `git diff --name-only HEAD origin/main` không có khác biệt với bản merge `d7a6f7635608d1ad44169372a6b5e0fbcbcd09c4`. Local đã đọc source và tải lại job/log CI đúng SHA; chưa tái chạy Go/MySQL/frontend/docs hoặc HTTP/DB/browser cho các finding này.

| Thứ tự / mục | Kết luận local và căn cứ | Đầu ra sửa / điều kiện nghiệm thu |
|---|---|---|
| 1a — N01: CI backend tái lập | **CONFIRMED / FIX_PLANNED.** Backend job117600583070 có224 test/6errors; MySQL, Go test và build SKIPPED. `scripts/tests/test_r096_intervention_live_fixture.py:90,111` đọc trạng thái/ledger chiến dịch thật; fixture dùng seed lịch sử và ignored ledger. | Tách test logic guard bằng root/history/ledger cố định trong thư mục tạm khỏi admission của chiến dịch lịch sử; kiểm tra cả chấp nhận và từ chối. Clean checkout không cần `.cvf/runtime`/secret; Python suite chạy, MySQL sẵn sàng, Go `-json -count=1` và DB skip gate thực sự chạy, backend build có exit/log gắn SHA. Không chỉ tăng fetch-depth, bỏ test/guard hoặc dùng continue-on-error. |
| 1b — N05: docs và capability wording | **CONFIRMED / FIX_PLANNED.** Docs job117600606612 lỗi Vue parser tại R105 draft; deploy SKIPPED. Markdown có `<new reader UUID>` chưa bọc code; cần xác nhận bằng build sau sửa. README dòng14 nói chưa có bước xác nhận dù source M3 đã có. | Sửa markup tối thiểu và mô tả chính xác implementation / synthetic evidence / ordinary startup disabled / full acceptance OPEN. Clean `npm ci` + `npm run docs:build` thành công, link gate chạy hết. Không bỏ kiểm tra link hoặc sửa lịch sử bằng chứng; không mặc định số dead link cũ còn đúng. Deploy là acceptance riêng. |
| 2 — N03: nguồn đổi sau human action | **SOURCE_SUPPORTED / REPRO_REQUIRED.** `intervention_m3.go:131–141` chặn source lệch; `intervention.go:156–161` bảo vệ human ownership; `intervention_waitdata.go:209` chỉ nhận WAIT_DATA chưa human assessment. Chưa có HTTP/DB tái hiện. | Trước hết tạo test synthetic: CONFIRM → ASSIGN → thêm/sửa message qua writer được hỗ trợ → RESOLVE; ghi HTTP/version/state/event và thử đường phục hồi hiện hành. Đối chứng nguồn không đổi; kiểm tra đổi trước CONFIRM/sau RESOLVED. Chỉ khi xác nhận thiếu lối ra mới chốt transition nhỏ giữ lịch sử/người chịu trách nhiệm và test hồi quy. Không tắt source guard hoặc ghi đè human decision. |
| 3 — N04: phần UI/terminal còn thiếu | **CONFIRMED / FIX_PLANNED**, lỗi harness R121, không phải UI crash. Readiness print `ensure_ascii=False` ở `scripts/r121_ui_supervisor.py:821` thất bại trước browser; terminal có waits/cleanup nhưng thiếu route ledger/accounting. | Sửa encoding ở biên process với thay đổi nhỏ; test redirected stdout có tiếng Việt trước runtime. Readiness cần process còn sống và endpoint sẵn sàng. Sau work order/admission mới, thu đúng active proposal/history, login/logout và original exit/waits/final accounting/cleanup. Không replay provider/lifecycle; R120/R121 failure và cap cũ giữ nguyên. Full workflow actions kiểm tra trong acceptance M3 riêng, không nhồi vào campaign đọc UI. |
| 4 — N02: kích hoạt synthetic thông thường | **CONFIRMED_LIMIT / CONTRACT_REQUIRED.** Runner/executor chưa được cài trong ordinary startup; service yêu cầu synthetic policy + demo Pancake. Đây là giới hạn có chủ đích. | Chốt contract server-owned tối thiểu cho synthetic, provider/model/endpoint/budget/quyền và capability disabled/not-configured rõ trên UI. Sau work order mới, clean install có cấu hình hợp lệ chạy được luồng cho phép; thiếu/sai cấu hình bị từ chối; không cần test binary/sửa DB thủ công. Không chỉ gán global/bỏ marker; nguồn thật/pilot cần quyết định riêng. |

Nguồn CI do local tải lại: [backend run39102198980](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/actions/runs/39102198980), [docs run39102198991](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/actions/runs/39102198991). Đây là log thực thi hosted đã đọc, không phải test ứng dụng local mới chạy. Các kiểm tra repository/doctor không chứng minh sản phẩm hoặc AI governance đã đạt.

Root ORCHESTRATOR lập work order nhỏ cho1a/1b trước, independent REVIEWER quyết định kết quả; Luna Max làm worker thông thường, Sol Medium cho phần transaction/state khó nếu được dispatch. Có thể chuẩn bị test N03 sau1a; nghiệm thu UI sau khi các prerequisite liên quan đạt. Acceptance clean-install cuối cùng phụ thuộc contract N02; test service bằng harness không thay được acceptance đó. Frontend test/typecheck/build phải có log gắn SHA; ba lỗi Dashboard đã biết giữ OPEN và được định danh, không bị bỏ qua hoặc tự nhận full-green.

Ủy quyền hiện tại chỉ cập nhật roadmap/continuity và commit tài liệu; **chưa dispatch BUILD, chưa cấp runtime/provider hoặc tái dùng campaign đã tiêu thụ**. Không tạo chuỗi tooling/biên bản mới ngoài prerequisite có lỗi cụ thể; không viết lại toàn bộ hoặc mở lại F01–F08 chỉ theo báo cáo ngoài. Mock/stub chỉ dùng cho UI structure không khẳng định AI governance; mọi acceptance governance cần provider thật theo AGENTS. Full M2/M3, M1 business, pilot/release vẫn OPEN; Facebook/Zalo OA vẫn parked. Mốc kết thúc là build tái lập và một luồng M3 dùng được với phạm vi cấu hình công bố, không phải số lượng review/campaign.

Các checkpoint bên dưới là lịch sử; không dùng dòng “next” cũ làm chỉ dẫn hiện hành.

## Historical R121 terminal hold — 2026-10-18

CCMAI-RUNTIME-121 CHANGES_REQUIRED: original serve exit1/wait observed; Unicode console encoding failure before browser, terminal HOLD/FINAL_EVIDENCE_MISSING. Owned fixture/backend/Vite stopped, three listeners absent, protected9containers10networks472volumes and405sourcepins unchanged. Preserve consumed campaign/caps and raw evidence; read-only bounded follow-up decision only, no automatic retry/successor/provider. FullM2/M3/M1business/pilot/release OPEN/accounts parked/no push-merge-deploy. Review: docs/reviews/R121_FINAL_TERMINAL_REVIEW_2026-10-18.md.

Earlier checkpoint sections are historical.


## Current R121 local capability hold — 2026-10-18

Owner chose local continuation after R120 read-only lifetime review. One harmless local parent/child admission observed cooperative stop/wait exit0 and receipt; original outer exec returned1 vs declared17, so A2 NOT_MET. Source BUILD/compile/product runtime all NOT_STARTED, capability1of1 consumed/no retry. Root execution command lacked explicit native exit propagation; exact shell mapping cause not reproduced. Minimal prospective correction is explicit PowerShell exit propagation, not supervisor redesign; needs separately granted confirmation before product admission. R120 UI bounded acceptance/terminal gaps preserved, no oldcap reset or automatic successor. FullM2/M3/M1business/pilot/release OPEN/accountsparked.

## Current R120 observed UI / terminal evidence hold — 2026-10-18

R120 actual current UI observation: correct active rev2 over later off-lineage rev3; three readable Vietnamese history events/system reason, synthetic notice. CSS390x844 has no horizontal overflow, original JPEG375x811 (not exact-raster390x844/PNG); B09 unsupported NOT_RUN and B07 inherited. Overall CHANGES_REQUIRED: original supervisor terminal/held-child wait-exit/final fingerprint-effect counters MISSING after session loss, cause UNKNOWN. Root separate identity-proven sole fixture stop succeeded: EXITED/Pid0/Ports{}, listeners3019/64408/64629absent, protected8containers/9networks/471volumesequal, source400/binary/executablesunchanged. UI observation is not AI governance or actual workflow proof. Runtime stopped; no replay/repair/automatic successor/provider. Read-only failed-terminal/lifetime review/transfer only before separate governed runtime decision. FullM2/M3/M1business/pilot/release OPEN, accountsparked/no push-merge-deploy. Review: docs/reviews/R120_FINAL_INDEPENDENT_UI_REVIEW_2026-10-18.md.

Read-only Luna Max diagnosis independently verified by Root: the stop command acknowledges a request only; terminal publication depends on the original supervisor reaching cleanup. Existing303member archive matches its manifest but contains no final markers/terminal/serve receipt. Actual disappearance cause UNKNOWN; no product source defect or repair established. Transfer decision: docs/reviews/R120_READONLY_LIFETIME_REVIEW_AND_DECISION_2026-10-18.md. Current implementation projection aligned to this R120 hold. No new runtime authority or automatic successor.

Earlier checkpoint sections are historical.

## Current R120 finite UI route — 2026-10-18

Read-only Root/Sol diagnosis confirms fatal readiness during MySQL initialization (exact greeting bytes UNKNOWN); Docker responds, old owned container EXITED/Pid0/Ports{} and host listener absent now, without retroactive R119 cleanup PASS. R119 start/stop/checks remain consumed. Separately bounded R120 fixes only accumulated pre-auth greeting and explicit Windows listener-table proof, with unit discriminators/Root independent review before one new exact-admitted current-proposal/history UI observation. New seed precedes BUILD; runtime HOLD for source/binary and resource commits. No provider/lifecycle replay/product redesign/automatic support successor. First prerequisite failure stops. Current UI still NOT_RUN, B07inherited/B08raster/B09unsupported/fullM2/M3/M1business/pilot/release OPEN; accountsparked, no push/merge/deploy. Decision: docs/reviews/R119_READONLY_CAUSE_AND_R120_DECISION_2026-10-18.md. Earlier checkpoints historical.


## Current R119 failed UI setup checkpoint — 2026-10-18

Current-source proposal/history UI remains NOT_RUN: reviewed harness96ba23cc and offline compile1PASS/source-binary admission7ea25d35 preceded the sole fresh setup, which exited1 MYSQL_PROTOCOL_INVALID before backend/Vite/browser. Fixture start1/stop1 consumed; exact owned fixture EXITED and protected7containers/8networks/470volumes equal, listener-close proof OPEN(connectEx10035). Root records readiness review miss; raw failed packet preserved. R119 CHANGES_REQUIRED/noFREEZE/no replay or automatic tooling successor. Finite next decision: read-only root-cause/failed-packet review -> only justified explicit fresh readiness/start authority -> same missing M3 current-proposal/history browser proof and terminal review. No provider needed. R118 source/component and spent R117 real evidence inherited; B07bounded/B08raster/B09unsupported, fullM2/M3/M1business/pilot/release OPEN; Facebook/ZaloOA parked. See docs/reviews/R119_FINAL_INDEPENDENT_REVIEW_2026-10-18.md. Earlier checkpoints below historical.


## Current R118 UI source checkpoint — 2026-10-18

M3 current-proposal display/action binding now follows the backend active pointer/current lineage with unique legacy fallback; current heading/new WAIT_DATA history/system reason localized. Root independently24focusedtests and app typecheckPASS, source/component REVIEW_PASS at7f74a75f, FREEZE OPEN. No product browser/server/fixture/provider effects, no AI governance claim. See docs/reviews/R118_INDEPENDENT_UI_SOURCE_REVIEW_2026-10-18.md.

Remaining finite product proof: exact source/binary/fixture admission then one targeted current proposal/history wording observation, no lifecycle/provider replay. B07 inherited bounded; B08 exact raster qualified, B09 unsupportedNOT_RUN. Ordinary installed executor activation/fullM2/M3/M1business/pilot/release OPEN; R117 one real call consumed, all old evidence/budgets unchanged, accounts parked, no push/merge/deploy. No generic tooling successor.


## Current checkpoint — R117 bounded M3 backend real flow accepted, 2026-10-18

Actual WAIT_DATA source revision -> one real Alibaba execution/new run/new snapshot -> new proposal on same case -> synthetic writer API CONFIRM persisted, independently accepted. Real HTTP200, one call/debit/grant, faithful340input297output; native11top11nested+engine1PASS, noSKIP. Source5d38dd51/harness8aafa0ce, exact live admission06e16ea8; no ordinary installed activation/current UI-render/actual human decision claim. Owner-authorized same-fixture restart/finalstop after shutdown succeeded; final EXITED/current operational protected equality observed, original interrupted cleanupHOLD/defaultbridge replacement retained.

Finite next product move is M3 current UI review/processing and remaining mobile/wording acceptance, inheriting current accepted backend/evidence and enforcing actual browser capability/admission. No further R117calls/native/runtime or generic tool-repair chain. FullM2/M3, B08raster/B09offline and other historical UI qualifiers, Dashboard3baselinefailures, M1business/pilot/release remain OPEN; Facebook/ZaloOA parked; no push/merge/deploy. See docs/reviews/R117_FINAL_INDEPENDENT_REVIEW_2026-10-18.md.

## R116 product route — 2026-10-17

Owner grants continuation with fresh Alibaba live testing. R115 intended design accepted only; R116 source-only implementation then Root-admitted isolated MySQL discriminators and capped max2 fresh real calls, stopfirstvalid/anyuncertainty. Goal is WAIT_DATA same-case new proposal available for human review with original lineage preserved. No old budget/runner replay or installed activation. UI raster/offline/terminal gaps, Dashboard3knownfailures, fullM2/M3/pilot/release and M1business stay OPEN; Facebook/ZaloOAparked. Three finite steps, no ETA/percentage. Current source return compiled offline but Root review0 CHANGES_REQUIRED; D01-D04 repair1 before any runtime admission. Native/DB/provider remain NOT_RUN. See docs/work_orders/CCMAI_RUNTIME_116_WAITDATA_EXECUTOR.md.


## Historical R115 M3-05 intended design — 2026-10-17

Source-backed executor/integration SPEC and non-dispatchable R2 implementation DRAFT prepared by Sol Medium and independently reviewed by Root after2documentrepairs;14sourcefiles/31referencebounds/hashpins verified against product68a3d30b. Product source unchanged. Intended path binds fresh execution authority, atomic current-permission/source claim+JobRun, canonical replay, durable call marker/cost, same-case active proposal and legacy compatibility, source/delete provenance fences. Unknown call/ownerloss stays manualhold, no automatic provider replay or invented terminal receipt. This is design acceptance only: executor/fences/activeprojection NOT_IMPLEMENTED, product/DB/liveproof NOT_RUN; no M3/S-stage exit.

Finite next route: separately seed exact R2 implementation order -> bounded source/unit and isolatedDBfault/concurrency proof -> fresh explicitly authorized real-provider acceptance plus independent result. Current candidate budgets are proposals, no provider/secret/runtime grant or deploy. R102 remains localrevision/nonexecutingintent; R114CHANGES_REQUIRED/noFREEZE/terminalgap, B08rasterqualified/B09unsupported, R1103Dashboardfailures and fullM2/M3/pilot/release/M1business gaps retained; accounts parked. No newUI/tool-repair chain, old budget reuse, push/merge/deploy. Earlier sections below are historical.

## Historical R114 bounded onboarding result — 2026-10-17

Sourceaa92c149/campaignHEAD95401340: shared banner now says **Cài đặt AI** and all5onboarding labels are fully readable/wrapped on desktop and actualCSS390x844 queue/detail. Root independent3focusedtestsPASS/exit0 and forcedtypecheckPASS; no full-suite rerun. This resolves the two observed R113onboarding defects on current source, not allUI/fullM3. [Independent review](../R114_INDEPENDENT_UI_EVIDENCE_REVIEW_2026-10-17.md). B07inheritedR113boundedUI; B08DOM390x844 plus prior form/history evidence qualified by JPEG raster(queue390x843/detail375x811); exact390x844rawPNGunobserved. DerivedPNGnoresize. B09unsupportedNOT_RUN.

Sole114fixture/backend/Vite/tab1,authPOST2(login/logout),caseaction/native/Go/runtimefrontendvalidation/provider/mock/retry0. Actual logout/viewportreset/ownedtab2close observed; baseline stale tab1untouched. Original supervisor42009terminal/cleanup/source-afterMISSING, duration/end/causeUNKNOWN; later Rootstoprequestafterdeadline is not cleanup proof. Independent final fixturealreadyEXITED/exclusive/protected4equal/PIDsandportsabsent, source/binarypinsmatch;0Rootfixturestop.185survivingrawfiles73584bytes hash/ZIP-preserved. CHANGES_REQUIRED/noFREEZE; no new114runtime/reset or tool-repair chain. Root continuity source-only/admission mismatch was held and fixed before start.

Finite next route: recover original terminal receipts read-only only if actually available; otherwise retain gap without replay. Next product planning is M3-05 actualWAIT_DATAexecutor/source-writer integration SPEC -> separate bounded BUILD/live-proof authority. No provider/secret authority here. R102remains localrevision/nonexecutingintentonly, R11026PASS/3DashboardFAIL/typecheckPASS unchanged, fullM2/M3/pilot/release OPEN and Facebook/ZaloOAparked. Earlier route/result sections below are historical; current114records control. No push/merge/deploy.

## Current R113 UI result and finite next move — 2026-10-17

Current source-bound integrated browser evidence now exists at UIa28cdd0a/binding0622d185/campaignHEAD0a266623. B07 boundedPASS: fresh reader after actual ownerlogout reads queue/detail/history without write controls, revision1. B08 actualCSS390x844 queue/detail/ownerform/four expanded historic changes observed, with raw-JPEG raster variance(queue390x843/detail-form-history375x811), derivedPNG no resize; no exact390x844rawPNG or globalmobilePASS. B09 unsupportedNOT_RUN. Visited intervention/history wording understandable, but shared onboarding “Cấu hình AI Provider” and mobile chip clipping remainOPEN. [Independent review](../R113_INDEPENDENT_UI_EVIDENCE_REVIEW_2026-10-17.md).

Sole113campaignspent,fixture/backend/Vite1/tab1/auth6of8/lifecycle0/native0/Go0/frontendvalidation0/provider0/retry0. Supervisor originalcleanup/result/exitreceiptMISSING after unknown-session; causeunknown. Root independently verified recordedPIDs/listenersabsent, stopped exact owned fixtureonce, then observed fixtureexited/exclusive/protectedbaselineequal. Actual logout/viewportreset/tabclose observed. Missing original evidence stays missing; no synthetic cleanupPASS. Raw196files80089bytes exactsource/export/zip verified; browser rawJPEG/DOM and derivedPNG/capturecodec/dimensions retained.

R113 CHANGES_REQUIRED/noFREEZE. Finite next route: bound source-only onboarding wording/mobile-chip repair SPEC/workorder -> independent source/tests and exact changed-source/binary admission -> one targeted changed-screen proof/independent result, inheriting B07/current qualified history without lifecycle replay. This is a concrete product defect route, not another generic browser/Docker repair campaign. No further113start or unusedslotreuse; oldR110/R111/R112 source/evidence/budgets immutable. R11026PASS3knownDashboardFAIL/typecheckPASS remains qualified. FullM2/M3/M4/M5 and S0-S7 exits OPEN; R102local backend only/noactualAIrerunexecutor, Facebook/ZaloOAparked; no provider/secret/push/merge/deploy authority. Earlier current-route paragraphs below are historical.

## Current browser tooling recovery — 2026-10-16

R103 recovered official exact driver1.57.0 plus Chromium/headless1200 from npm/official browser CDN. Root335package hashes and610preexisting cachefile hashes verified; actual local browser143 launch/control/PNG capture PASS. [Independent tooling review](../R103_INDEPENDENT_TOOLING_REVIEW_2026-10-16.md) accepts cache/component health only. Codex browser API currently exposes no IAB provider, native pipe is unavailable, and external Antigravity browser_subagent was not observed; conditional R101 UI/server/fixture branch did not run. Driver ZIP404 bootstrap dependency was addressed without source/project dependency changes or version faking. Next: external Claude/Antigravity client blank-page capture using restored cache, then source-bound R101B01..B10 only when capable. No repeated reinstall/old-CDN attempt. Independent M3-05 executor SPEC remains available while bridge is absent. R101 Dashboard3baseline/FREEZE and full milestones remain OPEN; R102 local backend FROZEN, accounts parked.

## Historical route after R101 browser retry2 — 2026-10-15

[Independent retry2 review](../R101_RETRY2_INDEPENDENT_REVIEW_2026-10-15.md) records BLOCKED with qualifications: all16 manifest entries verified, no screenshots, B01..B10 NOT_RUN, R101 counters unchanged. The reported stop avoided app-infrastructure launch; a repeated automatic Playwright/CDN fetch remains an attempted external effect, so strict no-download wording and universal browser incapability are not accepted. Operator/tool chronology is not independently authenticated. Original report/log retained; R101 REVIEW/FREEZE and Dashboard3baselinefailures stay OPEN. Do not repeat unchanged-tool UI attempts.

R102 remains FROZEN for local revision/nonexecuting intent. The selected next product route is M3-05 actual WAIT_DATA rerun executor/integration SPEC: audit current request/ownership/reservation paths -> define fresh source/permission, idempotency, linked run/history and failure/recovery contract plus real-provider acceptance -> bounded R2 work order before BUILD/effects. No source/provider/secret/browser-server campaign is dispatched here. UI acceptance resumes only with evidenced working capture or a separately scoped attributed manual route. M1 business decisions remain separate; fullM2/M3/pilot/release OPEN and Facebook/ZaloOA parked.

## Current R102 local backend FREEZE — 2026-10-15

R102 source6700de80 passed Root independent20native+7nested MySQL checks, noSKIP, plus2 engine checks at unchanged engine source. The local capability captures append-only source revisions and persists AWAITING_EXECUTION_AUTHORITY/NOT_RUN intents; original analysis/history/owner preserved. [Final bounded review](../R102_FINAL_INDEPENDENT_BACKEND_REVIEW_2026-10-15.md) accepts W01..08 backend software only. Actual AI executor and source writer integration are still absent from this slice. Go7 (worker4/reviewer3), repairs3, same sole fixture stopped/preserved with protected equality. Old failure and Root review miss retained; owner continuation7cbcd52 removed same-scope numerical caps, not effect/claim boundaries. Next bounded move: separately authorized SPEC/work order for actual WAIT_DATA executor/integration or parallel M1 people/SLA/corpus decisions. R101 browser/Dashboard3baselinefailures/FREEZE, fullM2/M3/M1business/pilot/release remain OPEN; Facebook/ZaloOA parked. No UI/provider/mock/download/originalDB/push/deploy effects. Earlier current/dispatch paragraphs are historical.

## Historical R102 source BUILD — 2026-10-15

Owner explicitly authorizes subagent work under Root supervision. Immutable seedc199f13 precedes dispatch11a60c0; final SPEC docs/specs/M3_WAITDATA_BACKEND_R102_SPEC_2026-10-15.md and order docs/work_orders/CCMAI_RUNTIME_102_WAITDATA_BACKEND.md govern source-only BUILD by /root/r102_worker Sol Medium, Root independent reviewer/commit steward. Intended local source revision and nonexecuting rerun intent only; no actual AI reanalysis. Root source review precedes any runtime, newR102Go0of8/fixture0of1. R101 REVIEW/browser/fullsuite/FREEZE remains OPEN, old resources/counters unchanged. Earlier candidate route below is historical preparation, not current dispatch truth. No UI/provider/mock/RunManual/guard bypass/installed activation/download/push/deploy.

## Historical backend-first route — WAIT_DATA candidate, 2026-10-15

Owner chooses work independent from blocked browser tooling and continues with M3-05 backend preparation. R101 source200df3e has inherited six native groups/four nested PASS and compile/newUI/build evidence; browserB01..B10 NOT_RUN, three inherited Dashboard tests FAIL, independent Claude return BLOCKED with driver-fetch qualification. R101 remains REVIEW/FREEZE OPEN. This updates the stale pre-dispatch current-state descriptions below; those sections are retained as historical checkpoints.

Prepared intended [WAIT_DATA SPEC](R102_WAITDATA_BACKEND_SPEC_2026-10-15.md) and [candidate work order](R102_WAITDATA_BACKEND_WORK_ORDER_2026-10-15.md). Backend capability: observe locally supplemented source in an append-only revision, then persist an explicit rerun intent AWAITING_EXECUTION_AUTHORITY/NOT_RUN without replacing original analysis or human history. Existing source writer supplies messages; no transcript mutation, executor, provider, UI repair, Dashboard repair or live channel work is included. Actual AI rerun and full M3-05 remain OPEN. Candidate R102 is not dispatched and has no installed seed or runtime authority.

Finite next route: finalize/commit immutable dispatcher authority -> implement the coherent revision/intent/API/test slice -> admit bounded synthetic MySQL checks and independently review local scope. R101 budgets/fixture2of2 remain unchanged and cannot fund this candidate. M1 people/SLA/corpus/threshold decisions may proceed in parallel; M4/M5 remain gated. Browser capability repair is not a prerequisite for this backend slice. Facebook/Zalo OA remain parked. No current runtime/provider/download/server/push/deploy effects.

## Current M3 dispatch route — R101, 2026-10-15

R100 bounded real synthetic proposal/faithful usage remains accepted at its exact recorded source; stopped after1/4 physical calls, unused slots unavailable. Current execution focus M3: accepted intended H01..H08 SPEC and immutable seed43c65ad bound review/correction/assignment/deadline/outcome, source/version/tenant guards and human history. Sol Medium implementation selected for transactional/source-lock/UI-lifetime difficulty after Luna Max inventory; Root independent review/commit. Source-only first, Root source admission before finite local MySQL/backend/browser proof (max16Go/two owned cached fixtures, zero provider/mock/secret). No installed activation or existing app/DB changes. Source/native/browser NOT_RUN before dispatch; fullM3 including WAIT_DATA rerun acceptance OPEN. M1 real people/SLA/corpus/threshold and fullM2/pilot/release OPEN, Facebook/Zalo OA parked. Route has three remaining steps: implement coherent human workflow -> actual isolated backend/browser proof -> independent acceptance/conditional local closure. No generic prerequisite successor; same-scope repairs bounded2, third without new cause cost escalation. Earlier current/next sections below are historical as of this R101 dispatch.

## Current product result and finite next move — R100, 2026-10-15

| Milestone | Current truth | Remaining acceptance / next move |
|---|---|---|
| M1 | Local synthetic intended contract accepted historically | Named reviewer/assignee, SLA, permitted corpus/holdout andquality/budget thresholds remainOPEN; settle alongsideM3planning |
| M2 current | One realAlibaba qwen3.7-flash complaint proposal valid on currentR099source + R100harness; exactsnapshot/ID/quote andKNOWN360/261usage persisted; DENY403/WAIT200 beforephysicalcall; API1/4 andGo3/4, stopfirstvalid, fixtureclean | Boundedmanualsynthetic slice accepted. FullM2/S2/S3 permissions/PII/entrypaths/recovery/quality exits remainOPEN; installedrunnerdisabled |
| M3 next | NOT_IMPLEMENTED / NOT_DISPATCHED | Sourceinventory andtestable contract -> boundedSPEC/workorder -> integrated confirm/reject/correct, assignment/deadline/outcome throughbrowser/backend withpermission/stale/concurrency checks |
| M4 | Shadow/pilot NOT_RUN | M2/M3scope pluslockedcorpus/thresholds/people/realdata/account authority; no autosend |
| M5 | Release/operations OPEN | RevisionCI, backup/restore/security/load/rollback/releaseacceptance; no currentpush/deployauthority |

Stop achieved atfirstvalid, remaining3slots notspent orreassigned; oldR0982/2/R099Go1 unchanged. Explanation/nextaction English inthiscase; Vietnamesequality/generalaccuracy NOT_ACCEPTED. Fixture foundstopped afterinterruption; samepinnedID restartedonly forindependentreadback thenremoved withnetwork/volume; originalapp/DB unchanged. RawCLIwrapperstdout expectation flag unresolved, no stdout-contract PASS; actualrequest/response + Rootstrictreceipt/DB review are the evidence. Facebook/ZaloOA parked, docs55/R086native/fence gaps remainOPEN without inventing currentcounts orprerequisitechain. [Independent review](../R100_FINAL_INDEPENDENT_LIVE_REVIEW_2026-10-15.md).

## Historical R100 owner-granted execution route — 2026-10-15

Owner grants **max4 freshcalls untilvalidresult**, superseding max2 proposal/semanticfailurestop only. CurrentmilestoneM2, R099source0640431 compiled/sourceaccepted; changed-source runtimeNOTRUN beforeR100. NewR100 seedb33d1fe83be35aa0e20d18590a5462e7f0ecd6fd precedes BUILD. Fourpredeclaredsyntheticcases/fixedthinkingfalseprompt, oneownedloopbackfixture; Rootaudit eachsemanticinvalid/faithfulusage thenexclusivecontinuation. Stop firstvalidproposal+faithfulknownusage ortransport/metadata/security/ledger/DBfailure, max4/300sec. Onecompile/oneseed/onelive <=4Go, no oldbudgetreset/sourcechange. Rootindependentactualreceipt review needsno duplicatecall. NextM3 humanworkflow afteractualproposal; M1businessparallel, pilotreleaseOPEN/accountsparked. Earlier max2 pendingapproval sections below are superseded planning history, not authority.

## Historical audited delivery route before owner max4 grant — 2026-10-15

| Route | Acceptance still missing | Required next move / owner | Stop condition |
|---|---|---|---|
| M2 current | Real valid proposal on R099 source and faithfully persisted observed usage | Root completes source/compile review commit; owner decides proposed R100 fresh max2 budget; dispatcher pins new seed and source/binary before worker | No grant: no provider/key/fixture. Call1 failure or missing proposal/usage: stop before call2; no retry/tuning |
| M3 next | Human confirm/reject/correct, assignment/deadline/outcome through browser and backend | Root prepares bounded SPEC/work order after valid source-bound proposal evidence | No automatic dispatch or customer send; reject stale/unauthorized writes |
| M1 business / M4 pilot | Named reviewer, SLA, permitted real corpus/holdout and locked quality/budget thresholds | Owner business decisions and Root proposals can proceed alongside synthetic M2 | No real-data/pilot disclosure before scoped authority and acceptance |
| M5 release | Revision-specific CI, recovery/security/operations and release acceptance | Separate release scope after product capability | No push, merge, deployment or production activation under this audit |
| Parked / inherited gaps | Facebook/Zalo OA accounts; docs55 and R086 native/fence remain open | Account route needs owner resumption; other gaps need demonstrated dependency before becoming prerequisites | No silent account activation or generic prerequisite successor; historical55 count is not reverified |

This route is planning only. R094 exact-old-source local proof and R098 real-provider failure receipts remain historical; changed R099 dependencies have no current runtime proof. Full M1–M5 and S0–S7 exits remain open. [Consolidated audit](../R099_ROADMAP_CONTINUITY_AUDIT_2026-10-15.md).

## Historical source candidate and proposed next milestone — R099

R099 source/compile REVIEW_PASS bounded, source0640431/proof302e873, onebuild PASS/0API/runtime. Candidate preserves knownmetadata through safe observed-error andsettlement, adds explicit immutable thinkingfalse once-only option. Actual changed-source proof NOT_RUN, FREEZE/fullM2 OPEN. HistoricalR094 acceptance preserved exactoldrevision; changeddependencies never inherititsruntimeacceptance. R098 real2HTTP200 at758each wereempty/no proposals, DBmetadata lost; budgets2/2/Go4/4 andcleanupverified unchanged. Root owns provider mode/cap/metadata miss.

Concrete R100 packet requests freshmax2 syntheticcalls with explicitthinkingfalse/literalcanonicalJSONfields; call2 onlyifcall1 validproposal+faithfulusage, otherwise stop unusedslot/no tune/retry. No newseed/key/provider/fixture effects beforefreshownergrant and source-boundsignature. Product acceptance stillneeds realproposal withcanonicalevidence/knownobservedusage; then M3 humanreview/assignment/outcome. No generic prerequisite successor, no ETA/percentage, M3/pilot/realchannel OPEN, accountsparked.

## Historical M2 live outcome and finite product repair — R098

R098 actual2HTTP200/qwen3.7,758completioneach, finishlength/answerempty, proposals0. Both rawprovider usages known but adapter/engine losesmetadata into DBUNKNOWN; R098 CHANGES_REQUIRED/FREEZE OPEN. Fixture verifiedclean, app/DB unchanged, physical2/2 andGo4/4 neverreset. Root owns requestmode/cap and metadata review gaps. R096 oldunknowncause unproved, R097 localTLS remains bounded accepted.

Independent new rootcauses justify one coherent R099 source-only product repair: preserve observed model/usage then explicit invalid-output settlement, add explicit immutable thinkingfalse future option supported by Alibaba docs. Zero API/key/DB/fixture/runtime, one compile/static independentreview; no mockgovproof. Fresh changed-request/liveapproval only after concrete candidate; no generic support successor/automatic rerun. Full M2/M3/pilot OPEN, Facebook/Zalo OA parked.

## Historical M2 checkpoint — R096/R097, 2026-10-14

M2 source/local backend prototype accepted at R094; actual live R096 attempts2/2 returned no provider HTTP response, usage/model/billing UNKNOWN, proposals0. R096 remains CHANGES_REQUIRED/FREEZE OPEN. Two consumed slots never reset; old R0932/2 and R095 pre-provider failure retained. Root owns the HTTP1 transport source admission miss; local reproduction identifies copied ALPN mismatch, remote cause remains unproved. Corrected source83c29ce is read-only; exactly one R097 local TLS regression PASS/exit0/noSKIP, proof only protocol tooling. Owned fixture cleaned, existing app/DB unchanged.

R097 independent review accepted bounded local TLS proof; FREEZE local tooling only, material510856d. Finite next route: present concrete corrected TLS live approval packet R098 for fresh max2 synthetic calls. No call before fresh owner grant and new immutable seed/source admission. No generic prerequisite successor: failed local proof or first unknown live result requires stop/reassess. Product outcome still pending is evidence-valid real AI proposal or explicit real-output rejection, then M3 human review/assignment/outcome; full M2/M3/pilot not accepted. Facebook/Zalo OA parked.

## 1. Đọc và áp dụng roadmap này

Đây là kế hoạch thực thi sản phẩm hiện hành, chuyển đánh giá độc lập ngày 2026-10-13 thành mốc hữu hạn và điều kiện bàn giao. Agent mới đọc bootstrap → memory → active handoff → tài liệu này → SPEC/work order của mốc được giao. Không đọc toàn bộ lịch sử mặc định. Roadmap cho phép chuẩn bị kế hoạch trong phạm vi owner đã giao; không phải work order cấp BUILD, provider, dữ liệu khách hàng, account, push hoặc deploy.

- [Định hướng sản phẩm](../../PRODUCT_DIRECTION.md) giữ mục tiêu và ranh giới nghiệp vụ.
- Roadmap S0–S7 (`docs/roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md`) giữ kiến trúc đích, hợp đồng exit toàn stage và lịch sử bằng chứng. M1–M5 là cách tổ chức delivery xuyên stage; hoàn tất một mốc không tự đóng cả stage.
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

Drift M1 đã đối chiếu với source hiện tại: spec database từng nói provider trước candidate; PRODUCT_DIRECTION từng dùng câu blanket “chưa triển khai” và nói chưa cấu hình provider thật. R092 sửa hai câu hiện hành theo source/owner/config readiness. `IMPLEMENTATION_STATUS.knownLimitations` hiện ghi R065 dispatch text là lịch sử/superseded; không coi đó là lệnh hiện hành. Các “next” trong ảnh chụp lịch sử UI được giữ nguyên như lịch sử. Không chạy lại full docs55/link inventory trong R092; số 55 không được khẳng định là count hiện tại. Audit 8 mục `--omit=dev` và 17 mục full audit có scope khác nhau; không gọi chênh số là regression nếu chưa so sánh lock/command/date.

## Historical R093 provider checkpoint và route hữu hạn

[Independent live review](../R093_INDEPENDENT_LIVE_OUTCOME_REVIEW_2026-10-13.md): Alibaba qwen3.7-flash HTTP compatibility observed twice with synthetic data; positive output rejected for invalid ID/missing complaint anchor, neutral PASS. Budget2/2 exhausted; no further R093 retry. R092/R093 are two support scopes, not usable end-user workflow. Next is one coherent M2 application packet including verified local toolchain/isolated DB setup, WAIT_DATA/DENY before disclosure, durable physical-call reservation and strict proposal evidence validation. M3 then completes human review/assignment/outcome in browser. No Go/Docker/MySQL found in checked paths; setup is an explicit prerequisite of that packet, not a new endless support campaign. Full M1/M4/M5 business/pilot decisions remain open.

## Historical R094 machine and bounded M2 route — 2026-10-14

Owner resumes synchronized main and appoints Root orchestrator/reviewer; Luna Max primary, Sol Medium fallback when hard. Luna source intake and Sol design identify durable concurrent physical-call admission, exact evidence and legacy bypass; Root selects Sol Medium. R094 accepted intended SPEC and committed seed activate one coherent M2 source/local synthetic packet, not another provider characterization. Current machine has Go1.27, Node24.19/npm11.17, Docker29.8 and cached mysql:8.0; previous missing-toolchain statements are historical, no install campaign needed. Existing app/DB protected. Full M2/live/M3/pilot remain OPEN; zero new provider/secret/push/deploy effects. R093 positive evidence failure and cap2/2 retained. Next concrete capability: evidence-bound proposal plus readable case queue/detail on one manual synthetic job, with deny/wait zero disclosure. See ../../work_orders/CCMAI_RUNTIME_094_M2_INTERVENTION_APPLICATION.md.

## Historical R094 independent local result — 2026-10-14

Sol Medium source plus one consolidated repair reviewed independently by Root. Actual authenticated app route/MySQL/loopback delivers validated proposal and readable case, with deny/wait, tenant permissions, stable cap, malformed/stale/cancel/transaction failure assertions. Two named mutations killed and exact source restored; final9top-level/four nested adapter cases PASS. Worker6/root4Go consumed10/10; owned fixtures removed and existing containers unchanged. See ../R094_FINAL_INDEPENDENT_M2_REVIEW_2026-10-14.md. This prototype is default disabled/unconfigured in installed app and has zero real provider calls; full M2 and M3/pilot stay OPEN. Next is new bounded live M2 cap/source activation approval, then M3 human workflow; no generic supporting successor or old-budget replay.

## 4. Thứ tự delivery và trạng thái

| Mốc | Kết quả nghiệm thu | Stage liên quan | Dependency | Trạng thái hiện hành |
|---|---|---|---|---|
| M1 | Hợp đồng bản đầu tiên và route an toàn hữu hạn | S0, một phần S1/S7 | Nguồn/review hiện hành | R092 local synthetic intended contract accepted historically; current M1 business/pilot acceptance OPEN. Does not block bounded synthetic M2 |
| M2 | Một đường phân tích có quyết định và đề xuất gắn nguồn | S1/S2/S3 | M1 contract; prerequisite thật sự chặn | R100boundedrealpathREVIEW_PASS/FROZEN: actual1call validproposal, faithfulKNOWN360/261usage, consumed1of4, cleanup/protectedPASS. FullM2/S2/S3 OPEN; M3planning next |
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

Owner cho phép đồng bộ repo không đồng nghĩa bật production. Nếu chỉ chuyển máy để code, có thể clone và tiếp tục trước M5 theo `AGENTS.md`; phải giữ nhãn development và các gap. M5 là nghiệm thu bản phát hành/vận hành, không phải prerequisite để mở IDE trên máy mới.

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
| D05 provider/model/key readiness | R100 current-source realAlibaba/qwen3.7-flash HTTP200, explicitthinkingfalse, validproposal andKNOWN360/261persistedusage; dollarbillingUNKNOWN | Ownermax4 stopsfirstvalid at1/4; no moreR100calls. Futureprovider effects require separatelybounded applicableauthority, neverreuseclosedmarkers | Boundedmanualsynthetic accepted; fullM2/pilotOPEN |
| D06 topology/recovery/fence | In-process coordinator + DB parent-lock admission; orphan running fail-closed; distributed cancellation, crash recovery, reclaim và mixed-binary rollout không có | Chỉ đề xuất một process/one selected manual synthetic job; giữ fail-closed, không auto-reclaim; không mở lại accepted inheritance review | Multi-process/restart/live-data readiness; không chặn route local synthetic |
| D07 Facebook/Zalo OA | OWNER_DEFERRED | Chỉ owner mở lại; không đưa vào prerequisite của Pancake synthetic; giữ nguyên evidence đã chấp nhận | Live riêng hai kênh |
| D08 release target và vận hành | OPEN | Chốt môi trường/người vận hành trước M5; không tự deploy | M5 release |

**Next allowed planning action:** R100 boundedrealproposal+faithfulusage accepted; Root commits finalreview/cleanup/roadmap then prepares M3sourceinventory, testablehumanreview/correction/assignment/deadline/outcome contract andboundedSPEC/workorder. Browser/backend negativeproof required. M1people/SLA/corpus/threshold decisions parallel, pilot/releaseOPEN; no generic prerequisite successor orfurtherR100call.

## 15. Source Authority và Evidence Comparison

Nguồn được đọc có chọn lọc tại baseline, không phải full-corpus audit: AGENTS.md, manifest/policy, bootstrap/memory/handoff R091, implementation current projections, roadmap S0–S7, PRODUCT_DIRECTION, UI roadmap current checkpoint, baseline S0, database filter spec, shared learning điều phối; source job model, result routes/handlers, analyzer provider boundary và workflow backend. Review R088/R090/R091 cung cấp acceptance kế thừa, không thay live proof.

Evidence Comparison: source có lazy provider initialization sau khi chuẩn bị snapshot; database spec đã được sửa để phản ánh đúng ordering nhưng vẫn ghi rõ các gate chưa có. R065 là historical/superseded trong current knownLimitations. Provider config có secret-safe metadata nhưng chưa live-verified; authorization không tạo provider proof. M1 contract là candidate local-synthetic route, không phải source/product exit. Giữ các gap có gắn revision; không suy từ tìm kiếm chọn lọc thành “không tồn tại toàn repo”.

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

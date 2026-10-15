# R101 — Bàn giao Claude kiểm tra UI M3 và trả bằng chứng

Status: DRAFT
Handoff readiness: PREPARED_FOR_OWNER_ASSIGNMENT
Date: 2026-10-15
Parent work order: docs/work_orders/CCMAI_RUNTIME_101_M3_HUMAN_WORKFLOW.md
Tranche: CCMAI-RUNTIME-101 (REVIEW; không mở tranche BUILD mới)
Risk ceiling: R2
Evidence operator: Claude, khi chủ dự án giao packet này
Independent reviewer / commit steward / closer: Codex /root
Commit mode: WORKER_MUST_NOT_COMMIT
providerExecutionAuthority: NONE

## 1. Kết quả cần bàn giao

Kiểm tra ứng dụng thật trong trình duyệt với backend thật và MySQL synthetic của R101: sửa đánh giá → giao việc và hạn xử lý → ghi kết quả → tải lại thấy dữ liệu và lịch sử còn nguyên. Bổ sung bằng chứng quyền chỉ đọc, xung đột phiên bản và màn hình hẹp. Trả báo cáo cùng ảnh/log có thể đối chiếu. Không sửa code, không tự kết luận M3 hoàn tất.

Đây là packet thu thập bằng chứng REVIEW bổ sung, kế thừa SPEC và work order R101, không thay thế authority seed, không đổi implementationWorker/reviewer trong seed. `DRAFT` biểu thị packet bàn giao chưa có kết quả; chủ dự án giao packet cho Claude là điểm xác nhận người thực hiện. Không dùng packet này để khởi động BUILD hoặc mở rộng quyền. Root đánh giá kết quả độc lập trước khi chấp nhận.

## 2. Đọc trước và xác nhận tiếp nhận

Làm First-Request Protocol trong AGENTS.md: manifest, policy, workspace rules, bootstrap read model, current state, phần Current Truth của memory, active handoff, implementation status, docs index; chạy workspace doctor. Không đọc toàn bộ lịch sử nếu không có mâu thuẫn.

Đọc các file sau:

- `CVF_SESSION/authority/CCMAI-RUNTIME-101.json` — bất biến, không sửa.
- `docs/specs/M3_HUMAN_WORKFLOW_R101_SPEC_2026-10-15.md` — H01–H08.
- Parent work order ở đầu packet.
- `docs/reviews/R101_REPAIRED_SOURCE_ADMISSION_2026-10-15.md`.
- `docs/reviews/R101_FINAL_LOCAL_FIXTURE_ROUTE_DISPOSITION_2026-10-15.md`.
- `docs/reviews/R101_BASELINE_FAILURE_DISPOSITION_2026-10-15.md`.
- `docs/reviews/probes/r101_root/browser_policy_stop_checkpoint.json`.
- `docs/reviews/probes/r101_root/fixture_owner.json`, `review_counters.json`, `formatted_source_manifest.json`, `review_campaign.py`.

Phát CVF Agent Declaration với giá trị hiện hành. Ghi acknowledgment vào báo cáo mới: vai trò chỉ thu thập bằng chứng REVIEW, source/binary hash, tài nguyên được phép, counters trước chạy. Nếu canonical state đã đổi, đối chiếu lại trước khi dùng các giá trị chụp dưới đây; không chọn một bản im lặng khi có drift.

## 3. Điểm xuất phát và giới hạn bằng chứng

- Source commit: `200df3e4a7bc2a326425922a5230508a4fd9b0fe`.
- Binary có sẵn: `.cvf/runtime/r101_worker/r101_handlers.test.exe`.
- SHA256 binary: `4c7575e180dd198d995bfe09bdc45061740a987be8ec0765f02b203e1c251f31`.
- Compile, 5 test UI mới, frontend build: PASS đã có; không chạy lại mặc định.
- Native MySQL: 6 nhóm top-level + 4 nested PASS, không SKIP; không cần chạy lại.
- Full frontend suite: 438/441 PASS, 3 lỗi Dashboard baseline đã tái hiện ở source trước. Giữ nguyên kết quả FAIL; sửa test Dashboard nằm ngoài packet này.
- Browser: NOT_RUN. Computer Use của Codex bị runtime dừng vì không hỗ trợ URL-policy enforcement cho Windows browser hiện tại. Đây không phải thiếu quyền người dùng.
- Dùng browser tool được môi trường Claude hỗ trợ và cho phép, hoặc người dùng thao tác bằng tay. Nếu chính môi trường Claude cũng chặn vì policy/capability, dừng và báo BLOCKED; không đổi sang stack khác để vượt chặn. Ghi rõ ai quan sát/thao tác, ảnh từ đâu; không biến lời kể thành bằng chứng agent trực tiếp quan sát.
- Đã dùng workerGo1, reviewerGo1, aggregateGo2/16, fixture2/2, Node checks3/6, build1/1; API0/mock0. Không reset counters.
- FREEZE OPEN; full M2/M3, WAIT_DATA rerun, M1 nghiệp vụ, pilot/release OPEN; Facebook/Zalo OA parked.

## 4. Phạm vi được phép

Chỉ source inspection, khởi động lại đúng fixture đã giữ, một lần chạy browser-fixture binary bằng reviewer budget hiện hữu, một Vite dev server loopback với dependencies cache, thao tác UI trên dữ liệu synthetic, và ghi báo cáo/bằng chứng mới.

Đầu ra mới được đặt tại:

- `docs/reviews/R101_CLAUDE_BROWSER_RETURN_2026-10-15.md`.
- `docs/reviews/probes/r101_claude/` — captures, log đã lọc secrets, manifest/hash, kết quả từng bước, process/cleanup receipts.
- Runtime logs có thể đặt dưới `.cvf/runtime/r101_claude/`.

Cập nhật shared counters trước binary launch bằng harness hiện hữu; giữ log cũ, không ghi đè native/baseline evidence. Nếu cần chỉnh port metadata của cùng container sau restart, lưu bản trước/sau vào evidence mới rồi cập nhật duy nhất trường port của `fixture_owner.json` sau khi xác minh đủ ownership. Không sửa source, seed, policy, gate, workflow, dependencies hay canonical phase/status. Root đồng bộ continuity/catalog và commit sau review.

Không được: provider/mock-provider/API key/credential thật; đụng app hoặc DB đang dùng; real customer/channel; gửi tin; bật installed runner; install/download/pull; migrate DB ngoài fixture; listener ngoài loopback; tạo fixture thứ ba; commit/push/merge/deploy; tự approve/freeze. Không chạy docker compose up/down hoặc prune.

## 5. Khởi động đúng fixture hiện có

Terminal luôn ở project root. Máy cần đúng checkout và local resources; nếu chạy máy khác thiếu binary/container thì báo BLOCKED_LOCAL_RESOURCES, không tạo thay thế.

Fixture đang STOPPED, phải giữ nguyên identity:

- Name/volume: `ccmai-r101-review-final-20261015`.
- Container: `e1555e03d69360e1cd7729afd9e52856443f5172d76454653beec981a7d6477a`.
- Network: `35c3959ac41517050defad05cbad7a08717b24c8879683e25baf3e2702625e98`.
- Image: `sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b`.
- Label: `ccmai.tranche=CCMAI-RUNTIME-101`.
- Database: `r101_fixture`; tài khoản/mật khẩu chỉ dành fixture có trong harness/source, không đọc `.env` hay credentials ứng dụng thật.

Trước effects, kiểm tra đủ 17 source hashes từ manifest và binary hash; kiểm tra Docker ID/label/image/mount/network exclusive bằng safe inspect fields. Không dump `.Config.Env`. Ghi inventory an toàn của container khác trước/sau; khác biệt do bên ngoài phải báo thật, không tuyên bố unchanged khi không có bằng chứng.

Sau khi xác minh identity mới chạy:

```powershell
docker start e1555e03d69360e1cd7729afd9e52856443f5172d76454653beec981a7d6477a
python docs/reviews/probes/r101_root/review_campaign.py ready_existing
python docs/reviews/probes/r101_root/review_campaign.py serve
```

Không chạy nhánh `setup` hoặc `native`. Port MySQL cũ 54884 chỉ là lịch sử: inspect sau restart, chỉ chấp nhận `127.0.0.1`, reconcile cùng identity nếu Docker đổi port. `ready_existing` so protection baseline lịch sử: nếu fail do container ngoài bị người dùng thay đổi, dừng và báo khác biệt, không sửa baseline để làm PASS. Nếu readiness fail, không tạo/recreate fixture; báo lỗi thật.

`serve` tăng reviewer/aggregate Go trước khi launch. Lưu counters trước/sau và process receipt. Dự kiến sau một launch: worker1/reviewer2/aggregate3; không launch lại tự động nếu fail. Backend-fixture sống tối đa 30 phút, test timeout 35 phút. Thiếu thời gian hoặc server hết hạn thì báo phần chưa chạy, không replay tự động.

Đọc `docs/reviews/probes/r101_root/browser_ready.json` lấy `apiUrl`, `ownerEmail`, `reader`, tenant/case. Chọn một port Vite rảnh, ví dụ 3011; nếu có process dùng thì chọn port rảnh khác, không kill process đó. Trong terminal riêng:

```powershell
$env:VITE_API_TARGET = '<apiUrl thực từ browser_ready.json>'
npm --prefix frontend run dev -- --host 127.0.0.1 --port 3011 --strictPort
```

Ghi port/PID/command thực tế. Chỉ đi qua `/api` proxy tới fixture; không truy cập MCP/OAuth, channel sync, jobs hay setup/reset demo. Browser mở origin Vite mới; không dùng app localhost cũ. Login owner bằng ownerEmail thực; password synthetic `R101SyntheticOnly!123`. Reader email là `<reader UUID>@example.invalid`, cùng password synthetic. Nếu trang hiện setup thay vì login hoặc tenant không đúng, dừng; không chạy setup trên app khác.

## 6. Ma trận kiểm tra và captures

Chỉ đánh dấu PASS khi đã quan sát kết quả. Mỗi hàng ghi action, expected/actual, case/version, ảnh/log, thời gian và người thực hiện. UI dùng tiếng Việt, desktop khoảng 1440×900 và narrow khoảng 390×844; ghi kích thước thực.

| ID | Thao tác | Kết quả cần quan sát |
| --- | --- | --- |
| B01 | Login owner → menu hồ sơ can thiệp → queue → đúng case synthetic | Đúng tenant/case, nguồn/proposal/lịch sử hiển thị; dữ liệu ban đầu được chụp trước thay đổi |
| B02 | Mở cùng case pending ở hai tab; tab A CORRECT với reason, giải thích và next action | Một thay đổi thành công; phần AI gốc riêng với sửa của người, history có actor/time/reason |
| B03 | Tab B chưa reload gửi action từ phiên bản cũ nếu UI cho phép | Conflict/409 hoặc bảo vệ rõ ràng trước submit; không báo thành công hoặc tự replay. Nếu UI tự refresh nên không tạo được stale request, ghi NOT_RUN stale-server thay vì đoán PASS |
| B04 | Trên trạng thái mới nhất, ASSIGN user thuộc tenant và hạn tương lai | Người xử lý/hạn/trạng thái đúng; reload vẫn còn; không gửi thông báo ra ngoài |
| B05 | RESOLVE với outcome; reload detail và tìm qua terminal filter | Outcome và trạng thái lưu thật; history trước đó còn nguyên |
| B06 | REOPEN với reason/hạn tương lai; reload | Trở lại trạng thái xử lý; outcome cũ còn trong history, AI proposal không bị ghi đè |
| B07 | Logout owner, login reader; mở queue/detail | Xem được theo quyền, không có thao tác ghi khả dụng; không dùng owner token hay giả lập interception |
| B08 | Narrow viewport: queue/detail, proposal/human/history và form nếu owner | Dễ đọc, controls dùng được, không che nội dung/hành động quan trọng |
| B09 | Một lỗi tải thực, an toàn: browser offline rồi reload detail, online lại và reload | Lỗi hiển thị thật, không success giả hoặc tự replay action; không sửa backend/firewall/Docker network. Nếu tooling không hỗ trợ offline hợp lệ, ghi NOT_RUN |
| B10 | Nếu còn thời gian, mở WAIT_DATA fixture có sẵn, owner/deadline/escalation và reload | Không thể confirm/resolve sai; không gọi AI; rerun integration vẫn OPEN. Không tạo thêm fixture để bổ sung |

B02/B03 thực hiện trước B04 để giữ được pending case. Trước và sau luồng chụp nội dung original proposal và history, đối chiếu không đổi bằng dữ liệu thực (UI/API đọc trên fixture nếu cần). Không dùng network interception/mock response để tạo kết quả. Không double-submit cố ý trên case đã mất điều kiện; thiếu test nào ghi thiếu. Native proof được dẫn nguồn kế thừa, không ghi là Claude đã rerun.

## 7. Stop, cleanup và báo cáo trả về

Dừng khi hash/ownership/tenant/protection mismatch, policy browser block, listener nonloopback, provider/network ngoài scope, budget hết, hoặc cần sửa source. Giữ failure receipts. Lỗi ứng dụng chỉ ghi defect/repro; không sửa lén. Không gọi test/build khác và không sửa 3 Dashboard tests trong packet này.

Cuối campaign, dừng chính Vite và backend process đã launch sau khi xác minh PID/command, dừng đúng owned MySQL container; giữ container/network/volume để Root quyết định cleanup. Không kill toàn bộ node/go/browser, không đóng tab của người dùng. Ghi process còn sống nếu không dừng an toàn được. Không delete tài nguyên hoặc dữ liệu ngoài scope.

Báo cáo trả về phải có:

1. Declaration/acknowledgment, thời gian, source SHA và 17 hash/binary verification; git dirty baseline để tách thay đổi tồn tại trước.
2. Cách điều khiển browser, ai quan sát; actual URL/viewport; fixture IDs, process PIDs và counters trước/sau.
3. B01–B10: PASS/FAIL/NOT_RUN/BLOCKED với evidence path cụ thể; ảnh không lộ token, cookie, password thật hay dữ liệu ngoài fixture.
4. Defects theo mức độ, thao tác tái hiện, expected/actual; raw error/log có lọc secrets. Giữ mọi lần fail, không chỉ kết quả cuối.
5. Cleanup/protected inventory, danh sách file mới/sửa; hash manifest ảnh/log.
6. Disposition: `EVIDENCE_COLLECTED_PENDING_INDEPENDENT_REVIEW` hoặc `BLOCKED_WITH_REASON`. Không tự ghi REVIEW_PASS/FROZEN.
7. Ghi rõ 3 full-suite baseline failures vẫn OPEN, browser chỉ chứng minh human software workflow synthetic; không chứng minh AI governance, M2 live integration, full M3, pilot hoặc release.

Root nhận báo cáo, kiểm chứng source/captures và giới hạn kết luận; sau đó mới quyết định repair scope riêng, continuity/catalog/gates/commit. Không cần cung cấp API key hoặc tài khoản Facebook/Zalo OA cho công việc này.

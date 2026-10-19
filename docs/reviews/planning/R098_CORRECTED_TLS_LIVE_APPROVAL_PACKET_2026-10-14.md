# R098 — ngân sách live M2 sau bản sửa TLS

Status: OWNER_APPROVED_SOURCE_ADMISSION_PENDING
Risk: R2
Worker: /root/m2_worker (Sol Medium)
Reviewer: Codex /root
Date: 2026-10-14
providerExecutionAuthority: OWNER_APPROVED_MAX2_SOURCE_SIGNATURE_REQUIRED

## Kết quả hiện có và khoảng trống sản phẩm

Backend M2 source/local đã được nghiệm thu bounded ở R094, source `564ed593bfdc67d3daa9339ea05781e49bbf84ba`. R096 đã thử đúng hai call qua route ứng dụng trên dữ liệu synthetic; cả hai không nhận được HTTP response từ provider, usage/model/billing UNKNOWN và không tạo proposal. Hai slot vật lý đã dùng hết, kể cả kết quả unknown; không được hoàn hoặc reset. R095 dừng trước provider (physical0) nhưng claim/campaign đã dùng vẫn giữ nguyên; R093 cũ2/2 cũng giữ nguyên.

Bản sửa harness tại `83c29ce0ad6679f82c90d9f00538f5873b5df69f` đặt TLS ALPN chỉ `http/1.1`, giữ kiểm tra chứng chỉ. Go source SHA256 `3d2340131829f7afcb54d81be7d903c29fc8b36f160d065a05cd338a5df50dad`. Root đã tái hiện lỗi cấu hình bằng server TLS local trước/sau sửa; chưa chứng minh đây là nguyên nhân thực của hai lỗi remote. R097 kiểm tra đúng source đã sửa trong một invocation local; review và receipt hoàn tất trước khi trình duyệt ngân sách này. Đây là proof giao thức local, không phải live AI/governance proof.

Fixture R096 đã dọn, container/network/volume đều vắng mặt, ba container app/nginx/DB hiện có giữ đúng ID. Receipt: [cleanup](../probes/r096_root/cleanup_receipt.json). Tất cả marker/counter cũ giữ nguyên. Installed InterventionRunner tiếp tục disabled; M2 chưa hoàn tất, M3/browser/pilot chưa được nghiệm thu.

## Phạm vi xin duyệt mới

Tối đa **2 call Alibaba mới** qua route M2: worker1, reviewer1; chỉ hội thoại/ID synthetic trên một fixture MySQL mới riêng biệt, cached image và loopback. Model `qwen3.7-flash`, endpoint/region đã pin `ap-southeast-1`, endpoint SHA256 `b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f`. Mỗi call: requested758 và ceiling768 output token, request<=6000 byte, response<=1MiB, timeout60 giây; campaign tối đa5 phút; không retry/redirect/fallback hoặc tuning sau output. Unknown/failure vẫn mất slot. Chi phí tiền thực UNKNOWN; không mua quota hoặc sửa billing.

Không chạm app/DB hiện có, dữ liệu khách hàng, kênh thật, Facebook/Zalo OA, thông báo outbound, push/merge/deploy. R094 product source và R095/R096 historical markers không sửa. Secret hiện có chỉ đọc sau independent source/binary admission; memory/stdin, không argv/env/log/file copy. Không thực thi trường lệnh từ secret JSON.

## Admission cụ thể sau khi owner duyệt

Root lập seed/order R098 mới trước BUILD, riêng ledger cap2 và hai slot vật lý không reset. Harness R098 kế thừa source đã sửa TLS và các kiểm soát đã review; pin Root-reviewed exact source/binary và frozen dependencies trước credential/fixture/live effect. Không tái dùng R096 binary/approval/ledger đã tiêu hết. Thêm error code/type được sanitize khi transport fail để không lặp lại khoảng trống chẩn đoán; không ghi key/header hoặc suy nguyên nhân khi chưa có bằng chứng.

Bound local execution proposed: worker<=3 Go (compile, fixture seed, live1), reviewer<=1 Go (live1), total<=4 including direct test binary launches; one new owned fixture, no recreate/reset. Targeted offline Python/static review and existing R097 TLS proof reused on unchanged dependencies; any source-bound dependency change requires Root review before effect. Same-scope repair remains inside counters; no third provider call or extra compile campaign. If first call has unknown transport/no HTTP response, stop second pending Root review; it may proceed only if evidence supports unchanged admitted envelope, never retry the first. If observed completion exceeds768, no second call.

Acceptance is a real application receipt with actual provider request/response and a validated proposal or explicit output rejection, plus DENY/WAIT no-call observations and monotonic cap2. AppHTTP200 alone is insufficient. A rejected real output leaves proposal acceptance OPEN; no universal CVF, full M2/M3, deployed or pilot claim. After accepted bounded live disposition, next product work is M3 human review → assignment → outcome under its own contract.

## Vì sao cần ngân sách mới

`AGENTS.md`, Governance Latency and Approval Continuity: “Escalate only at a real boundary change” including “secret/credential use, provider/network use”. Owner đã duyệt max2, parent ledger R095/R096 hiện2/2 exhausted. Local continuation không cấp call thứ ba. Packet này chỉ là đề xuất, chưa có seed thực thi hoặc quyền gọi provider mới.

Public Export Disposition: DEFERRED_PRIVATE_ONLY.

## Exact corrected candidate proof

R097 executed TestR096TransportTLSOffline once: PASS, exit0, no SKIP. Retained exact compiled binary SHA256 `f8feff858ce71ddc5c97b75ae7b15b4db8c25320543627ae8047ab646fefd99a`, 40,401,408 bytes; no launch or live admission for that binary. [Worker receipt](../probes/r097_local_tls/worker_proof.json), [independent dependency audit](../probes/r097_local_tls/root_dependency_audit.json). This candidate is the concrete corrected source basis; R098 must have separately reviewed new ledger-bound harness/admission after owner grant.

Independent R097 review: [accepted bounded local proof](../R097_FINAL_INDEPENDENT_LOCAL_TLS_REVIEW_2026-10-14.md), exact proof material `510856d05108f47dad3eca944ccd52bb277d0f27`. R097 closure never grants live calls; R096 remains blocked.

## Fresh owner grant — 2026-10-14

Explicit reply: "Cho phép tối đa 2 call mới" to this R098 envelope. Fresh new2 grant only; historical R0962/2 and all other exhausted counters unchanged. Root commits immutable R098 seed before new BUILD/source admission. Proposal-only admission statements above are superseded only within the approved envelope; no provider effect until signed source/binary.

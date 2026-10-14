# R100 — đề nghị live M2 với thinking tắt rõ ràng

Status: PROPOSED_OWNER_APPROVAL_REQUIRED
Risk: R2
Worker: /root/m2_worker (Sol Medium)
Reviewer: Codex /root
Date: 2026-10-14
providerExecutionAuthority: NONE_PENDING_NEW_OWNER_GRANT
Source candidate: 0640431dc3c31b3b8403d775ae50a6bc6dfdf601

## Kết quả thực và bản sửa cụ thể

R098 đã gọi đúng2 lần, HTTP200/modelqwen3.7-flash cả hai, completion758 mỗi lần; finishlength/answerempty/proposals0. Wire báo prompt272/266 và completion758/758; DB lạiUNKNOWN vì adapter bỏ metadata khi answer rỗng. R098CHANGES_REQUIRED/FREEZE OPEN. Fixture đã dọn, app/DB/nginx hiện có giữ ID; toàn bộ marker/budget cũ giữ nguyên, không hoàn lại call unknown/failure. [Actual review](../R098_FINAL_INDEPENDENT_LIVE_REVIEW_2026-10-14.md), [wire/application receipts](../probes/r098_root/live_outcome_summary.json), [cleanup](../probes/r098_root/cleanup_receipt.json).

R099 đã chuẩn bị source giữ parsed model/usage qua safe observed-response error, settle metadata trước OUTPUT_INVALID; transport/unparsed/mismatch vẫn UNKNOWN. Lựa chọn immutable WithAlibabaQwen37OnceThinkingDisabled chỉ áp dụng AnalyzeChatOnce, clone provider, kiểm tra model/endpoint/tokens, thêm top-level enable_thinking:false khi gọi rõ ràng; chưa caller nào opt-in và startup vẫn disabled. Build source-only ai/engine/handlers PASS, workerGo1/1,0API/key/DB/runtime. Hashes: openai_compatible.go a1ac47fcba0e813d139b2ab4dcd47d872ac5614b9d5528c805cd007df34e99fd; chat_once_response.go 7a809aea477017ee6205a47803f4caac68064fa610e7b7dc7679cf236734246a; intervention.go 59bf11d0e74e6f0cb39ab52a006a32c5dd9de26be320680b9a2d87b577b5ecf3. [Compile receipts](../probes/r099_source/pre_build_identity.json), [source admission](../R099_PRE_BUILD_INDEPENDENT_SOURCE_REVIEW_2026-10-14.md). Đây chưa là proof runtime cho source đã thay đổi; R094/R098 không được dùng để chứng minh runtime mới.

Alibaba docs checked2026-10-14 xác nhận qwen3.7-flash bật thinking mặc định và hỗ trợ tắt ở hybrid mode; raw HTTP dùng enable_thinking ở top-level. [Official thinking docs](https://www.alibabacloud.com/help/en/model-studio/deep-thinking), [official Chat API](https://www.alibabacloud.com/help/en/model-studio/qwen-api-via-openai-chat-completions). Chất lượng/answer khi modefalse vẫn NOT_RUN, không cam kết proposal sẽ đúng.

## Ngân sách và điều kiện dừng mới xin duyệt

Tối đa2 call Alibaba M2 mới, synthetic only, qwen3.7-flash và cùng endpoint/regionSingapore pinnedSHA b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f, **enable_thinking:false** rõ ràng. Requested758/ceiling768 output,<=6000B request/1MiB response/60sec each/300sec campaign, no retry/redirect/fallback/tuning/refund. Worker1 complaint conversation; reviewer1 separate waiting conversation **chỉ nếu worker1 tạo proposal hợp lệ, model/usage/snapshot đúng và không vượt768**. Nếu answerempty/invalid/truncated, transport/noresponse, metadata bị mất, source/model/endpoint mismatch hoặc không có proposal: giữ failure, dừng call2, không chỉnh prompt/call lại. Call2 đã được phép tối đa nhưng không bắt buộc tiêu hết.

Một NEW fixture MySQLcached/loopback riêng, một job cap2/ledger2 slots freshRoot-owned. Không chạm app/DB/nginx hiện có, khách hàng/kênh thật, Facebook/Zalo OA, outbound, billing/account/quota, push/merge/deploy. Secret chỉmemory/stdin sau reviewed source/binary admission, không argv/env/log/filecopy/commandfield. Tiền billedUNKNOWN, provider reportedusage không phải invoice. R093old2/2, R095preprovider claim, R096old2/2/Go7/7, R097localGo1/1, R098new2/2/Go4/4 và R099Go1/1 giữ nguyên.

## Source-bound route sau duyệt

Root mới lập immutable R100 seed/order và canonical BUILD; source packet này không cấp quyền thực thi. New harness kế thừa unchanged TLS helper/sourceSHA3d234013, giữ current R0993 source hash + other10 historical R094dependencies làm current exactidentity; old095/096/098 harness không sửa hoặc replay. Explicit opt-in clone used, recorder verifies request enable_thinkingfalse and neverrewrites payload. Future prompt declares exact canonical JSON keys/schema/conversation/digest/evidence_refs message_id and quote, without aliases/markdown and without relaxing strict validation; không đổi R099 prompt hoặc sửa R098 sau output. Literal field clarification là proposed new source-bound request, không phải thử lại output cũ.

Finite Go budget proposed worker3 (compile/seed/live1) and reviewer1(live1), aggregate4 includes directbinarylaunches; one fixture, no recreation/recompile after failure. Root consolidated source audit beforecompile, signed source/binary/manifest beforekey/configprecheck/setup. New physical/role/campaign markers permanentlyconsume attempts and remain aftercleanup. Root stops/adjudicates failure without another call. No mockprovider as governance proof. Acceptance requires actual current-source valid proposal with canonical evidence and faithful metadata, DENY/WAIT zero provider before positives, capmonotonic and clean protected state; actual outputrejection may be recorded but fullM2 staysOPEN. Next productmilestone remainsM3 human review/assignment/outcome only after suitableM2 evidence.

## Vì sao cần duyệt mới

[AGENTS.md](../../../AGENTS.md) requires boundary escalation for “secret/credential use, provider/network use”. R0982/2 hết và request mới thêm explicit thinkingfalse/fieldclarification; local source continuation không cho phép call thứ3 hay đổi request trong campaign đã xong. Đề nghị mới chỉ cấp tối đa2 effect có điều kiện dừng ở trên, không reset seed/budget cũ.

Public Export Disposition: DEFERRED_PRIVATE_ONLY.

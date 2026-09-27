# CCMAI-RUNTIME-001: S0 baseline và S1 sync truth

**Trạng thái:** BUILD_COMPLETE / REVIEW_PENDING · **Rủi ro:** R2 · **Ngày:** 2026-09-27.

## Authority

Owner yêu cầu bắt đầu nâng cấp CCMA theo roadmap sau khi hoàn tất dọn nền CQA. Work order này kế thừa roadmap và quyết định SoT-first, nhưng chỉ cấp quyền cho baseline synthetic và tính trung thực của đồng bộ.

Authority spec: `docs/specs/RUNTIME_FOUNDATION_S0_S1_2026-09-27.md`.

## Allowed scope

- `backend/engine/sync.go`, `backend/engine/scheduler.go`, test unit liên quan và corpus dưới `backend/engine/testdata/`;
- UI trạng thái đồng bộ trong `frontend/src/views/Channels.vue` và `frontend/src/views/Channels/ChannelDetail.vue`;
- roadmap/spec/work-order/review, catalog/index, continuity và implementation status;
- build/restart local app + MySQL trên Compose project `ccma` với dữ liệu phát triển hiện đang rỗng.

## Forbidden scope

- Không gọi provider AI/LLM, không dùng credential/provider key và không gửi dữ liệu khách hàng ra ngoài.
- Không sửa adapter credential, không sync kênh thật, không tự gửi phản hồi khách hàng.
- Không triển khai S2/S3/S5, không sửa CVF core, không deploy/push/public release.
- Mock/fake chỉ được dùng để test data-sync logic; không được dùng làm governance proof.

## Evidence và failure conditions

- Ghi baseline database/corpus và claim boundary.
- Test Go cho helper/trạng thái/checkpoint; build backend và frontend.
- Compose restart, migration/startup log sạch, pricing sync vẫn tắt.
- Catalog check, docs build, workspace doctor và diff check.
- Dừng nếu `partial` vẫn dời checkpoint, vẫn kích after-sync job, UI gọi partial là success, hoặc có network/provider call ngoài build dependency fetch.

Role route: ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → REVIEWER (độc lập cho R2 trước FREEZE) → COMMIT_STEWARD → SESSION_SYNC_STEWARD → ORCHESTRATOR.

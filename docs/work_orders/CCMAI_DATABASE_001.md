# CCMAI-DATABASE-001: database mặc định và luồng lọc MySQL

**Trạng thái:** BUILD_COMPLETE / OWNER_REVIEW_ACCEPTED · **Rủi ro:** R2 · **Ngày:** 2026-09-27.

## INTAKE

Owner yêu cầu điều chỉnh luồng database cho kiến trúc SoT/filter/AI, đổi database mặc định thành `CCMA`, và đánh giá phần có thể học từ `Blackbird081/pg-jev`.

## DESIGN / SPEC

Giữ MySQL 8 và GORM. Fresh install dùng schema mặc định `CCMA`; bản cài có `.env` giữ schema đã khai báo. Luồng đích dùng SQL cho candidate thô, Go cho snapshot/policy/rules/admission và chỉ resolve provider sau khi item thực sự cần LLM. Học batching, projection, cache/content digest, spend guard và observability từ `pg-jev`; không tích hợp PostgreSQL extension hoặc TypeSafe service. Authority: `docs/specs/DATABASE_FILTER_PIPELINE_2026-09-27.md`.

## BUILD scope

Allowed paths:

- `.env.example`, `docker-compose.yml`;
- `backend/config/config.go`, `backend/config/config_test.go`;
- `docs/guide/installation.md`, `docs/reference/env-vars.md`, `docs/PRODUCT_DIRECTION.md`;
- roadmap/decision/spec/work-order indexes and `AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md`;
- project continuity and `IMPLEMENTATION_STATUS.json`.
- `docs/reviews/DATABASE_DEFAULT_AND_FILTER_DESIGN_2026-09-27.md` để ghi bằng chứng BUILD.

Không sửa schema/table, không di chuyển/xóa database, không sửa provider runtime, không gọi API, không đưa credential vào repo. Integration-test fallback database `cqa` là test fixture độc lập và không thuộc default production config.

Owner-directed validation follow-up (2026-09-27): được phép tạo một Compose project, container, volume và schema hoàn toàn cô lập bằng dữ liệu giả để kiểm chứng fresh install. Phép thử có thể tạo/ghi/xóa một probe table trong schema cô lập nhằm xác minh quyền AutoMigrate/DML của application user; phải dùng tên project riêng, không kết nối database hiện hữu, không dùng credential thật, và phải dọn container/volume cùng `.env` tạm sau phép thử.

Owner-directed development setup follow-up (2026-09-27): workspace này không có dữ liệu CQA cần giữ hoặc migration. Được phép đổi default application database user từ `cqa` sang `ccma`, tạo `.env` git-ignored bằng secret sinh cục bộ, và khởi tạo/giữ Compose MySQL volume mặc định làm database phát triển thật của project. Không dùng dữ liệu hay credential production; không chạy toàn bộ application stack nếu chưa cần.

Nếu live setup phát hiện healthcheck báo sẵn sàng trước khi MySQL hoàn tất init user/schema, được phép sửa healthcheck để xác thực bằng application user và database đích. Acceptance phải đợi authenticated query thành công, không chỉ dựa vào unauthenticated `mysqladmin ping`.

Nếu restart trên persistent volume phát hiện AutoMigrate tạo lại index đã tồn tại, được phép sửa đường migration thủ công thành idempotent và trả lỗi thật thay vì bỏ qua mọi lỗi DDL. Xác minh bằng ít nhất một lần rebuild/restart trên chính volume phát triển.

Nếu lần khởi động backend để AutoMigrate tự phát sinh kết nối ngoài không cần cho database setup, phải ghi nhận sự kiện, tắt mặc định kết nối đó và restart xác minh. Không được diễn giải metadata fetch là provider/LLM call hay governance proof.

## Evidence / failure conditions

Chạy `go test ./config`, render/validate Compose config với secret giả cục bộ, catalog `-Check`, workspace doctor và `git diff --check`. Dừng nếu default không nhất quán hoặc Compose không render. Owner review đã loại migration CQA khỏi phạm vi workspace này; claim về `pg-jev` vẫn phải giữ đúng ranh giới thiết kế, không biến thành dependency runtime.

Validation follow-up phải chứng minh schema được tạo đúng tên/case `CCMA`, charset/collation dự kiến, không tự tạo schema `cqa`, application user kết nối được và hoàn tất probe DDL/DML. Kết quả và cleanup được ghi vào BUILD evidence; đây không phải migration test hay bằng chứng CVF runtime governance.

Role route: ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD → ORCHESTRATOR. Owner reviewed and corrected the live-development database scope; no separate database review remains. Runtime machine-gate and provider behavior remain outside this work order.

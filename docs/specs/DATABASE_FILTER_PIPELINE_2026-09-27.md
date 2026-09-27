# Đặc tả luồng MySQL và lọc trước AI

**Trạng thái:** BUILD_COMPLETE / OWNER_REVIEW_ACCEPTED · **Ngày:** 2026-09-27 · **Work order:** `CCMAI-DATABASE-001` · **Rủi ro:** R2.

## Phạm vi

Đợt này đổi tên database mặc định của bản cài mới từ `cqa` thành `CCMA` và chốt luồng dữ liệu MySQL → bộ lọc ứng dụng → AI/LLM. Không đổi database engine, không di chuyển dữ liệu của bản cài hiện có, không tích hợp extension `pg-jev` hoặc TypeSafe API, và không triển khai machine gate runtime trong work order này.

## Lưu trữ hiện tại

- MySQL 8 là database runtime; Go truy cập qua GORM MySQL driver.
- `channels` giữ cấu hình nguồn và trạng thái đồng bộ; credential được mã hóa.
- `conversations` giữ định danh hội thoại, kênh, khách hàng, mốc tin cuối và metadata.
- `messages` giữ nội dung, vai trò người gửi, thời gian, attachment và raw payload.
- `jobs`, `job_runs`, `job_results` giữ cấu hình phân tích, lượt chạy và output; `ai_usage_logs` giữ token/chi phí.
- File đính kèm nằm ở local/S3 theo cấu hình; database lưu metadata và storage key.

Tên schema không phải product boundary hoặc SoT authority. `CCMA` chỉ là mặc định mới cho cài đặt mới; `DB_NAME` vẫn là cấu hình có thể thay đổi.

## Hợp đồng tương thích tên database

1. `.env.example`, Compose fallback và backend fallback dùng `CCMA`.
2. Bản cài đã có `.env` với `DB_NAME=cqa` tiếp tục kết nối schema `cqa` và không đổi hành vi.
3. Đổi `DB_NAME` trên volume MySQL đã khởi tạo không tự tạo hoặc di chuyển schema. Muốn chuyển dữ liệu phải có backup/restore đã kiểm chứng và work order migration riêng.
4. Không tự dò rồi đổi sang schema khác: hành vi đó có thể che lỗi cấu hình hoặc mở nhầm dữ liệu.
5. Workspace phát triển mới dùng `DB_USER=ccma`; override vẫn được hỗ trợ cho môi trường triển khai có cấu hình riêng. Đường dẫn file kế thừa chưa đổi trong tranche này và độc lập với database identity.

## Luồng lọc đích

```text
channel adapters
  → normalize + persist raw evidence vào MySQL
  → SQL candidate selector (tenant, channel, time, changed snapshot, status)
  → snapshot builder (messages + role + full timestamp + coverage/provenance)
  → deterministic eligibility/policy checks
  → local rule pack / typed questions
  → WAIT_DATA | DENY | local result | LLM admission
  → chỉ LLM admission mới resolve provider/key và dispatch
  → validate evidence/output → proposal → review/action lifecycle
```

Database làm các phép lọc cấu trúc rẻ và nhất quán. Go application giữ policy, rule version, decision trace, privacy/budget admission và provider dispatch. Phân tích ngữ nghĩa tiếng Việt thuộc AI/LLM hoặc người theo policy; không nhét lời gọi mạng vào MySQL.

Điểm chèn runtime dự kiến là `Analyzer.runJobInternalExt`: sau khi tạo `JobRun`, đọc job/channel scope và chọn candidate/snapshot, trước `getProvider`. Hiện source gọi `getProvider` trước truy vấn candidate; tranche machine gate phải đảo dependency này cho mọi đường single/batch/manual/scheduled/re-run.

## Học từ pg-jev

Repo [Blackbird081/pg-jev](https://github.com/Blackbird081/pg-jev) là PostgreSQL extension gọi TypeSafe Jev trên từng row. Dự án học các nguyên tắc sau và triển khai ở tầng Go/MySQL:

- chạy SQL predicate rẻ trước semantic judgment;
- tạo projection/snapshot chỉ gồm trường cần cho câu hỏi;
- batch có trần, đo chất lượng theo kích thước thay vì chọn batch lớn nhất;
- cache theo content digest + question/rule/model version; đổi display threshold không buộc inference lại khi raw judgment còn hiệu lực;
- có giới hạn rows/chars, concurrency, timeout, cancel và cost/usage stats;
- viết câu hỏi cụ thể, quan sát phân bố và đánh giá threshold trên corpus đích;
- coi dữ liệu gửi provider là external disclosure cần policy và receipt.

Không dùng trực tiếp extension vì sản phẩm đang chạy MySQL, `pg-jev` cần PostgreSQL 14–17, `plpython3u`, superuser và TypeSafe API. Đưa semantic network call vào database cũng làm mờ admission, audit và provider-neutral boundary mà roadmap yêu cầu.

## Acceptance

- Mặc định database `CCMA` và application user `ccma` nhất quán ở config, Compose, `.env.example`, test và tài liệu biến môi trường.
- Workspace phát triển dùng database mới; tài liệu không ngụ ý clone hoặc migration dữ liệu CQA.
- Roadmap mô tả điểm lọc MySQL/Go/provider và bài học có chọn lọc từ `pg-jev`.
- `go test ./config`, Compose config với biến tối thiểu, catalog check, workspace doctor và diff check đạt.
- Không có provider call hoặc claim governance runtime từ tranche này.

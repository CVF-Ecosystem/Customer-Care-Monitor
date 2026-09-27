# S0 runtime baseline: CSKH intervention pilot

**Tranche:** `CCMAI-RUNTIME-001` · **Ngày:** 2026-09-27 · **Kết quả ban đầu:** BASELINE_RECORDED / NO_PROVIDER_CALL.

## Pilot boundary

- Adapter pilot: Pancake.
- Use case: phát hiện hội thoại có khiếu nại hoặc khách chờ phản hồi để tạo proposal cho người phụ trách; tranche hiện tại chưa tạo proposal hay gọi AI.
- Runtime contract là channel-neutral; Pancake chỉ cung cấp nguồn pilot.
- Corpus: synthetic-only `s0-vi-intervention-v1`; không chứa dữ liệu khách hàng.

## Database baseline

Đọc trực tiếp từ volume phát triển `ccma_mysql_data` trước BUILD:

| Surface | Rows |
|---|---:|
| tenants | 0 |
| channels | 0 |
| conversations | 0 |
| messages | 0 |
| jobs | 0 |
| job_runs | 0 |
| job_results | 0 |
| ai_usage_logs | 0 |

Schema `CCMA` có 16 bảng. Vì chưa có lượt chạy, chưa có baseline call, token, cost, latency, false positive/negative hoặc human-review time. Các số 0 không được dùng để claim tiết kiệm hay chất lượng.

## Error inventory locked for v1

Chat rỗng; thiếu lịch sử; duplicate/replay; khiếu nại rõ; phủ định; mỉa mai; tiếng Việt không dấu; xen ngôn ngữ; PII; prompt injection trong lời khách; claim hoàn tiền chưa có SoT giao dịch; source bị sửa sau đánh giá.

## Claim boundary

S0 chỉ khóa dữ liệu synthetic và baseline rỗng có thật. Không provider API nào được gọi và không có claim rằng CVF/machine gate đang điều phối AI runtime.

# Alibaba Cloud MaaS — Free Quota Provider Reference

**Ngày ghi nhận:** 2026-09-27
**Workspace:** 默认业务空间 (`ws-remplsp27g5oicq1`)
**Region:** ap-southeast-1
**Endpoint (OpenAI-compatible):** `https://ws-remplsp27g5oicq1.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`

**API Key:** lưu trong `.env` dưới key `ALIBABA_MAAS_API_KEY` — không được commit vào git.

## Mục đích trong CVF

Dùng làm provider thực cho các work order S2/S3+ yêu cầu `liveGovernanceEvidenceRequired: true`.
Không dùng để sync kênh thật hoặc xử lý dữ liệu khách hàng thật cho đến khi có work order được chấp thuận.
Free quota: đủ cho integration tests và live governance evidence nhỏ.

## Model catalog và free quota (ghi nhận 2026-09-27)

| Model | Quota còn lại | Hết hạn | Ghi chú |
|---|---|---|---|
| `qwen3.8-27b` | 999.93K / 1M | 2026-11-16 | Flagship 27B |
| `qwen3.7-flash-2026-07-15` | 1M / 1M | 2026-10-22 | Flash, hết hạn sớm nhất |
| `qwen3.8-flash` | 990.47K / 1M | 2026-11-24 | Flash mới nhất |
| `kimi-k3` | 1M / 1M | 2026-11-16 | Kimi (Moonshot) |
| `deepseek-v4-flash-0731` | 1M / 1M | 2026-10-30 | DeepSeek V4 flash |
| `qwen3.8-max-0902` | 994.63K / 1M | 2026-11-30 | Qwen Max |
| `deepseek-v4.1-flash` | 1M / 1M | 2026-12-12 | **Hết hạn muộn nhất** |
| `glm-5.3` | 1M / 1M | 2026-11-22 | GLM (Zhipu) |
| `deepseek-v4-pro-0813` | 999.91K / 1M | 2026-11-12 | DeepSeek V4 Pro |
| `qwen3.8-2.4t-a95b` | 1M / 1M | 2026-11-11 | MoE variant |

## Khuyến nghị lựa chọn model

- **Test nhanh / unit evidence:** `qwen3.8-flash` hoặc `deepseek-v4.1-flash` (1M còn nguyên, thời hạn dài)
- **Chất lượng phân tích cao:** `qwen3.8-27b` hoặc `deepseek-v4-pro-0813`
- **Ưu tiên hạn dùng lâu nhất:** `deepseek-v4.1-flash` (hết hạn 2026-12-12)

## Cảnh báo hết hạn

- `qwen3.7-flash-2026-07-15`: **hết hạn 2026-10-22** (chỉ còn ~25 ngày từ ngày ghi nhận)
- Kiểm tra lại bảng trước mỗi work order S2/S3 sử dụng provider này.

## Cách cấu hình provider trong backend

```go
// Ví dụ: OpenAI-compatible client
baseURL := os.Getenv("ALIBABA_MAAS_BASE_URL")
apiKey  := os.Getenv("ALIBABA_MAAS_API_KEY")
model   := "deepseek-v4.1-flash"  // hoặc model khác từ catalog
```

Header `Authorization: Bearer <ALIBABA_MAAS_API_KEY>` — cùng chuẩn OpenAI.

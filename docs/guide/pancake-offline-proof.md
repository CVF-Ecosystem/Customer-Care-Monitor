# Chạy thử công cụ kiểm chứng Pancake ngoại tuyến

Công cụ `pancake-proof` là một chương trình dòng lệnh nhỏ để **chạy thử trên máy của bạn, không cần mạng, không cần tài khoản Pancake**. Nó đọc một kịch bản hội thoại giả (cố định, nằm sẵn trong chương trình), so sánh với danh sách hội thoại **bạn kỳ vọng** (một file JSON giả do bạn tự viết) rồi in ra một biên nhận kết quả.

> **Chỉ là bản chạy thử.** Mọi kết quả ở đây đều ghi rõ `SYNTHETIC_OFFLINE`, `live: false` và `governance_claim: false`. Nó **không** chứng minh kênh Pancake thật hoạt động, không chứng minh dữ liệu thật đã được đồng bộ đầy đủ, không chứng minh cơ chế kiểm soát AI hoạt động. Muốn kiểm chứng kênh thật cần một đợt làm việc khác, có dữ liệu và quyền truy cập riêng ([lộ trình](/reviews/F02_PANCAKE_LIVE_PROOF_PACKET_2026-10-03)).

## Công cụ làm gì, không làm gì

- Chạy hoàn toàn trong bộ nhớ: không mở kết nối mạng, không đọc `.env`, cấu hình, khoá hay cơ sở dữ liệu, không tải ảnh, không ghi file.
- Không có chế độ "chạy thật", không có tuỳ chọn nhập token, địa chỉ máy chủ, đường dẫn cấu hình hay cơ sở dữ liệu. Thêm tuỳ chọn lạ vào dòng lệnh sẽ bị từ chối.
- Phần "kết quả quan sát" luôn là kịch bản giả cố định của chương trình. File JSON của bạn **chỉ thay phần kỳ vọng**, không bao giờ sinh ra kết quả quan sát.

## Chuẩn bị

Cần Go đã cài sẵn và các gói đã có trong bộ nhớ đệm của máy (không tải gì thêm, không cần trình biên dịch C, không cần quyền quản trị). Mở PowerShell **tại thư mục gốc của dự án** (thư mục có `backend` và `docs`). Đường dẫn có dấu cách thì luôn đặt trong dấu nháy kép.

```powershell
$env:GOPROXY = 'off'; $env:GOSUMDB = 'off'; $env:GOTOOLCHAIN = 'local'; $env:CGO_ENABLED = '0'

# Thư mục tạm riêng của bạn, trên ổ cục bộ, không phải đường dẫn mạng
$tmp = Join-Path ([System.IO.Path]::GetTempPath()) 'pancake-proof-demo'
New-Item -ItemType Directory -Force -Path $tmp | Out-Null

# Biên dịch chỉ riêng công cụ này ra thư mục tạm
go -C backend build -o "$tmp\pancake-proof.exe" ./cmd/pancake-proof
```

Luôn dùng `-o` như trên. Chạy `go build` mà không có `-o` sẽ để lại file `pancake-proof.exe` trong thư mục `backend` của dự án.

Sao chép file mẫu vào thư mục tạm (file mẫu có sẵn trong kho mã tại `docs/examples/pancake-proof/synthetic-inventory.json`, nội dung đầy đủ ở phần cuối trang):

```powershell
Copy-Item -LiteralPath 'docs\examples\pancake-proof\synthetic-inventory.json' -Destination "$tmp\inventory.json"
```

## Chạy

Hai thông tin bắt buộc cho mọi lần chạy:

- `-source-sha`: đúng 40 ký tự thập lục phân viết thường. Đây chỉ là **nhãn bạn tự khai** để ghi vào biên nhận. Công cụ không kiểm tra nó có khớp với mã nguồn đã biên dịch hay không; muốn ghi lại mã nguồn thực tế, hãy tự lấy `git rev-parse HEAD`.
- `-key`: một chuỗi bất kỳ không rỗng, dùng để băm (ẩn) mã hội thoại, mã tin nhắn trong biên nhận. Đây là **khoá minh hoạ**, đừng dùng khoá API, mật khẩu hay token thật.

```powershell
$sha = (git rev-parse HEAD)
$key = 'demo-pseudonym-key-0001'
```

**1. Kịch bản mặc định (không dùng file của bạn):**

```powershell
& "$tmp\pancake-proof.exe" -source-sha $sha -key $key
"exit $LASTEXITCODE"
```

**2. Dùng file kỳ vọng của bạn** (chỉ với kịch bản `pass`, mặc định hoặc ghi rõ `-scenario pass`):

```powershell
& "$tmp\pancake-proof.exe" -source-sha $sha -key $key -inventory "$tmp\inventory.json"
"exit $LASTEXITCODE"
```

Kết quả mong đợi với file mẫu: thoát với mã `0`, biên nhận có `"disposition": "PASS"`, `"evidence_type": "SYNTHETIC_OFFLINE"`, `"live": false`, `"governance_claim": false`, `"provenance": "synthetic-docs-v1"`, 2 hội thoại, 3 tin nhắn và `"attempts_used": 12`. Chạy lại lần hai cho cùng nội dung, trừ các trường thời gian (`started_at`, `finished_at`, `conversation_until`, `elapsed_ms`) và thời lượng thì hai biên nhận giống nhau.

Biên nhận in ra màn hình (stdout); dòng cảnh báo "SYNTHETIC OFFLINE ..." in ở kênh lỗi (stderr). Muốn lưu, hãy tự chuyển hướng stdout. Công cụ không tự ghi file nào.

Mỗi lần chạy gộp **hai lượt** gọi giả và dùng chung một ngân sách. Với kịch bản mẫu cần đúng 12 lần gọi; đặt `-max-attempts 11` sẽ cho kết quả `INCOMPLETE` vì hết ngân sách.

## Các tuỳ chọn được hỗ trợ

| Tuỳ chọn | Mặc định | Ghi chú |
|---|---|---|
| `-source-sha` | (bắt buộc) | 40 ký tự thập lục phân viết thường |
| `-key` | (bắt buộc) | chuỗi không rỗng, khoá minh hoạ |
| `-scenario` | `pass` | `pass`, `missing-conversation`, `redirect`, `empty-inventory` |
| `-max-attempts` | `20` | từ 1 đến 50, tính chung cho cả hai lượt |
| `-max-duration` | `1m` | lớn hơn 0 và tối đa `10m`, ví dụ `30s` |
| `-inventory` | (không dùng) | một file JSON cục bộ; chỉ dùng với `pass`; ghi hai lần bị từ chối |

Dùng `-inventory` cùng kịch bản khác `pass` bị từ chối **trước khi mở file** (thoát mã 2, không in biên nhận). Mọi tuỳ chọn khác, kể cả `-live`, `-token`, `-config`, `-base-url`, `-db`, đều bị từ chối.

## Ý nghĩa mã thoát

| Mã | Ý nghĩa |
|---|---|
| `0` | PASS ngoại tuyến: kết quả quan sát khớp với kỳ vọng |
| `1` | FAIL (không khớp) hoặc INCOMPLETE (chưa đủ điều kiện kết luận, ví dụ hết ngân sách hoặc kịch bản rỗng); biên nhận vẫn được in |
| `2` | Dùng sai tuỳ chọn, file đầu vào bị từ chối, hoặc biên nhận không qua bước kiểm tra che giấu; **không** in biên nhận, stdout rỗng |

Khi gọi nhiều lệnh liền nhau, hãy đọc `$LASTEXITCODE` ngay sau từng lệnh như ở trên, kẻo mã thoát `1` hay `2` bị lệnh sau che mất.

Thông báo lỗi luôn là câu cố định (ví dụ `pancake-proof: inventory rejected`). Công cụ không in lại đường dẫn file, nội dung file, mã hội thoại, tên file đính kèm hay khoá. Trường hợp hợp lệ cũng vậy: biên nhận chỉ chứa mã băm; nhãn `provenance` bạn khai báo được giữ nguyên nên đừng đặt thông tin nhạy cảm vào đó.

### Ví dụ kết quả không phải PASS

Kỳ vọng rỗng (`"conversations": []`) trong khi kịch bản giả vẫn có hội thoại: kết quả `FAIL`, lý do `reconciliation_mismatch`, thoát mã `1`, 4 lần gọi. Chạy kịch bản có sẵn `-scenario empty-inventory` (không dùng `-inventory`): kết quả `INCOMPLETE`, lý do `empty_inventory`, thoát mã `1`.

```powershell
& "$tmp\pancake-proof.exe" -source-sha $sha -key $key -scenario empty-inventory
"exit $LASTEXITCODE"
```

## Định dạng file kỳ vọng

File JSON UTF-8, đúng một đối tượng, giản đồ `pancake-proof-synthetic-inventory/1`. Mọi trường dưới đây **bắt buộc, xuất hiện đúng một lần**, không được thừa trường nào.

| Cấp | Trường | Quy tắc |
|---|---|---|
| gốc | `schema_version` | đúng `pancake-proof-synthetic-inventory/1` |
| gốc | `evidence_type` | đúng `SYNTHETIC_OFFLINE` |
| gốc | `provenance` | nhãn 1 đến 64 ký tự gồm chữ, số, `.` `_` `:` `-`, phải bắt đầu bằng `synthetic-` |
| gốc | `page_id` | đúng `synthetic-page-001` |
| gốc | `conversations` | mảng, có thể rỗng |
| hội thoại | `id` | bắt đầu bằng `syn-`, không quá 128 byte, không chứa `/` `\` `%` `?` `#` `;` `:` khoảng trắng hay ký tự điều khiển, không trùng nhau |
| hội thoại | `updated_at` | thời điểm RFC 3339 có múi giờ rõ ràng (`Z` hoặc `+07:00`), khác thời điểm không |
| hội thoại | `messages` | mảng |
| tin nhắn | `id` | như mã hội thoại, không trùng trong cùng hội thoại |
| tin nhắn | `sent_at` | như `updated_at` |
| tin nhắn | `sender_type` | `customer` hoặc `agent` |
| tin nhắn | `content_type` | `text` hoặc `attachment` |
| tin nhắn | `attachments` | mảng, có thể rỗng |
| đính kèm | `type` | đúng `image` |
| đính kèm | `name` | 1 đến 256 byte, không có ký tự điều khiển |

Các thời điểm cùng một khoảnh khắc nhưng viết khác múi giờ được coi là như nhau; thứ tự hội thoại và tin nhắn trong file không ảnh hưởng.

Bị từ chối (thoát mã 2): trường lạ hoặc thiếu ở bất kỳ cấp nào (tên trường phân biệt hoa thường), trường **lặp lại dù cùng giá trị**, giá trị `null` hoặc sai kiểu, văn bản thừa sau JSON, UTF-8 sai, mã không an toàn hoặc trùng, thời điểm thiếu múi giờ hoặc bằng không, và vượt giới hạn sau. File có chứa trường kiểu token hay dữ liệu thô cũng bị từ chối vì là trường lạ.

| Giới hạn | Giá trị |
|---|---|
| Kích thước file | tối đa 1 MiB (chương trình chỉ đọc tối đa 1 MiB + 1 byte và từ chối, không cắt) |
| Số hội thoại | 100 |
| Tổng số tin nhắn | 1000 |
| Số đính kèm mỗi tin nhắn | 16 |
| Độ lồng JSON | 64 cấp |

### Quy tắc về đường dẫn file

Chỉ nhận một file cục bộ cụ thể, đã có sẵn. Bị từ chối: đường dẫn rỗng, `-`, địa chỉ `http://` hay `file://`, đường dẫn mạng dạng `\\máy\thư-mục` hoặc `//máy/...` (kể cả viết lẫn `\` và `/`) và các dạng `\\?\`, `\\.\`, `\??\`, thư mục, thiết bị. Mọi thư mục cha và chính file không được là liên kết tượng trưng (symlink) hay điểm phân tích lại như junction của Windows; kể cả thư mục chuyển hướng nằm trong hồ sơ người dùng cũng bị từ chối. Nếu file hợp lệ mà vẫn bị từ chối, hãy chép sang một thư mục thường trên ổ cục bộ (ví dụ `C:\Temp\demo`).

## Những điều chưa được xác nhận

- **Symlink chưa được kiểm chứng trên máy phát triển.** Bài kiểm tra đã có hai ca symlink nhưng chúng bị bỏ qua (SKIP) vì máy thiếu quyền tạo symlink; việc từ chối thư mục junction được kiểm tra riêng. Việc từ chối symlink cần chạy lại trên máy có quyền tạo symlink.
- **Chưa chạy kiểm tra tranh chấp luồng (race).** Máy không có trình biên dịch C.
- Ổ đĩa mạng được ánh xạ thành chữ cái (ví dụ `Z:`) hoặc ổ `subst` không phát hiện được bằng cách nhìn đường dẫn. Hãy chỉ dùng thư mục cục bộ do bạn tự tạo.
- Có một khoảng hở nhỏ giữa lúc kiểm tra và lúc mở file (TOCTOU) chưa loại bỏ hoàn toàn. Đây không phải cơ chế cách ly đường dẫn tuyệt đối; chỉ dùng file của chính bạn.
- Chưa có chế độ nạp danh sách hội thoại thật, chưa có xác thực nguồn gốc dữ liệu thật, thông tin truy cập hay đường truyền thật. Nhãn `provenance` bạn khai không chứng minh dữ liệu là thật hay đã ổn định.

Hồ sơ nguồn: [đặc tả R041](../specs/PANCAKE_OFFLINE_INVENTORY_INPUT_R041_2026-10-03.md), [bản dựng và sửa lỗi R041](/reviews/PANCAKE_OFFLINE_INVENTORY_INPUT_R041_BUILD_2026-10-03), [đánh giá độc lập](/reviews/CCMAI_RUNTIME_041_INDEPENDENT_REVIEW_2026-10-03).

## File mẫu đầy đủ

Nội dung của `docs/examples/pancake-proof/synthetic-inventory.json`, hoàn toàn giả:

```json
{
  "schema_version": "pancake-proof-synthetic-inventory/1",
  "evidence_type": "SYNTHETIC_OFFLINE",
  "provenance": "synthetic-docs-v1",
  "page_id": "synthetic-page-001",
  "conversations": [
    {
      "id": "syn-conv-a",
      "updated_at": "2026-09-24T10:00:00Z",
      "messages": [
        {
          "id": "syn-msg-a1",
          "sent_at": "2026-09-24T09:00:00Z",
          "sender_type": "customer",
          "content_type": "text",
          "attachments": []
        },
        {
          "id": "syn-msg-a2",
          "sent_at": "2026-09-24T09:05:00Z",
          "sender_type": "agent",
          "content_type": "attachment",
          "attachments": [
            { "type": "image", "name": "photo-001.jpg" }
          ]
        }
      ]
    },
    {
      "id": "syn-conv-b",
      "updated_at": "2026-09-20T00:00:00Z",
      "messages": [
        {
          "id": "syn-msg-b1",
          "sent_at": "2026-09-20T00:00:00Z",
          "sender_type": "customer",
          "content_type": "text",
          "attachments": []
        }
      ]
    }
  ]
}
```

Kịch bản giả cố định ứng với file này gồm hai hội thoại hợp lệ (một hội thoại cũ hơn mốc thời gian bị loại sẵn) và ba tin nhắn. Muốn thử trường hợp lệch, hãy sửa một giá trị (ví dụ đổi `sender_type`) trong bản sao ở thư mục tạm rồi chạy lại: kết quả sẽ là `FAIL`, thoát mã `1`.

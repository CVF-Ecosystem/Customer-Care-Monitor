# Hướng dẫn tiếp tục dự án trên máy khác (CVF portable continuation)

Project: Customer-Care-Monitor-AI
CVF Core (public, read-only): `https://github.com/Blackbird081/Controlled-Vibe-Framework-CVF.git` ghim tại `e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba`
Gate profile: `cvf.downstreamGateProfile@1.1.0` · Continuity contract: `cvf.downstreamContinuityContract@1.0.0`

## 1. Phạm vi và những gì tài liệu này KHÔNG chứng minh

Mục tiêu: một người hoặc agent clone dự án vào một thư mục sạch bất kỳ, khởi tạo liên kết tới CVF Core công khai đúng commit đã ghim, chạy các kiểm tra tĩnh của dự án, rồi tiếp tục từ trạng thái hiện tại do repo ghi lại.

Giới hạn trung thực:

- Các kiểm tra ở đây là kiểm tra tĩnh trên repository (cấu trúc, ghim, nhất quán trạng thái). Chúng không chứng minh CVF điều khiển hành vi AI lúc ứng dụng chạy; việc đó cần bằng chứng gọi provider thật và chưa được thực hiện trong gói này.
- Chưa có kết quả GitHub Actions nào được xác minh cho gói này. File workflow chỉ là cấu hình, không phải bằng chứng CI.
- Khả năng chạy trên máy khác chỉ được chứng minh trong phạm vi bằng chứng của review round1 (clone vào đường dẫn khác trên Windows, qua initializer và doctor). Phần PowerShell (initializer, doctor, catalog) chưa được kiểm chứng trên Linux/macOS; không suy ra khả chuyển tuyệt đối.
- Tồn đọng của sản phẩm vẫn mở và không bị gói này xử lý: 55 liên kết chết trong tài liệu kế thừa (docs55) chặn FREEZE sản phẩm; R086 mới chỉ chấp nhận ở mức source, native readiness NOT_RUN và tích hợp durable fence/recovery còn OPEN; tài khoản Facebook/Zalo OA đang PARKED; trạng thái triển khai là UNKNOWN.

## 2. Yêu cầu môi trường

- Git và truy cập mạng tới hai repo công khai: dự án và CVF Core. Repo có đường dẫn rất dài (evidence trong `docs/reviews/probes/`): trên Windows hãy bật `core.longpaths` khi clone (lệnh dưới đã làm) và chọn thư mục container ngắn (ví dụ `D:\CCM`); đường dẫn gốc quá dài khiến checkout lỗi `Filename too long`.
- Python 3.10 trở lên (đã dùng 3.11 khi kiểm chứng; CI cấu hình 3.12). Luôn chạy với `-B` hoặc `PYTHONDONTWRITEBYTECODE=1` để không sinh `__pycache__` (gate cảnh báo đường dẫn chưa theo dõi).
- PyYAML 6.0.2, bắt buộc cho gate `workflows` của `scripts/cvf_downstream_gate.py`: `python -m pip install PyYAML==6.0.2`.
- Windows PowerShell 5.1 hoặc PowerShell 7 cho `scripts/initialize_cvf_clone.ps1`, doctor và catalog manager. Đây là phần mới được kiểm chứng trên Windows.
- Không cần provider API key, Docker, SQL, Go hay dữ liệu khách hàng cho các bước dưới đây.

## 3. Clone và khởi tạo

Dự án dùng bố cục "container": Core được clone thành thư mục anh em `../.Controlled-Vibe-Framework-CVF` và `../WORKSPACE_RULES.md` được tạo cạnh dự án. Vì vậy hãy clone vào một thư mục container trống, không clone thẳng vào thư mục chứa nhiều dự án khác.

```powershell
mkdir D:\CCM                      # thư mục trống ngắn bất kỳ; không cần trùng đường dẫn máy gốc
cd D:\CCM
git clone -c core.longpaths=true --branch codex/cvf-portable-inheritance https://github.com/CVF-Ecosystem/Customer-Care-Monitor.git Customer-Care-Monitor
cd Customer-Care-Monitor
python -m pip install PyYAML==6.0.2
powershell -ExecutionPolicy Bypass -File scripts/initialize_cvf_clone.ps1
```

Ghi chú:

- Repo: `https://github.com/CVF-Ecosystem/Customer-Care-Monitor.git`. Nhánh giao hàng của gói GOV-002 là `codex/cvf-portable-inheritance`. Nhánh này chỉ mang gói đã được kiểm tra sau khi Local review và xuất bản; trước đó nó chỉ có commit chuẩn bị. Sau khi gói được merge vào `main`, hãy clone `main` (bỏ `--branch`).
- Initializer đọc `.cvf/manifest.json`, clone Core công khai nếu chưa có, từ chối Core bẩn hoặc không khớp lịch sử công khai (không reset, không hạ cấp), ghim Core đúng commit, ghi liên kết cục bộ vào `.cvf/local-binding.json` (bị gitignore) và chạy doctor. Kết quả mong đợi cuối cùng là dòng `FRESH_CLONE_CONTINUITY_PASS`.
- Nếu Core trong container đã tồn tại nhưng bẩn hoặc không liên quan, script dừng và báo lỗi: hãy xử lý bằng `scripts/update_cvf_workspace_public_core.ps1` của Core thay vì xóa thủ công.

## 4. Xác minh sau khi khởi tạo

Chạy từ thư mục gốc dự án:

```powershell
# Doctor độc lập của Core (đối chiếu từng byte profile đã cài với nguồn Core công khai)
powershell -ExecutionPolicy Bypass -File ../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .
# Bảy gate legacy của dự án
python -B scripts/cvf_downstream_gate.py preflight
# Tự kiểm thử của gate (legacy + tương thích profile)
python -B -m unittest discover -s scripts/tests -p "test_cvf_downstream_gate*.py"
# Các phase của profile đã cài (thêm --packet khi có work order/review đang xét, xem AGENTS.md)
python -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase pre-dispatch --project-root .
```

Kỳ vọng: doctor `RESULT: PASS`, preflight `PASS (7/7 executed gates passed)`, unittest `OK`, runner `RESULT: PASS`. Mỗi control của runner in ra dòng riêng; outcome `NOT_APPLICABLE_WITH_REASON` nghĩa là control đó KHÔNG được kiểm tra trong lần chạy này, không phải là đã đạt. Không dùng các kết quả này để tuyên bố trạng thái nằm ngoài những gì chúng kiểm tra.

## 5. Tiếp tục công việc (đọc theo từng bước, không đọc toàn bộ lịch sử)

1. `AGENTS.md`: giao thức khởi động, vai trò, và bảng lệnh phase của profile đã cài.
2. `CVF_SESSION/ACTIVE_SESSION_BOOTSTRAP_READ_MODEL.json` (mô hình khởi động nhanh): chế độ, phase, handoff đang hoạt động, bước đi kế tiếp được phép, checkpoint bị park.
3. `CVF_SESSION_MEMORY.md`: phần "Current Truth" và chỉ thị roadmap hiện hành của owner.
4. Handoff đang hoạt động do mô hình khởi động ở bước 2 chỉ ra (trường `activeHandoff`; đừng giả định một tên file cố định, vì tên thay đổi theo tranche), cùng [IMPLEMENTATION_STATUS.json](../../IMPLEMENTATION_STATUS.json) khi cần sự thật về năng lực.
5. Chỉ khi một sự thật hiện tại bị thiếu hoặc mâu thuẫn mới tra cứu có mục tiêu `CVF_SESSION/ACTIVE_SESSION_STATE.json` đầy đủ hoặc handoff lịch sử. Nếu bốn bề mặt (state, memory, handoff, implementation status) mâu thuẫn nhau, dừng và báo `BLOCKED_CONTINUITY_DRIFT`.
6. Trước khi chọn phạm vi kế tiếp, đọc phần checkpoint S0–S7 và P0–P4 của `docs/roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md` và bài học [kiểm soát roadmap của orchestrator](../reviews/learnings/feedback_orchestrator_roadmap_control.md). Kho bài học chung nằm ở [docs/reviews/learnings](../reviews/learnings/README.md); bản mẫu record mới là [LEARNING_RECORD_TEMPLATE.md](../reviews/learnings/LEARNING_RECORD_TEMPLATE.md).

Trước hành động vật chất đầu tiên, agent phát `CVF Agent Declaration` theo AGENTS.md từ giá trị file hiện tại, không dựa vào lịch sử chat hay bộ nhớ riêng của provider (bộ nhớ riêng chỉ là nguồn bổ sung, không phải authority).

## 6. Quy tắc cần giữ khi làm việc

- Worker không commit, không push, không merge; Commit Steward / Reviewer độc lập thực hiện. Với rủi ro R2 trở lên, reviewer phải độc lập với worker.
- Không commit `.cvf/local-binding.json`, `.cvf/runtime/`, `__pycache__`, file `.env`, khóa hoặc token. Gate secrets quét file thay đổi nhưng có thể có âm tính giả.
- Không sửa tay module profile đã cài trong `scripts/cvf_gates/`; chúng được ghim bằng `.cvf/gate-profile.lock.json` và chỉ được cài lại từ Core công khai.
- Core công khai là chỉ-đọc: không chỉnh sửa, không commit artifact dự án vào đó. Kho CVF cha riêng tư không thuộc gói này và không được xuất.
- Tài khoản Facebook / Zalo OA tiếp tục PARKED cho tới khi owner mở lại; không lưu credential trong repo.

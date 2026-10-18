# Hợp đồng kích hoạt can thiệp trên dữ liệu minh họa

Status: SPEC_ONLY_ACTIVATION_NOT_IMPLEMENTED

N02 được local xác nhận là giới hạn triển khai có chủ đích. Bản sửa hiện tại công khai trạng thái cấu hình qua API có kiểm tra quyền đọc và thông báo dễ hiểu trên màn hình vụ việc. Nó không tự bật runner, cấp ngân sách/grant, đọc khóa hay chuyển dữ liệu thật sang luồng synthetic.

## Phạm vi khả năng hiện hành

`GET /api/v1/tenants/:tenantId/intervention-capability` trả scope `synthetic_only`, `generation_configured` và `rerun_configured`. Đây là trạng thái transport/policy trong process, không phải quyền người dùng, budget còn lại hoặc provider health. Không có endpoint/key/model/config secret trong response. Ordinary startup hiện trả false/false. Các vụ việc có sẵn vẫn được xem và thao tác theo quyền.

## Contract tối thiểu trước khi kích hoạt

1. Chế độ mặc định disabled; chế độ opt-in duy nhất ban đầu là synthetic. Cấu hình server-owned chỉ rõ provider/model/HTTPS endpoint được duyệt, nguồn khóa ngoài repository, phiên bản prompt/policy, timeout và trần request/token. Thiếu/sai cấu hình phải fail closed với lỗi an toàn, không fallback sang provider/model khác.
2. Một loader được dùng thống nhất khi ordinary startup cài cả `InterventionRunner` và `WaitDataExecutor`; request/browser/grant không được cung cấp transport, endpoint hay secret. Việc cài transport không tự cho phép dữ liệu hoặc tiêu tiền.
3. Guard `ccma.synthetic.intervention.v1`, kênh Pancake có `IsDemoFixture`, tenant/job/input source hợp lệ giữ nguyên. Không coi metadata do người dùng gửi là marker do server cấp. Real-source activation cần contract riêng.
4. Cần đường quản trị được hỗ trợ để chuẩn bị fixture/job và cấp call budget/grant hữu hạn trước khi gọi. Không yêu cầu sửa DB bằng tay hoặc dùng test binary; không tự reset consumed/unknown attempt hoặc tự nới hạn mức khi restart. Grant gắn actor/request/revision/config và có deadline, idempotency, current permission/source checks.
5. UI phân biệt disabled, cấu hình một phần, chỉ dành cho minh họa và chưa kiểm tra được; không gọi một transport đã cài là sẵn sàng cho production. Tác vụ không được tự gửi khách hàng hoặc tự xếp loại nhân viên.
6. Nghiệm thu clean install: disabled không gọi AI; cấu hình thiếu/sai bị từ chối; cấu hình đúng + fixture/quyền/budget hợp lệ đi qua tạo case → review → assign → resolve/reopen và WAIT_DATA revision/rerun. Tenant/quyền sai, source đổi, version stale, hết budget hoặc completion unknown không thành công giả. Cần log API/DB/UI cùng SHA và shutdown/cleanup.
7. Test transport stub chứng minh software flow, không chứng minh AI/governance. Provider thật chỉ được chạy với phạm vi và chi phí được cấp riêng; không tái dùng evidence/cap của R100/R117/R121.

## Điều kiện chưa đạt

Ordinary startup activation, quản trị budget/grant và clean-install flow vẫn chưa triển khai/nghiệm thu. Vì vậy N02 chỉ được đánh dấu contract + capability visibility đã có, không được đánh dấu toàn bộ activation FIXED hoặc full M3/pilot/release PASS.

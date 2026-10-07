# Review publication và tính toàn vẹn receipt

Ngày 2026-10-08; project Customer-Care-Monitor-AI, R063. Maintainer: REVIEWER / SESSION_SYNC_STEWARD. Đọc trước khi xuất bản metadata cùng raw evidence. Đây là finding root tự phát hiện và đối soát, không phải owner-reported.

Trong publication sau campaign tại archive `661113d`, root dùng lại biến path registry làm biến vòng lặp proof files. Lệnh ghi registry sau vòng lặp ghi đè summary mới. Kiểm tra manifest cuối cùng phát hiện JSON sai shape trước commit. Summary gốc và snapshot inspect mất; source, seed, worker packet, raw JSONL và manifest còn nguyên. Xem [incident](../probes/r063_review_publication_incident.json) và [review độc lập](../R063_INDEPENDENT_RULE_OBSERVATION_REVIEW_2026-10-08.md).

Áp dụng ngay: dùng tên đường dẫn riêng theo mục đích; kiểm tra đích ghi và typed shape sau mỗi lần ghi; tách raw packet khỏi metadata sinh tự động, giữ bản gốc trước publication. Không dùng biến vòng lặp làm write target sau vòng lặp. Reviewer kiểm tra toàn bộ hash/reference và scope trước commit, giữ cả failed checks.

Khi mất receipt, dừng publication; không tạo lại snapshot giả hoặc chạy thêm runtime vượt budget. Chỉ đối soát nguồn thực tế còn lại và ghi rõ provenance/giới hạn. R063 dùng JSONL, full/mutant manifests, plan, actual Docker daemon events, restored sandbox, second archive và fresh resource absence; receipt thay thế mang nhãn DERIVED_RECONCILIATION_FROM_ACTUAL_RETAINED_EVIDENCE. Đối soát đạt với zero Go mới; bản gốc vẫn UNRECOVERABLE, không gọi là khôi phục nguyên trạng.

Project disposition: đã áp dụng bằng explicit registry path, kiểm tra receipt shape/hashes và script `docs/reviews/probes/r063_reconcile_review_evidence.py`; formal review ghi giới hạn inspect đã mất. Tài liệu không cấp acceptance hoặc authority mới. Upstream disposition: DEFERRED cho CVF parent intake; đề xuất xem xét write-target/receipt-integrity control, chưa gửi hay sửa CVF core/tooling. Không có claim runtime CVF điều khiển agent.

# Proposal

## Why

Hiện saveme chỉ khóa theo lịch (daily) hoặc theo thời điểm (timed). Người dùng không thể nói "mỗi ngày tôi chỉ được dùng facebook.com 30 phút". Cần một giới hạn thời gian sử dụng thực tế trong ngày, tính riêng cho từng domain, và khi hết giới hạn thì domain bị khóa không sửa được đến hết ngày.

## What Changes

- Mỗi domain có thể có thêm một **quota/ngày**: tổng thời gian được dùng, đo bằng số giây domain đó có kết nối TCP đang mở (`ESTABLISHED`). Mở tab thì đếm, đóng tab thì dừng.
- Quota **tồn tại song song** với lock lịch/timed trên cùng một domain. Domain bị chặn khi lock đang active HOẶC quota đã hết. Quota chỉ đếm khi domain đang mở.
- Khi hết quota, domain bị chặn đến 00:00 theo zone của daemon, đồng thời mọi `edit_domain`/`delete_domain`/lệnh quota nhận `ERR_LOCKED` (tái dùng cơ chế `targetActive`).
- Đặt quota cần mã xác nhận như lock; còn quota thì xóa quota và xóa domain được tự do, không cần mã.
- Mỗi domain được **đếm độc lập**: gán kết nối cho entry bằng SNI trong TLS ClientHello, không dùng IP. `facebook.com`, `instagram.com`, `messenger.com` không trừ quota của nhau khi dùng chung IP. Subdomain chỉ tính khi được liệt kê riêng, giữ nguyên quy tắc `facebook.com`/`www.facebook.com` hiện có.
- Status trả thêm số liệu quota (giới hạn, đã dùng, còn lại, thời điểm reset) cho mọi entry có quota, kể cả khi đang bị chặn. TUI hiển thị phần quota còn lại.
- State nâng lên version 3 (`Entry.Quota`), có migration từ v2.
- Chỉ hỗ trợ Linux. Trên Windows, lệnh quota bị từ chối với lỗi rõ ràng.
- **BREAKING**: binary cũ không đọc được state v3 (giống lần nâng v1 → v2). Bản cập nhật tự động vẫn phải giữ lock và quota đã lưu.

## Capabilities

### New Capabilities
- `domain-daily-quota`: giới hạn thời gian dùng mỗi ngày cho từng domain, cách kết hợp với lock lịch/timed, hành vi khi hết quota, reset ngày, trạng thái hiển thị và quy tắc sửa/xóa.
- `domain-usage-metering`: đo thời gian dùng thật của từng domain bằng kết nối đang mở được gán theo SNI, kèm hành vi an toàn khi bộ đo lỗi hoặc không đọc được SNI.

### Modified Capabilities
<!-- Dự án chưa có spec chính nào trong openspec/specs, nên không có capability hiện hữu để sửa. -->

## Impact

- Code: `internal/core` (`Entry`, `State` v3, `EntryStatus`, `Request`/`Command`), `internal/store` (đọc v2, ghi v3), `internal/daemon` (`engine.go`, `pending.go`, `mutate.go`, vòng tick), `internal/enforce` (nhánh Linux cho đo kết nối và cắt kết nối), `internal/dns` (cấp thông tin đã resolve nếu cần), `internal/ui` (màn chính, countdown, nhập quota), `cmd/saved`.
- Update guard: `saved -check-update` và `deploy/update.sh` hiện chỉ xét lock; phải xét cả quota đã lưu và sao lưu/khôi phục state v3.
- Docs: `docs/protocol.md`, `docs/setup.md` (mục Limits), `README.md`.
- Hệ thống: thêm thành phần đọc gói để lấy SNI (quyền root, chỉ Linux). Rủi ro cần kiểm chứng bằng spike: ECH làm SNI không đọc được, QUIC (UDP 443) không có `ESTABLISHED`, trình duyệt có thể giữ socket sau khi đóng tab.

# Tasks

## 1. Spike kiểm chứng giả định (làm trước, quyết định có tiếp tục hay không)

- [x] 1.1 Dùng `ss -tn state established` trong lúc mở rồi đóng tab Messenger, Facebook, Instagram trên trình duyệt thật; ghi lại sau bao lâu socket biến mất. Kết quả ghi vào `.docs/handoff.md`, đủ để chốt dung sai ở design (Open Questions).
- [x] 1.2 Kiểm tra ba domain có dùng ECH hay HTTP/3 không (bản ghi DNS HTTPS và lưu lượng UDP 443); kết quả ghi vào `.docs/handoff.md`. Nếu SNI không đọc được với domain đích, dừng và báo người dùng trước khi sang nhóm 5.
- [x] 1.3 Xác nhận Messenger web chạy dưới `facebook.com` hay `messenger.com` bằng SNI quan sát được; kết quả ghi vào `.docs/handoff.md`.

## 2. Mô hình dữ liệu và store

- [x] 2.1 Thêm `Quota{LimitSec, UsedSec, Day}` vào `core.Entry`, nâng `StateVersion` lên 3, cập nhật `Clone` và `Validate` (giới hạn 1m–1440m, `UsedSec >= 0`); chạy `go test ./internal/core/...` và có test cho giá trị biên.
- [x] 2.2 Cập nhật `internal/store` để đọc v2 (và v1 qua chuỗi hiện có) rồi luôn ghi v3, theo mẫu `legacy.go`; chạy `go test ./internal/store/...` có test v2→v3 giữ nguyên lock và checksum.
- [x] 2.3 Thêm trường quota vào `core.EntryStatus` (`limit_sec`, `used_sec`, `remaining_sec`, `reset_at`) và các `Command` mới `set_quota`, `clear_quota`; chạy `go test ./internal/core/...`.

## 3. Daemon: quy tắc active và lệnh quota

- [x] 3.1 Mở rộng `Engine.entryActive` thành `lockActive || quotaExhausted`, và `entryStatus` hợp nhất `end_at`/`next_at` với thời điểm reset 00:00 theo zone; chạy `go test ./internal/daemon/...` với các kịch bản lock hết nhưng quota còn, quota hết nhưng lock hết, đồng hồ chưa tin cậy.
- [x] 3.2 Cài `set_quota` qua `pending` (chuẩn bị + mã xác nhận như lock) và `clear_quota` qua `mutate`, tái dùng `core.Duration`; kiểm `targetActive` trả `ERR_LOCKED` khi hết quota; chạy `go test ./internal/daemon/...` cho đặt, sửa, xóa, hết quota, giá trị ngoài khoảng.
- [x] 3.3 Cộng thời gian dùng trong `Tick`/`refresh`, reset khi sang ngày chỉ lúc đồng hồ tin cậy, lưu trước khi áp dụng chặn và lưu ngay khi đạt quota; chạy `go test ./internal/daemon/...` với đồng hồ giả qua nửa đêm và restart giữa ngày.
- [x] 3.4 Từ chối `set_quota`/`clear_quota` bằng `ERR_INVALID` kèm thông báo nền tảng không hỗ trợ trên Windows; chạy `GOOS=windows go build ./...` và test cho nhánh từ chối.

## 4. Docs giao thức

- [x] 4.1 Cập nhật `docs/protocol.md` (lệnh quota, trường status, state v3, lỗi) và mục Limits trong `docs/setup.md` (ECH, QUIC, Linux-only); chạy `python3 .docs/check.py` nếu có.

## 5. Bộ đo kết nối (Linux)

- [x] 5.1 Viết parser TLS ClientHello lấy `server_name` (giới hạn kích thước, từ chối dữ liệu hỏng) trong gói riêng; chạy `go test` với ClientHello thật, bị cắt ngắn và hỏng.
- [x] 5.2 Cài bảng gán 5-tuple → entry theo SNI, dùng `core.DomainBlocked` để khớp tên; chạy `go test` cho hai domain chung IP không cộng chéo, subdomain không liệt kê không được tính.
- [x] 5.3 Đọc kết nối `ESTABLISHED` mỗi giây qua netlink sock_diag và trả danh sách entry đang dùng; chạy `go test` với nguồn kết nối giả, và kiểm thủ công bằng `ss` so khớp.
- [x] 5.4 Hủy socket đã gán và drop 5-tuple khi entry hết quota; chặn UDP 443 tới domain có quota; chạy test nftables trong môi trường cô lập (không cần quyền root trên máy phát triển, đưa lệnh cho người dùng chạy ở VM có root).
- [x] 5.5 Fail-closed: bộ đo lỗi hoặc không khởi động giữ nguyên chặn, status báo `ERR_ENFORCEMENT`; chạy `go test ./internal/enforce/...` với bộ đo giả bị lỗi.
- [x] 5.6 Nối bộ đo vào `cmd/saved/run_linux.go` và `Engine.Tick`; chạy `make test` và `make check`.

## 6. Cập nhật phần mềm

- [x] 6.1 Mở rộng `cmd/saved/update.go`, `preserved.go`, `verify_linux.go` để coi entry có quota là cam kết cần giữ (quota và `UsedSec` không giảm trong cùng ngày) và để rollback khôi phục state v3; chạy `go test ./cmd/saved/...` cho cập nhật thành công, rollback, và quota bị mất sau restart.
- [x] 6.2 Cập nhật `deploy/update.sh` để sao lưu và khôi phục state v3; chạy `make check` và kiểm bằng kịch bản trong `deploy/` (chỉ trong môi trường cô lập, không cài lên host).

## 7. TUI

- [x] 7.1 Hiển thị quota còn lại cạnh domain ở màn chính (`entries.go`) và phân biệt "hết quota, mở lại lúc 00:00 (còn Xh Ym)" trong `lockedView`; chạy `go test ./internal/ui/...` với kịch bản có quota, hết quota, không quota.
- [x] 7.2 Thêm màn đặt và xóa quota tái dùng bộ chọn thời lượng, chỉ cho phép khi domain không active, kèm giải thích khi bị vô hiệu như `disabled.go`; chạy `go test ./internal/ui/...` và chạy `make demo` để xem thực tế.
- [x] 7.3 Cập nhật `README.md` (Features) và `docs/setup.md` (mục Use) mô tả quota; kiểm các lệnh trong tài liệu chạy đúng như viết.

## 8. Tích hợp

- [x] 8.1 Chạy `make test`, `make check`, `make demo` bằng tài khoản thường; ghi lệnh, kết quả và giới hạn vào `.docs/handoff.md` và `.docs/trace.md`.
- [x] 8.2 Đề xuất người dùng chạy `make update` trên VM hoặc máy có quyền root để kiểm thực tế mở/đóng tab Messenger, Facebook, Instagram; không chạy `sudo` trên máy này.

## Workflow follow-up

- Xem lại change cùng người dùng, rồi archive khi các yêu cầu review của dự án được đáp ứng.
- Kiểm tra kết quả sau khi archive.

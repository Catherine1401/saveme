# Design

## Context

Hiện trạng đã đọc từ code:

- `core.Entry` chỉ có `Domain` và `Lock*`. `Engine.entryActive` trả `Lock != nil && (!Trusted || Lock.Active(now))`; mọi cơ chế "không sửa được" (`targetActive`, `ERR_LOCKED`) và việc chặn (`blocked` → `Enforcer.Apply` → `Filter.Set` + nftables) đều dựa vào hàm này.
- Chặn domain là DNS trả lời rỗng (`dns.Resolver`); nftables chỉ redirect DNS và drop DoT/DoH. Hiện không có rule nào khớp IP của domain, không có counter hay conntrack.
- `Engine.Tick` chạy mỗi giây từ `cmd/saved/run.go`; `refresh` lưu state mỗi 60 giây nếu cần.
- Store lưu hai generation kèm checksum; v1 được chuyển sang v2 trong `internal/store/legacy.go`.
- Luồng cập nhật (`cmd/saved/update.go`, `preserved.go`, `verify_linux.go`) so sánh lock trước và sau khi khởi động lại.

Xem proposal.md - Why cho động cơ. Các quyết định đã chốt khi thảo luận: chỉ Linux; đếm bằng kết nối `ESTABLISHED`; gán theo SNI để độc lập; quota song song với lock; còn quota thì sửa/xóa tự do.

## Goals / Non-Goals

**Goals:**
- Quota ngày theo từng domain, đếm độc lập, bền qua restart và update.
- Tái dùng `entryActive` làm điểm duy nhất quyết định "active", để chặn và `ERR_LOCKED` không cần sửa logic riêng.
- Fail-closed: lỗi bộ đo không bao giờ nới chặn.

**Non-Goals:**
- Windows (không có bộ đo; lệnh quota bị từ chối).
- Đo theo từng tab hay từng tiến trình; đếm theo lưu lượng byte.
- Khôi phục độ chính xác khi trình duyệt dùng ECH (chỉ ghi nhận giới hạn).
- Siết chặt việc sửa quota (đã chốt là tự do khi còn quota).

## Decisions

### 1. Mô hình dữ liệu: `Entry.Quota` tách khỏi `Lock`
`Entry` thêm `Quota *Quota{LimitSec, UsedSec, Day}`. `Day` là ngày (theo zone) mà `UsedSec` thuộc về, dùng để reset khi sang ngày mới.
- Vì sao: `Lock` là một con trỏ duy nhất gắn với `ERR_HAS_LOCK`; nhét quota vào đó phá quy tắc "tối đa một lock" và làm lẫn kiểu `Daily`/`Range`.
- Phương án đã loại: thêm kind thứ ba `Quota` vào `Lock` (không song song được với lock lịch).

### 2. Quy tắc active hợp nhất
`entryActive = lockActive || quotaExhausted`, với `quotaExhausted = Quota != nil && UsedSec >= LimitSec` (khi đồng hồ chưa tin cậy thì giữ trạng thái đã lưu, không reset). `Transition`/`end_at`/`next_at` lấy mốc muộn hơn giữa hết lock và reset ngày (00:00 theo zone) khi quota đã hết.
- Vì sao: giữ một nguồn sự thật, các đường `targetActive`, `blocked`, `status` tự đúng.

### 3. State v3 và migration
`StateVersion = 3`, `Entry.Quota` có `omitempty`. `store` đọc v2 (và v1 qua chuỗi migrate hiện có) rồi luôn ghi v3, theo mẫu `legacy.go`. Binary cũ không đọc được v3 (ghi vào BREAKING trong proposal).
- Phương án đã loại: giữ v2 và thêm trường tùy chọn. Binary cũ sẽ lặng lẽ bỏ qua quota và mở khóa domain hết quota, vi phạm commitment.

### 4. Lệnh quota
`set_quota` đi qua `pending` như `create_daily`/`create_range`: chuẩn bị trả mã và bản xem trước (`Entry.Quota`, không có `Lock`), `confirm` mới lưu. `clear_quota` theo mẫu `delete_domain`: áp dụng trực tiếp qua `mutate`. `Duration`/`minuteDuration` hiện có (1m–1440m) được tái dùng để phân tích giá trị. Kiểm tra `targetActive` chạy trước nên domain hết quota trả `ERR_LOCKED`.
- Vì sao `set_quota` cần mã: người dùng yêu cầu đặt quota có cùng độ ma sát với đặt lock (đã đổi so với quyết định ban đầu là không cần mã).
- Vì sao `clear_quota` vẫn trực tiếp: còn quota thì xóa tự do là quyết định đã chốt; hết quota thì `ERR_LOCKED`.

### 5. Bộ đo kết nối (Linux)
Thành phần mới trong `internal/enforce`, chạy trong daemon, với hai việc nhỏ tách riêng:
1. **Gán kết nối theo SNI:** socket `AF_PACKET` (SOCK_DGRAM, BPF chỉ giữ TCP tới cổng 443 có payload) đọc thụ động các byte đầu của mỗi kết nối đi ra, gom ClientHello trải trên nhiều segment, parser đọc `server_name`, tra entry theo quy tắc khớp tên hiện có (`core.DomainBlocked`), rồi lưu kết nối (5-tuple) → entry.
   - Vì sao không dùng nfqueue như dự kiến ban đầu: chưa có thư viện nfqueue trong dự án; nfqueue cần verdict cho từng gói (daemon chậm hoặc chết làm nghẽn mạng), còn bắt thụ động không chen vào đường đi của gói. ClientHello của Chrome (hơn 1,4 KB) có thể nằm trên hai segment nên bộ đo gom luồng thay vì đọc một gói.
2. **Đọc trạng thái kết nối:** mỗi giây hỏi kernel (netlink sock_diag, như `ss`) lấy các kết nối `ESTABLISHED` rồi đối chiếu với bảng gán ở bước 1. Entry có ít nhất một kết nối thì cộng 1 giây vào `UsedSec`.
- Vì sao `ESTABLISHED` mà không phải byte counter: tab Messenger im lặng chỉ có keepalive nhỏ, counter dao động; kết nối dài phản ánh đúng "tab đang mở".
- Vì sao không theo IP: IP CDN dùng chung nên không độc lập được (yêu cầu của người dùng).
- Máy kiểm tra: `ss -tn state established` chạy được không cần root; `/proc/net/nf_conntrack` không có, nên không dựa vào conntrack. `nf_conntrack_tcp_timeout_established` là 432000 giây nên không dùng timeout conntrack để suy ra kết nối đã chết.

### 6. Cắt kết nối khi hết quota
Khi một entry chuyển sang active vì hết quota, ngoài việc `Filter.Set` (DNS rỗng cho truy vấn mới), bộ đo phải hủy các socket đã gán cho entry đó (sock_diag `SOCK_DESTROY`) và thêm rule drop theo 5-tuple, để tab đang mở không còn nhận dữ liệu. Kết nối mới của domain bị DNS chặn; kết nối qua IP đã cache bị cắt vì SNI vẫn khớp.

### 7. QUIC và ECH
- Chặn UDP 443 tới domain có quota để buộc trình duyệt dùng TCP. Cần có IP của domain, lấy từ câu trả lời DNS qua `dns.Resolver` (chỉ cho mục đích chặn UDP, không dùng để gán kết nối).
- ECH: không đọc được SNI nên không đếm được. Ghi vào `docs/setup.md` mục Limits; không có cách khắc phục trong phạm vi change này.

### 8. Tích hợp tick và lưu
Mỗi `Tick`, `refresh` nhận số giây dùng của bộ đo, cộng vào `UsedSec` của entry không active, reset khi `Day` khác ngày hiện tại (chỉ khi đồng hồ tin cậy), rồi dùng đường lưu hiện có (lưu trước khi áp dụng chặn). `UsedSec` lưu theo chu kỳ 60 giây của `refresh` và ngay khi entry đạt quota để restart không mở lại domain.

### 9. Cập nhật phần mềm
`checkUpdate`, `preserved.go` và `verify_linux.go` hiện chỉ xét `Lock`. Mở rộng để coi entry có quota là có cam kết cần giữ: sau khi khởi động lại, quota và `UsedSec` phải không giảm so với snapshot trong cùng ngày; rollback khôi phục state v3 từ sao lưu.

### 10. UI
Màn chính (`entries.go:mainView`/`entryLabel`) hiển thị quota còn lại cạnh domain. `lockedView` phân biệt "khóa theo lịch" với "hết quota, mở lại lúc 00:00 (còn Xh Ym)". Thêm màn nhập quota tái dùng bộ chọn thời lượng (`duration.go`), chỉ cho phép khi domain không active.

## Risks / Trade-offs

- **ECH hoặc QUIC làm SNI/kết nối không đọc được** (chưa kiểm chứng với Facebook/Instagram/Messenger) → chặn UDP 443; ghi giới hạn ECH vào tài liệu; chạy spike thật trước khi viết phần gán SNI.
- **Trình duyệt giữ socket sau khi đóng tab** (chưa kiểm chứng) → thời gian dùng có thể bị cộng thừa vài phút; chấp nhận và ghi tài liệu, đo bằng spike.
- **Messenger chạy dưới `facebook.com`** (chưa kiểm chứng) → cùng SNI nên tính vào `facebook.com`; chỉ `messenger.com` đếm riêng.
- **Bộ đo là thành phần mới có quyền cao, đọc gói** → fail-closed: lỗi bộ đo giữ nguyên chặn, báo `ERR_ENFORCEMENT`; parser chỉ đọc ClientHello, giới hạn kích thước, có test với dữ liệu hỏng.
- **Xóa quota hoặc domain trước khi hết quota để lách** → đây là lựa chọn đã chốt của người dùng, ghi nhận không giảm nhẹ.
- **Binary cũ không đọc được state v3** → giống lần nâng v1→v2; update giữ nguyên state qua snapshot/rollback.

## Migration Plan

1. Phát hành binary đọc v2 và ghi v3; state v2 cũ được nâng khi ghi lần đầu.
2. Update tự động chạy qua luồng giữ commitment đã mở rộng (mục 9); lỗi sau khởi động lại thì rollback bằng snapshot gồm cả state v3.
3. Rollback sang binary cũ không khả thi sau khi đã ghi v3; chỉ rollback trong transaction update, như cơ chế hiện có.

## Open Questions

- Số giây dung sai chấp nhận được khi trình duyệt giữ socket sau khi đóng tab (phụ thuộc kết quả spike).
- Có cần cấu hình riêng chu kỳ lưu `UsedSec` ngoài 60 giây hiện có không (có thể quyết khi cài đặt).

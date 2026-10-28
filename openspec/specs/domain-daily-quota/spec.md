# domain-daily-quota Specification

## Purpose

Cho phép người dùng giới hạn tổng thời gian dùng một domain trong một ngày; hết giới hạn thì domain bị chặn và không sửa được đến hết ngày.

## Requirements

### Requirement: Mỗi domain có tối đa một quota ngày
Daemon SHALL cho phép mỗi domain có tối đa một quota ngày, độc lập với lock lịch hoặc timed của domain đó và với quota của domain khác.

#### Scenario: Đặt quota cho domain chưa có quota
- **WHEN** người dùng xác nhận một quota 30 phút cho `facebook.com` chưa có quota
- **THEN** daemon lưu quota cho domain đó và không đổi quota hay lock của domain khác

#### Scenario: Đặt quota khi đã có quota
- **WHEN** người dùng tạo quota mới cho domain đã có quota và domain chưa hết quota
- **THEN** daemon thay quota cũ bằng quota mới và giữ nguyên thời gian đã dùng trong ngày

#### Scenario: Quota và lock cùng tồn tại
- **WHEN** domain đã có lock lịch và người dùng đặt thêm quota
- **THEN** cả hai được lưu và không trả `ERR_HAS_LOCK`

### Requirement: Giới hạn quota hợp lệ
Daemon SHALL chỉ nhận quota từ 1 phút đến 24 giờ, tính theo số phút nguyên, và từ chối giá trị khác bằng `ERR_INVALID`.

#### Scenario: Quota ngoài khoảng
- **WHEN** người dùng gửi quota `0m` hoặc `1441m`
- **THEN** daemon trả `ERR_INVALID` và không lưu gì

#### Scenario: Quota biên
- **WHEN** người dùng gửi quota `1m` hoặc `1440m`
- **THEN** daemon chấp nhận

### Requirement: Đặt quota cần mã xác nhận, xóa quota áp dụng trực tiếp
Daemon SHALL yêu cầu chuẩn bị và mã xác nhận cho lệnh đặt quota như với lock, SHALL chỉ lưu khi mã đúng, và SHALL lưu trước khi áp dụng chặn. Lệnh xóa quota SHALL áp dụng ngay không cần mã.

#### Scenario: Chuẩn bị đặt quota
- **WHEN** người dùng gửi lệnh đặt quota hợp lệ cho domain không active
- **THEN** daemon trả mã xác nhận kèm bản xem trước giới hạn và chưa lưu gì

#### Scenario: Xác nhận bằng mã đúng
- **WHEN** người dùng xác nhận đúng mã trên cùng kết nối trước khi hết hạn
- **THEN** daemon lưu quota và trả status mới

#### Scenario: Mã sai, hết hạn hoặc bị hủy
- **WHEN** mã sai ba lần, mã hết hạn hoặc lệnh bị hủy
- **THEN** daemon không lưu quota

#### Scenario: Xóa quota
- **WHEN** người dùng xóa quota của domain không active
- **THEN** daemon bỏ quota ngay không cần mã

#### Scenario: Lưu thất bại
- **WHEN** lưu state thất bại lúc xác nhận
- **THEN** daemon trả `ERR_STORAGE` và không áp dụng quota mới

### Requirement: Domain bị chặn khi lock active hoặc quota đã hết
Daemon SHALL coi domain là active khi lock của nó active hoặc thời gian đã dùng trong ngày đạt hoặc vượt quota, và SHALL chặn domain đó.

#### Scenario: Hết quota khi không có lock
- **WHEN** thời gian đã dùng của `facebook.com` đạt quota ngày
- **THEN** status báo `facebook.com` active và domain bị chặn

#### Scenario: Lock kết thúc nhưng quota đã hết
- **WHEN** lock lịch của domain kết thúc và quota của domain đã hết
- **THEN** domain vẫn active cho đến thời điểm reset ngày

#### Scenario: Quota còn nhưng lock active
- **WHEN** lock của domain đang active và quota còn dư
- **THEN** domain active theo lock và thời gian dùng không tăng

### Requirement: Quota chỉ đếm khi domain đang mở
Daemon SHALL chỉ cộng thời gian dùng của một domain trong lúc domain không active.

#### Scenario: Domain mở và đang dùng
- **WHEN** domain không active và đang được dùng theo capability `domain-usage-metering`
- **THEN** thời gian đã dùng của domain tăng theo thời gian trôi qua

#### Scenario: Domain không dùng
- **WHEN** domain không active nhưng không có hoạt động
- **THEN** thời gian đã dùng không tăng

### Requirement: Reset thời gian dùng lúc 00:00
Daemon SHALL đặt lại thời gian đã dùng của mọi domain về 0 vào 00:00 theo múi giờ của daemon, và SHALL mở lại domain đã bị chặn vì hết quota nếu không còn lock active.

#### Scenario: Qua nửa đêm
- **WHEN** đồng hồ tin cậy qua 00:00 theo múi giờ của daemon
- **THEN** thời gian đã dùng về 0 và domain hết quota được mở lại nếu lock không active

#### Scenario: Đồng hồ chưa tin cậy
- **WHEN** đồng hồ chưa tin cậy
- **THEN** daemon không reset ngày và giữ nguyên trạng thái chặn của quota

### Requirement: Không sửa được domain khi hết quota
Daemon SHALL trả `ERR_LOCKED` cho `edit_domain`, `delete_domain` và mọi lệnh thay đổi quota hoặc lock của domain đang active vì hết quota.

#### Scenario: Sửa domain hết quota
- **WHEN** người dùng gửi `edit_domain` cho domain đã hết quota
- **THEN** daemon trả `ERR_LOCKED` và không đổi state

#### Scenario: Xóa domain hết quota
- **WHEN** người dùng gửi `delete_domain` cho domain đã hết quota
- **THEN** daemon trả `ERR_LOCKED` và domain vẫn còn

#### Scenario: Sửa quota khi hết quota
- **WHEN** người dùng chuẩn bị đổi hoặc xóa quota của domain đã hết quota
- **THEN** daemon trả `ERR_LOCKED`

### Requirement: Sửa và xóa khi còn quota
Daemon SHALL cho phép đặt lại quota (kèm mã xác nhận), xóa quota và xóa domain khi domain không active.

#### Scenario: Xóa quota khi còn quota
- **WHEN** domain không active và người dùng xóa quota
- **THEN** daemon bỏ quota và domain không bị chặn vì quota

#### Scenario: Xóa domain có quota còn dư
- **WHEN** domain không active và người dùng xóa domain
- **THEN** daemon xóa domain cùng quota và thời gian đã dùng

### Requirement: Status báo quota còn lại
Status SHALL trả cho mọi domain có quota: giới hạn, thời gian đã dùng, thời gian còn lại và thời điểm reset kế tiếp, kể cả khi domain đang active.

#### Scenario: Domain đang dùng
- **WHEN** quota 30 phút và đã dùng 12 phút 34 giây
- **THEN** status báo còn lại 17 phút 26 giây và thời điểm reset là 00:00 ngày kế tiếp

#### Scenario: Domain hết quota
- **WHEN** quota đã hết
- **THEN** status báo còn lại 0 và kèm thời điểm reset để biết khi nào mở lại

#### Scenario: Domain không có quota
- **WHEN** domain không có quota
- **THEN** status không có trường quota cho domain đó

### Requirement: Quota bền qua khởi động lại và cập nhật
Daemon SHALL lưu quota và thời gian đã dùng vào state trước khi áp dụng chặn, và SHALL giữ chúng qua khởi động lại và cập nhật phần mềm.

#### Scenario: Khởi động lại trong ngày
- **WHEN** daemon khởi động lại cùng ngày sau khi domain đã dùng 10 phút
- **THEN** thời gian đã dùng vẫn từ 10 phút trở lên và domain hết quota vẫn bị chặn

#### Scenario: Cập nhật khi còn quota đã lưu
- **WHEN** có bản cập nhật và state còn quota hoặc lock
- **THEN** cập nhật chỉ được phép theo quy tắc hiện có cho lock đã lưu, và quota được khôi phục nguyên vẹn khi rollback

### Requirement: Quota chỉ hỗ trợ trên Linux
Daemon trên Windows SHALL từ chối mọi lệnh tạo hoặc sửa quota bằng `ERR_INVALID` kèm thông báo nêu rõ nền tảng không hỗ trợ.

#### Scenario: Lệnh quota trên Windows
- **WHEN** người dùng Windows gửi lệnh tạo quota
- **THEN** daemon trả `ERR_INVALID` với thông báo không hỗ trợ và không lưu gì

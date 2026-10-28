# domain-usage-metering Specification

## Purpose

Đo thời gian dùng thực tế của từng domain bằng kết nối mạng đang mở, gán cho đúng domain theo tên trong TLS, để quota đếm độc lập và phản ánh việc mở hoặc đóng tab.

## Requirements

### Requirement: Domain đang dùng khi có kết nối đang mở
Daemon SHALL coi một domain là đang được dùng khi có ít nhất một kết nối TCP đang mở (`ESTABLISHED`) được gán cho domain đó, và không được dùng khi không còn kết nối nào.

#### Scenario: Mở tab
- **WHEN** người dùng mở một tab tới domain và trình duyệt thiết lập kết nối TCP
- **THEN** domain được coi là đang dùng trong thời gian kết nối còn mở

#### Scenario: Đóng tab
- **WHEN** người dùng đóng tab và kết nối cuối cùng của domain đóng
- **THEN** domain được coi là không dùng và thời gian đã dùng ngừng tăng

#### Scenario: Tab im lặng
- **WHEN** tab đang mở nhưng không có lưu lượng đáng kể, kết nối vẫn `ESTABLISHED`
- **THEN** domain vẫn được coi là đang dùng

### Requirement: Gán kết nối theo tên miền TLS
Daemon SHALL gán mỗi kết nối cho entry có tên khớp với server name trong TLS ClientHello của kết nối đó, và không được gán theo địa chỉ IP.

#### Scenario: Hai domain chung IP
- **WHEN** `facebook.com` và `instagram.com` được phục vụ từ cùng một IP và người dùng chỉ mở Instagram
- **THEN** chỉ thời gian dùng của `instagram.com` tăng

#### Scenario: Tên ngoài danh sách
- **WHEN** kết nối có server name không khớp entry nào
- **THEN** kết nối không được tính cho domain nào

#### Scenario: Subdomain không liệt kê
- **WHEN** kết nối có server name là subdomain không được liệt kê riêng của domain đã cấu hình
- **THEN** kết nối không được tính cho entry của domain cha, theo quy tắc khớp tên hiện có

### Requirement: Đếm độc lập giữa các domain
Daemon SHALL tính thời gian dùng của từng entry riêng biệt; một kết nối chỉ được cộng cho đúng một entry.

#### Scenario: Dùng song song hai domain
- **WHEN** người dùng mở `messenger.com` và `facebook.com` cùng lúc
- **THEN** mỗi entry tăng thời gian dùng theo kết nối của chính nó, không cộng chéo

### Requirement: Cắt kết nối khi hết quota
Daemon SHALL chấm dứt các kết nối đang mở đã gán cho domain vừa hết quota và chặn kết nối mới đến hết ngày.

#### Scenario: Hết quota giữa phiên
- **WHEN** domain đạt quota trong lúc tab đang mở
- **THEN** các kết nối đã gán bị cắt và tab không tiếp tục nhận dữ liệu

#### Scenario: Mở lại sau khi hết quota
- **WHEN** người dùng mở lại domain đã hết quota trước 00:00
- **THEN** kết nối mới không thành công

### Requirement: Giao thức không đo được bị xử lý an toàn
Daemon SHALL không để lưu lượng của domain thoát khỏi quota vì giao thức không đọc được tên miền, bằng cách chặn UDP 443 của domain đang có quota để buộc trình duyệt dùng TCP.

#### Scenario: QUIC
- **WHEN** trình duyệt thử HTTP/3 qua UDP 443 tới domain có quota
- **THEN** UDP 443 bị chặn và trình duyệt quay về TCP để được đo

#### Scenario: ECH ẩn tên miền
- **WHEN** kết nối dùng Encrypted ClientHello nên không đọc được server name
- **THEN** daemon không tính kết nối cho domain nào và ghi nhận giới hạn này trong tài liệu

### Requirement: Bộ đo lỗi thì không nới lỏng chặn
Khi bộ đo kết nối lỗi hoặc không khởi động được, daemon SHALL giữ nguyên mọi chặn đang có, SHALL không coi quota là còn dư vì mất số liệu, và SHALL báo `ERR_ENFORCEMENT` trong status.

#### Scenario: Bộ đo ngừng giữa ngày
- **WHEN** bộ đo lỗi trong lúc domain đã hết quota
- **THEN** domain vẫn bị chặn và status báo `ERR_ENFORCEMENT`

#### Scenario: Khởi động không có bộ đo
- **WHEN** daemon khởi động mà không dựng được bộ đo
- **THEN** quota đã lưu vẫn được áp dụng cho domain đã hết, status báo `ERR_ENFORCEMENT`

### Requirement: Chỉ chạy trên Linux
Bộ đo SHALL chỉ khả dụng trên Linux; trên nền tảng khác daemon SHALL không bật quota.

#### Scenario: Windows
- **WHEN** daemon chạy trên Windows
- **THEN** không có bộ đo và lệnh quota bị từ chối theo capability `domain-daily-quota`

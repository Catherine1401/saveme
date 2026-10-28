package meter

import (
	"context"
	"errors"
	"time"
	"unsafe"

	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"
)

const captureBuffer = 1 << 16

// Capture đọc thụ động ClientHello của các kết nối TCP 443 đi ra và gán chúng vào Table.
type Capture struct {
	Table *Table
	fds   []int
}

func htons(value uint16) uint16 { return value<<8 | value>>8 }

// Mở socket AF_PACKET (không cần xác nhận gói) cho một họ địa chỉ kèm bộ lọc BPF.
// ETH_P_ALL là bắt buộc để nhận cả gói đi ra; socket gắn một giao thức cụ thể chỉ nhận gói đi vào.
func openPacket(ipv6 bool) (int, error) {
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, int(htons(unix.ETH_P_ALL)))
	if err != nil {
		return -1, err
	}
	raw, err := bpf.Assemble(CaptureFilter(ipv6))
	if err != nil {
		unix.Close(fd)
		return -1, err
	}
	filter := make([]unix.SockFilter, len(raw))
	for index, instruction := range raw {
		filter[index] = unix.SockFilter{Code: instruction.Op, Jt: instruction.Jt, Jf: instruction.Jf, K: instruction.K}
	}
	program := unix.SockFprog{Len: uint16(len(filter)), Filter: (*unix.SockFilter)(unsafe.Pointer(&filter[0]))}
	if err := unix.SetsockoptSockFprog(fd, unix.SOL_SOCKET, unix.SO_ATTACH_FILTER, &program); err != nil {
		unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

// Open mở socket bắt gói; lỗi (thường là thiếu quyền) nghĩa là bộ đo không khởi động được.
func (c *Capture) Open() error {
	for _, ipv6 := range []bool{false, true} {
		fd, err := openPacket(ipv6)
		if err != nil {
			c.Close()
			return err
		}
		c.fds = append(c.fds, fd)
	}
	return nil
}

func (c *Capture) Close() {
	for _, fd := range c.fds {
		unix.Close(fd)
	}
	c.fds = nil
}

// Run đọc gói tới khi ctx kết thúc hoặc socket lỗi.
func (c *Capture) Run(ctx context.Context) error {
	results := make(chan error, len(c.fds))
	for _, fd := range c.fds {
		go func() { results <- c.read(fd) }()
	}
	select {
	case <-ctx.Done():
		c.Close()
		for range c.fds {
			<-results
		}
		return nil
	case err := <-results:
		c.Close()
		return err
	}
}

func (c *Capture) read(fd int) error {
	flows := NewFlows(time.Now)
	buffer := make([]byte, captureBuffer)
	for {
		size, _, err := unix.Recvfrom(fd, buffer, 0)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return err
		}
		segment, ok := ParsePacket(buffer[:size])
		if !ok {
			continue
		}
		segment.Payload = append([]byte{}, segment.Payload...)
		if name, done := flows.Feed(segment); done {
			c.Table.Assign(segment.Tuple, name)
		}
	}
}

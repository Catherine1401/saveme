package meter

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"crypto/tls"

	"golang.org/x/sys/unix"
)

// Cần root (CAP_NET_RAW): bắt ClientHello thật tới một server TLS loopback rồi kiểm bảng gán.
func TestCaptureAssignsLoopbackTLSConnection(t *testing.T) {
	const name, wait = "www.instagram.com", 3 * time.Second
	table := NewTable()
	table.SetDomains([]string{"instagram.com"})
	capture := &Capture{Table: table}
	if err := capture.Open(); err != nil {
		if errors.Is(err, unix.EPERM) || errors.Is(err, unix.EACCES) {
			t.Skip("AF_PACKET needs CAP_NET_RAW:", err)
		}
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go capture.Run(ctx)
	// Loopback dùng cổng ngẫu nhiên nên bộ lọc cổng 443 không khớp; chỉ kiểm khi chạy được trên 443.
	listener, err := net.Listen("tcp", "127.0.0.1:443")
	if err != nil {
		t.Skip("cannot bind 443:", err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
	}()
	conn, err := tls.Dial("tcp", "127.0.0.1:443", &tls.Config{ServerName: name, InsecureSkipVerify: true})
	if conn != nil {
		defer conn.Close()
	}
	_ = err
	local := netip.MustParseAddrPort("127.0.0.1:443")
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		for _, tuple := range table.Tuples([]string{"instagram.com"}) {
			if tuple.Remote == local {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("ClientHello not assigned")
}

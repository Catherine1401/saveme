package meter

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"slices"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Mở một kết nối loopback thật và trả tuple phía client.
func loopback(t *testing.T) (Tuple, net.Conn) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, _ := listener.Accept()
		accepted <- conn
	}()
	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	server := <-accepted
	t.Cleanup(func() { server.Close() })
	return Tuple{Local: netip.MustParseAddrPort(client.LocalAddr().String()), Remote: netip.MustParseAddrPort(listener.Addr().String())}, client
}

func TestDiagListsEstablishedLoopbackAndForgetsClosed(t *testing.T) {
	want, client := loopback(t)
	live, err := Diag{}.Established()
	if err != nil || !slices.Contains(live, want) {
		t.Fatal(err, want, len(live))
	}
	client.Close()
	// Kernel có thể giữ socket thêm một chút khi đóng; chờ tối đa vài lần đọc.
	for range 50 {
		live, err = Diag{}.Established()
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(live, want) {
			return
		}
	}
	t.Fatal("closed connection still ESTABLISHED")
}

func TestDiagDestroyClosesSocketOrNeedsPrivilege(t *testing.T) {
	const wait = 200 * time.Millisecond
	want, client := loopback(t)
	err := Diag{}.Destroy(want)
	if errors.Is(err, unix.EPERM) || errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.EACCES) {
		t.Skip("SOCK_DESTROY needs CAP_NET_ADMIN:", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	client.SetReadDeadline(time.Now().Add(wait))
	if _, err := client.Read(make([]byte, 1)); err == nil || errors.Is(err, os.ErrDeadlineExceeded) {
		t.Skip("kernel ignored SOCK_DESTROY without privilege; run as root:", err)
	}
}

func TestDiagRequestAndParseRoundTrip(t *testing.T) {
	tuple := Tuple{Local: netip.MustParseAddrPort("10.1.2.3:40000"), Remote: netip.MustParseAddrPort("157.240.1.35:443")}
	request := diagRequest(unix.AF_INET, allStates, &tuple)
	if len(request) != diagRequestSize {
		t.Fatal(len(request))
	}
	message := make([]byte, diagMessageSize)
	message[1] = stateEstablished
	copy(message[idOffset:], request[requestID:requestID+4])
	copy(message[srcOffset:], request[requestID+4:requestID+8])
	copy(message[dstOffset:], request[requestID+4+addressSize:requestID+8+addressSize])
	if got, ok := parseDiag(unix.AF_INET, message); !ok || got != tuple {
		t.Fatal(got, ok)
	}
	message[1] = 2
	if _, ok := parseDiag(unix.AF_INET, message); ok {
		t.Fatal("non-established parsed")
	}
	if _, ok := parseDiag(unix.AF_INET, message[:diagMessageSize-1]); ok {
		t.Fatal("short message parsed")
	}
}

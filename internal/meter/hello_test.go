package meter

import (
	"crypto/tls"
	"errors"
	"net"
	"testing"
)

// Sinh ClientHello thật bằng crypto/tls và lấy record đầu tiên.
func realHello(t *testing.T, name string) []byte {
	t.Helper()
	const bufferSize = 4096
	client, server := net.Pipe()
	defer server.Close()
	go func() {
		conn := tls.Client(client, &tls.Config{ServerName: name, InsecureSkipVerify: true})
		conn.Handshake()
		client.Close()
	}()
	buffer := make([]byte, bufferSize)
	size, err := server.Read(buffer)
	if err != nil {
		t.Fatal(err)
	}
	return buffer[:size]
}

func TestServerNameFromRealClientHello(t *testing.T) {
	const name = "www.Facebook.com"
	got, err := ServerName(realHello(t, name))
	if err != nil || got != "www.facebook.com" {
		t.Fatal(got, err)
	}
}

func TestServerNameNeedsMoreBytesWhenTruncated(t *testing.T) {
	hello := realHello(t, "instagram.com")
	for _, size := range []int{0, 3, recordHeader, len(hello) / 2, len(hello) - 1} {
		if _, err := ServerName(hello[:size]); !errors.Is(err, ErrIncomplete) {
			t.Fatalf("size %d: %v", size, err)
		}
	}
}

func TestServerNameRejectsCorruptData(t *testing.T) {
	hello := realHello(t, "instagram.com")
	notTLS := []byte("GET / HTTP/1.1\r\nHost: instagram.com\r\n\r\n")
	badType := append([]byte{}, hello...)
	badType[recordHeader] = 2
	oversize := append([]byte{}, hello...)
	oversize[3], oversize[4] = 0xff, 0xff
	for name, data := range map[string][]byte{"http": notTLS, "wrong handshake": badType, "oversize record": oversize} {
		if _, err := ServerName(data); !errors.Is(err, ErrMalformed) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// Mọi cách cắt và sửa từng byte không được làm panic.
	for index := range hello {
		mutated := append([]byte{}, hello...)
		mutated[index] ^= 0xff
		ServerName(mutated)
	}
}

func TestServerNameMissingWithoutSNI(t *testing.T) {
	hello := realHello(t, "")
	if _, err := ServerName(hello); !errors.Is(err, ErrNoName) {
		t.Fatal(err)
	}
}

package meter

import (
	"encoding/binary"
	"errors"
	"net/netip"

	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

const (
	diagRequestSize  = 56
	diagMessageSize  = 72
	stateEstablished = 1
	addressSize      = 16
	idOffset         = 4 // inet_diag_msg: id ngay sau 4 byte family/state/timer/retrans
	requestID        = 8 // inet_diag_req_v2: id sau family/protocol/ext/pad và states
	srcOffset        = idOffset + 4
	dstOffset        = srcOffset + addressSize
	noCookie         = 0xffffffff
	allStates        = 0xffffffff
)

// Diag đọc và hủy socket TCP qua netlink sock_diag (như `ss`).
type Diag struct{}

// Xây inet_diag_req_v2; tuple rỗng nghĩa là liệt kê mọi socket theo states.
func diagRequest(family uint8, states uint32, tuple *Tuple) []byte {
	const protocolOffset, statesOffset, sportOffset, dportOffset = 1, 4, requestID, requestID + 2
	const requestSrc, requestDst, cookieOffset = requestID + 4, requestID + 4 + addressSize, requestID + 4 + 2*addressSize + 4
	data := make([]byte, diagRequestSize)
	data[0], data[protocolOffset] = family, unix.IPPROTO_TCP
	binary.NativeEndian.PutUint32(data[statesOffset:], states)
	if tuple == nil {
		return data
	}
	binary.BigEndian.PutUint16(data[sportOffset:], tuple.Local.Port())
	binary.BigEndian.PutUint16(data[dportOffset:], tuple.Remote.Port())
	local, remote := tuple.Local.Addr().As16(), tuple.Remote.Addr().As16()
	if family == unix.AF_INET {
		copy(data[requestSrc:], tuple.Local.Addr().AsSlice())
		copy(data[requestDst:], tuple.Remote.Addr().AsSlice())
	} else {
		copy(data[requestSrc:], local[:])
		copy(data[requestDst:], remote[:])
	}
	binary.NativeEndian.PutUint32(data[cookieOffset:], noCookie)
	binary.NativeEndian.PutUint32(data[cookieOffset+4:], noCookie)
	return data
}

// Giải mã inet_diag_msg thành Tuple; chỉ nhận socket ESTABLISHED.
func parseDiag(family uint8, data []byte) (Tuple, bool) {
	const stateOffset, sportOffset, dportOffset = 1, idOffset, idOffset + 2
	if len(data) < diagMessageSize || data[stateOffset] != stateEstablished {
		return Tuple{}, false
	}
	read := func(offset int) netip.Addr {
		if family == unix.AF_INET {
			return netip.AddrFrom4([4]byte(data[offset : offset+4]))
		}
		return netip.AddrFrom16([16]byte(data[offset : offset+addressSize])).Unmap()
	}
	local := netip.AddrPortFrom(read(srcOffset), binary.BigEndian.Uint16(data[sportOffset:]))
	remote := netip.AddrPortFrom(read(dstOffset), binary.BigEndian.Uint16(data[dportOffset:]))
	return Tuple{Local: local, Remote: remote}, true
}

// Established liệt kê mọi kết nối TCP đang ESTABLISHED, cả IPv4 và IPv6.
func (Diag) Established() ([]Tuple, error) {
	conn, err := netlink.Dial(unix.NETLINK_SOCK_DIAG, nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	var result []Tuple
	for _, family := range []uint8{unix.AF_INET, unix.AF_INET6} {
		request := netlink.Message{
			Header: netlink.Header{Type: unix.SOCK_DIAG_BY_FAMILY, Flags: netlink.Request | netlink.Dump},
			Data:   diagRequest(family, 1<<stateEstablished, nil),
		}
		replies, err := conn.Execute(request)
		if err != nil {
			return nil, err
		}
		for _, reply := range replies {
			if tuple, ok := parseDiag(family, reply.Data); ok {
				result = append(result, tuple)
			}
		}
	}
	return result, nil
}

// Destroy hủy một socket đã gán; socket đã đóng sẵn không phải lỗi.
func (Diag) Destroy(tuple Tuple) error {
	family := uint8(unix.AF_INET6)
	if tuple.Local.Addr().Is4() {
		family = unix.AF_INET
	}
	conn, err := netlink.Dial(unix.NETLINK_SOCK_DIAG, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	request := netlink.Message{
		Header: netlink.Header{Type: unix.SOCK_DESTROY, Flags: netlink.Request | netlink.Acknowledge},
		Data:   diagRequest(family, allStates, &tuple),
	}
	if _, err := conn.Execute(request); err != nil && !errors.Is(err, unix.ENOENT) {
		return err
	}
	return nil
}

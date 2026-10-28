package meter

import (
	"encoding/binary"
	"errors"
	"net/netip"
	"time"
)

const (
	protocolTCP    = 6
	httpsPort      = 443
	ipv4Minimum    = 20
	ipv6Header     = 40
	tcpMinimum     = 20
	maxStream      = recordHeader + maxRecord
	maxFlows       = 2048
	flowLifetime   = 10 * time.Second
	versionIPv4    = 4
	versionIPv6    = 6
	ipv4Protocol   = 9
	ipv6Next       = 6
	ipv4Source     = 12
	ipv6Source     = 8
	tcpSeqOffset   = 4
	tcpOffsetByte  = 12
	tcpOffsetShift = 4
	tcpOffsetUnit  = 4
)

// Segment là một gói TCP từ máy này tới cổng 443 mang dữ liệu.
type Segment struct {
	Tuple   Tuple
	Seq     uint32
	Payload []byte
}

// ParsePacket đọc gói bắt từ AF_PACKET bắt đầu ở header mạng; chỉ nhận TCP tới cổng 443 có payload.
func ParsePacket(packet []byte) (Segment, bool) {
	var source, destination netip.Addr
	var transport []byte
	switch {
	case len(packet) >= ipv4Minimum && packet[0]>>4 == versionIPv4:
		header := int(packet[0]&0xf) * tcpOffsetUnit
		if header < ipv4Minimum || len(packet) < header || packet[ipv4Protocol] != protocolTCP {
			return Segment{}, false
		}
		source = netip.AddrFrom4([4]byte(packet[ipv4Source : ipv4Source+4]))
		destination = netip.AddrFrom4([4]byte(packet[ipv4Source+4 : ipv4Source+8]))
		transport = packet[header:]
	case len(packet) >= ipv6Header && packet[0]>>4 == versionIPv6:
		if packet[ipv6Next] != protocolTCP {
			return Segment{}, false
		}
		source = netip.AddrFrom16([16]byte(packet[ipv6Source : ipv6Source+16]))
		destination = netip.AddrFrom16([16]byte(packet[ipv6Source+16 : ipv6Source+32]))
		transport = packet[ipv6Header:]
	default:
		return Segment{}, false
	}
	if len(transport) < tcpMinimum {
		return Segment{}, false
	}
	offset := int(transport[tcpOffsetByte]>>tcpOffsetShift) * tcpOffsetUnit
	if offset < tcpMinimum || offset >= len(transport) || binary.BigEndian.Uint16(transport[2:4]) != httpsPort {
		return Segment{}, false
	}
	tuple := Tuple{
		Local:  netip.AddrPortFrom(source.Unmap(), binary.BigEndian.Uint16(transport[0:2])),
		Remote: netip.AddrPortFrom(destination.Unmap(), httpsPort),
	}
	return Segment{Tuple: tuple, Seq: binary.BigEndian.Uint32(transport[tcpSeqOffset:]), Payload: transport[offset:]}, true
}

type flow struct {
	base    uint32
	stream  []byte
	expires time.Time
}

// Flows gom các byte đầu của mỗi kết nối cho tới khi đọc được server name.
type Flows struct {
	flows map[Tuple]*flow
	now   func() time.Time
}

func NewFlows(now func() time.Time) *Flows { return &Flows{flows: map[Tuple]*flow{}, now: now} }

// Feed nhận một segment; trả tên và true khi ClientHello của kết nối đã đọc xong.
func (f *Flows) Feed(segment Segment) (string, bool) {
	const none = ""
	current, tracked := f.flows[segment.Tuple]
	if !tracked {
		if segment.Payload[0] != recordHandshake {
			return none, false
		}
		if len(f.flows) >= maxFlows {
			f.flows = map[Tuple]*flow{}
		}
		current = &flow{base: segment.Seq, expires: f.now().Add(flowLifetime)}
		f.flows[segment.Tuple] = current
	}
	offset := int(segment.Seq - current.base)
	if f.now().After(current.expires) || offset < 0 || offset > len(current.stream) {
		delete(f.flows, segment.Tuple)
		return none, false
	}
	if end := offset + len(segment.Payload); end > len(current.stream) {
		current.stream = append(current.stream, segment.Payload[len(current.stream)-offset:]...)
	}
	name, err := ServerName(current.stream)
	if errors.Is(err, ErrIncomplete) && len(current.stream) < maxStream {
		return none, false
	}
	delete(f.flows, segment.Tuple)
	return name, err == nil
}

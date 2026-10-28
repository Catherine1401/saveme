package meter

import "golang.org/x/net/bpf"

const (
	acceptAll      = 0xffff
	dataOffsetMask = 0xf0
	dataOffsetPow  = 2
	portOffset     = 2
	flagsOffset    = 12
	ipv6PortOffset = ipv6Header + portOffset
	versionShift   = 4
)

// CaptureFilter trả chương trình BPF giữ gói TCP tới cổng 443 có payload; gói bắt ở header mạng (SOCK_DGRAM).
func CaptureFilter(ipv6 bool) []bpf.Instruction {
	const reject = 0
	tcpPayload := []bpf.Instruction{
		bpf.LoadIndirect{Off: flagsOffset, Size: 1},
		bpf.ALUOpConstant{Op: bpf.ALUOpAnd, Val: dataOffsetMask},
		bpf.ALUOpConstant{Op: bpf.ALUOpShiftRight, Val: dataOffsetPow},
		bpf.ALUOpX{Op: bpf.ALUOpAdd},
		bpf.TAX{},
		bpf.LoadIndirect{Off: 0, Size: 1}, // vượt biên khi không có payload nên gói bị loại
		bpf.RetConstant{Val: acceptAll},
		bpf.RetConstant{Val: reject},
	}
	version := uint32(versionIPv4 << versionShift)
	if ipv6 {
		version = versionIPv6 << versionShift
	}
	// Socket ETH_P_ALL nhận cả IPv4 và IPv6 nên bộ lọc tự kiểm version ở byte đầu.
	check := []bpf.Instruction{
		bpf.LoadAbsolute{Off: 0, Size: 1},
		bpf.ALUOpConstant{Op: bpf.ALUOpAnd, Val: dataOffsetMask},
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: version, SkipFalse: 12},
	}
	if ipv6 {
		head := []bpf.Instruction{
			bpf.LoadAbsolute{Off: ipv6Next, Size: 1},
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: protocolTCP, SkipFalse: 10},
			bpf.LoadAbsolute{Off: ipv6PortOffset, Size: 2},
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: httpsPort, SkipFalse: 8},
			bpf.LoadConstant{Dst: bpf.RegX, Val: ipv6Header},
		}
		return append(append(check, head...), tcpPayload...)
	}
	head := []bpf.Instruction{
		bpf.LoadAbsolute{Off: ipv4Protocol, Size: 1},
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: protocolTCP, SkipFalse: 10},
		bpf.LoadMemShift{Off: 0},
		bpf.LoadIndirect{Off: portOffset, Size: 2},
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: httpsPort, SkipFalse: 7},
	}
	return append(append(check, head...), tcpPayload...)
}

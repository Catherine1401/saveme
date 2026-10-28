package meter

import (
	"net/netip"
	"slices"
	"sync"

	"saveme/internal/core"
)

// Tuple định danh một kết nối TCP; Local là đầu máy này, Remote là máy chủ.
type Tuple struct {
	Local  netip.AddrPort
	Remote netip.AddrPort
}

// Table gán kết nối cho entry theo server name; mỗi kết nối chỉ thuộc tối đa một entry.
type Table struct {
	mu       sync.Mutex
	domains  []string
	assigned map[Tuple]string
	addrs    map[netip.Addr]string
}

func NewTable() *Table { return &Table{assigned: map[Tuple]string{}, addrs: map[netip.Addr]string{}} }

// SetDomains đặt danh sách domain có quota; kết nối đã gán cho domain bị bỏ sẽ bị quên.
func (t *Table) SetDomains(domains []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.domains = append([]string{}, domains...)
	for tuple, domain := range t.assigned {
		if !slices.Contains(t.domains, domain) {
			delete(t.assigned, tuple)
		}
	}
	for addr, domain := range t.addrs {
		if !slices.Contains(t.domains, domain) {
			delete(t.addrs, addr)
		}
	}
}

// Assign gán kết nối cho entry khớp tên theo quy tắc chặn hiện có; trả domain được gán hoặc "".
func (t *Table) Assign(tuple Tuple, name string) string {
	const none = ""
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, domain := range t.domains {
		if core.DomainBlocked(name, []string{domain}) {
			t.assigned[tuple] = domain
			t.learn(domain, tuple.Remote.Addr())
			return domain
		}
	}
	return none
}

// Domain trả entry của kết nối đã gán.
func (t *Table) Domain(tuple Tuple) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	domain, ok := t.assigned[tuple]
	return domain, ok
}

// Tuples liệt kê kết nối đã gán cho các domain cho trước.
func (t *Table) Tuples(domains []string) []Tuple {
	t.mu.Lock()
	defer t.mu.Unlock()
	var result []Tuple
	for tuple, domain := range t.assigned {
		if slices.Contains(domains, domain) {
			result = append(result, tuple)
		}
	}
	return result
}

// Retain giữ lại các kết nối còn trong danh sách đang mở.
func (t *Table) Retain(live []Tuple) {
	t.mu.Lock()
	defer t.mu.Unlock()
	keep := make(map[Tuple]struct{}, len(live))
	for _, tuple := range live {
		keep[tuple] = struct{}{}
	}
	for tuple := range t.assigned {
		if _, ok := keep[tuple]; !ok {
			delete(t.assigned, tuple)
		}
	}
}

const maxAddrs = 4096

func (t *Table) learn(domain string, addr netip.Addr) {
	if len(t.addrs) < maxAddrs {
		t.addrs[addr] = domain
	}
}

// Observe ghi địa chỉ IP trong câu trả lời DNS của tên khớp một domain có quota, chỉ để chặn UDP 443.
func (t *Table) Observe(name string, addrs []netip.Addr) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, domain := range t.domains {
		if core.DomainBlocked(name, []string{domain}) {
			for _, addr := range addrs {
				t.learn(domain, addr.Unmap())
			}
			return
		}
	}
}

// Addrs liệt kê địa chỉ đã biết của các domain có quota.
func (t *Table) Addrs() []netip.Addr {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]netip.Addr, 0, len(t.addrs))
	for addr := range t.addrs {
		result = append(result, addr)
	}
	slices.SortFunc(result, netip.Addr.Compare)
	return result
}

// Used liệt kê domain đang có ít nhất một kết nối đã gán.
func (t *Table) Used() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var result []string
	for _, domain := range t.assigned {
		if !slices.Contains(result, domain) {
			result = append(result, domain)
		}
	}
	slices.Sort(result)
	return result
}

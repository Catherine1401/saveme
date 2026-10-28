package meter

import (
	"net/netip"
	"sync"
)

// Source đọc và hủy kết nối TCP của hệ điều hành.
type Source interface {
	Established() ([]Tuple, error)
	Destroy(Tuple) error
}

// Firewall chặn kết nối đã gán theo 5-tuple và UDP 443 tới các địa chỉ của domain có quota.
type Firewall interface {
	Drop(tuples []Tuple, udp []netip.Addr) error
}

// Meter đo domain đang dùng từ kết nối đang mở đã gán theo SNI.
type Meter struct {
	Table    *Table
	Source   Source
	Firewall Firewall
	mu       sync.Mutex
	failure  error
}

// Fail đánh dấu bộ đo đã hỏng (ví dụ capture dừng); mọi lần đọc sau đó đều trả lỗi này.
func (m *Meter) Fail(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failure = err
}

func (m *Meter) failed() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.failure
}

// InUse trả các domain có kết nối ESTABLISHED đã gán; lỗi đọc kết nối là lỗi của bộ đo.
func (m *Meter) InUse(quota []string) ([]string, error) {
	if err := m.failed(); err != nil {
		return nil, err
	}
	m.Table.SetDomains(quota)
	live, err := m.Source.Established()
	if err != nil {
		return nil, err
	}
	m.Table.Retain(live)
	return m.Table.Used(), nil
}

// Cut hủy kết nối đã gán cho domain bị chặn (best-effort) và dựng rule drop 5-tuple cùng UDP 443; lỗi rule là lỗi của bộ đo.
func (m *Meter) Cut(blocked []string) error {
	tuples := m.Table.Tuples(blocked)
	for _, tuple := range tuples {
		m.Source.Destroy(tuple)
	}
	return m.Firewall.Drop(tuples, m.Table.Addrs())
}

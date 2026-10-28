package enforce

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
	"saveme/internal/meter"
)

const (
	quotaTableName  = "saveme_quota"
	quotaChainName  = "quota_output"
	quicMarker      = "saveme:quota-udp443"
	tupleMarker     = "saveme:quota-tuple"
	maxQuotaRules   = 1024
	quotaRecheck    = 30 * time.Second
	quicPort        = 443
	ipv4SourceStart = 12
	ipv6SourceStart = 8
)

// QuotaFirewall giữ bảng nftables riêng cho quota: drop 5-tuple đã cắt và UDP 443 tới địa chỉ của domain có quota.
type QuotaFirewall struct {
	mu      sync.Mutex
	last    string
	checked time.Time
}

func quotaTable() *nftables.Table {
	return &nftables.Table{Name: quotaTableName, Family: nftables.TableFamilyINet}
}

// Drop dựng lại bảng khi nội dung đổi hoặc theo chu kỳ kiểm lại; không còn gì để chặn thì xóa bảng.
func (q *QuotaFirewall) Drop(tuples []meter.Tuple, udp []netip.Addr) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	key := fmt.Sprint(tuples, udp)
	if key == q.last && time.Since(q.checked) < quotaRecheck {
		return nil
	}
	connection := &nftables.Conn{}
	target := quotaTable()
	if err := dropQuotaTable(connection, target); err != nil {
		return err
	}
	if len(tuples) > 0 || len(udp) > 0 {
		chain := &nftables.Chain{Name: quotaChainName, Table: target, Type: nftables.ChainTypeFilter, Hooknum: nftables.ChainHookOutput, Priority: nftables.ChainPriorityFilter}
		connection.AddTable(target)
		connection.AddChain(chain)
		for _, rule := range quotaRules(target, chain, tuples, udp) {
			connection.AddRule(rule)
		}
	}
	if err := connection.Flush(); err != nil {
		return err
	}
	q.last, q.checked = key, time.Now()
	return nil
}

// Đưa lệnh xóa bảng quota vào batch nếu bảng đang tồn tại.
func dropQuotaTable(connection *nftables.Conn, target *nftables.Table) error {
	tables, err := connection.ListTables()
	if err != nil {
		return err
	}
	for _, existing := range tables {
		if existing.Name == target.Name && existing.Family == target.Family {
			connection.DelTable(target)
		}
	}
	return nil
}

// Clear xóa bảng quota (khi tắt daemon hoặc gỡ cài đặt).
func (q *QuotaFirewall) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.last = ""
	connection := &nftables.Conn{}
	if err := dropQuotaTable(connection, quotaTable()); err != nil {
		return err
	}
	return connection.Flush()
}

func quotaRules(target *nftables.Table, chain *nftables.Chain, tuples []meter.Tuple, udp []netip.Addr) []*nftables.Rule {
	var rules []*nftables.Rule
	for _, tuple := range tuples {
		if len(rules) >= maxQuotaRules {
			return rules
		}
		exprs := append(protocolPort(unix.IPPROTO_TCP, tuple.Remote.Port()), addressMatch(net.IP(tuple.Remote.Addr().AsSlice()))...)
		exprs = append(exprs, sourceMatch(tuple.Local)...)
		exprs = append(exprs, &expr.Verdict{Kind: expr.VerdictDrop})
		rules = append(rules, &nftables.Rule{Table: target, Chain: chain, Exprs: exprs, UserData: []byte(tupleMarker)})
	}
	for _, addr := range udp {
		if len(rules) >= maxQuotaRules {
			return rules
		}
		exprs := append(protocolPort(unix.IPPROTO_UDP, quicPort), addressMatch(net.IP(addr.AsSlice()))...)
		exprs = append(exprs, &expr.Verdict{Kind: expr.VerdictDrop})
		rules = append(rules, &nftables.Rule{Table: target, Chain: chain, Exprs: exprs, UserData: []byte(quicMarker)})
	}
	return rules
}

// Khớp địa chỉ và cổng nguồn của kết nối cục bộ.
func sourceMatch(local netip.AddrPort) []expr.Any {
	const register, portOffset, portSize = 1, 0, 2
	offset, size := uint32(ipv6SourceStart), uint32(16)
	data := local.Addr().As16()
	value := data[:]
	if local.Addr().Is4() {
		four := local.Addr().As4()
		offset, size, value = ipv4SourceStart, 4, four[:]
	}
	port := make([]byte, portSize)
	binary.BigEndian.PutUint16(port, local.Port())
	return []expr.Any{
		&expr.Payload{DestRegister: register, Base: expr.PayloadBaseNetworkHeader, Offset: offset, Len: size},
		&expr.Cmp{Op: expr.CmpOpEq, Register: register, Data: value},
		&expr.Payload{DestRegister: register, Base: expr.PayloadBaseTransportHeader, Offset: portOffset, Len: portSize},
		&expr.Cmp{Op: expr.CmpOpEq, Register: register, Data: port},
	}
}

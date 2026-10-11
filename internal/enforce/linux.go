package enforce

import (
	"bytes"
	"encoding/binary"
	"net"
	"sync"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
	"saveme/internal/core"
)

type Linux struct {
	mu         sync.Mutex
	ForwardUID uint32
	DoH        []net.IP
	Filter     *core.Filter
}

func table() *nftables.Table {
	const name = "saveme"
	return &nftables.Table{Name: name, Family: nftables.TableFamilyINet}
}

func chains(target *nftables.Table) []*nftables.Chain {
	const dnsName, filterName = "dns_output", "filter_output"
	return []*nftables.Chain{
		{Name: dnsName, Table: target, Type: nftables.ChainTypeNAT, Hooknum: nftables.ChainHookOutput, Priority: nftables.ChainPriorityNATDest},
		{Name: filterName, Table: target, Type: nftables.ChainTypeFilter, Hooknum: nftables.ChainHookOutput, Priority: nftables.ChainPriorityFilter},
	}
}

func (l *Linux) Apply(state core.State, active bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Filter.Set(state, active)
	return l.ensure(active)
}

func (l *Linux) ensure(active bool) error {
	const dnsIndex, filterIndex = 0, 1
	connection := &nftables.Conn{}
	target := table()
	tables, err := connection.ListTables()
	if err != nil {
		return err
	}
	found := false
	for _, existing := range tables {
		found = found || (existing.Name == target.Name && existing.Family == target.Family)
	}
	if !active {
		if !found {
			return nil
		}
		connection.DelTable(target)
		return connection.Flush()
	}
	chainList := chains(target)
	expected := l.rules(target, chainList[dnsIndex], chainList[filterIndex])
	if found && intact(connection, chainList, expected) {
		return nil
	}
	if found {
		connection.DelTable(target)
	}
	connection.AddTable(target)
	for _, chain := range chainList {
		connection.AddChain(chain)
	}
	for _, rule := range expected {
		connection.AddRule(rule)
	}
	return connection.Flush()
}

func intact(connection *nftables.Conn, chainList []*nftables.Chain, expected []*nftables.Rule) bool {
	const zero = 0
	var actual []*nftables.Rule
	for _, chain := range chainList {
		rules, err := connection.GetRules(chain.Table, chain)
		if err != nil {
			return false
		}
		actual = append(actual, rules...)
	}
	if len(actual) != len(expected) {
		return false
	}
	for index := zero; index < len(expected); index++ {
		if !bytes.Equal(actual[index].UserData, expected[index].UserData) {
			return false
		}
	}
	return true
}

func (l *Linux) Clear() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.ensure(false)
}

func protocolPort(protocol uint8, port uint16) []expr.Any {
	const register, portOffset, portSize = 1, 2, 2
	portBytes := make([]byte, portSize)
	binary.BigEndian.PutUint16(portBytes, port)
	return []expr.Any{
		&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: register},
		&expr.Cmp{Op: expr.CmpOpEq, Register: register, Data: []byte{protocol}},
		&expr.Payload{DestRegister: register, Base: expr.PayloadBaseTransportHeader, Offset: portOffset, Len: portSize},
		&expr.Cmp{Op: expr.CmpOpEq, Register: register, Data: portBytes},
	}
}

func (l *Linux) rules(target *nftables.Table, dns, filter *nftables.Chain) []*nftables.Rule {
	const register, uidSize, dnsPort, dotPort, httpsPort, forwardPort, zero = 1, 4, 53, 853, 443, 5353, 0
	const redirectMarker, dotMarker, dohMarker = "saveme:redirect", "saveme:dot", "saveme:doh"
	uid := make([]byte, uidSize)
	binary.NativeEndian.PutUint32(uid, l.ForwardUID)
	var dnsRules, filterRules []*nftables.Rule
	protocols := []uint8{unix.IPPROTO_UDP, unix.IPPROTO_TCP}
	for _, protocol := range protocols {

		const forwardMarker = "saveme:forwarder-private"
		redirect := []expr.Any{&expr.Meta{Key: expr.MetaKeySKUID, Register: register}, &expr.Cmp{Op: expr.CmpOpNeq, Register: register, Data: uid}}
		redirect = append(redirect, protocolPort(protocol, dnsPort)...)
		redirect = append(redirect, &expr.Immediate{Register: register, Data: []byte{zero, dnsPort}}, &expr.Redir{RegisterProtoMin: register})
		dnsRules = append(dnsRules, &nftables.Rule{Table: target, Chain: dns, Exprs: redirect, UserData: []byte(redirectMarker)})
		dot := append(protocolPort(protocol, dotPort), &expr.Verdict{Kind: expr.VerdictDrop})
		filterRules = append(filterRules, &nftables.Rule{Table: target, Chain: filter, Exprs: dot, UserData: []byte(dotMarker)})

		private := append(protocolPort(protocol, forwardPort), &expr.Meta{Key: expr.MetaKeySKUID, Register: register}, &expr.Cmp{Op: expr.CmpOpNeq, Register: register, Data: make([]byte, uidSize)}, &expr.Verdict{Kind: expr.VerdictDrop})
		filterRules = append(filterRules, &nftables.Rule{Table: target, Chain: filter, Exprs: private, UserData: []byte(forwardMarker)})
		for _, address := range l.DoH {
			doh := append(addressMatch(address), protocolPort(protocol, httpsPort)...)
			doh = append(doh, &expr.Verdict{Kind: expr.VerdictDrop})
			filterRules = append(filterRules, &nftables.Rule{Table: target, Chain: filter, Exprs: doh, UserData: []byte(dohMarker)})
		}
	}
	return append(dnsRules, filterRules...)
}

func addressMatch(address net.IP) []expr.Any {
	const register, ipv4Offset, ipv6Offset, ipv4Size, ipv6Size = 1, 16, 24, 4, 16
	family, offset, size := byte(unix.NFPROTO_IPV6), uint32(ipv6Offset), uint32(ipv6Size)
	data := address.To16()
	if v4 := address.To4(); v4 != nil {
		family, offset, size, data = unix.NFPROTO_IPV4, ipv4Offset, ipv4Size, v4
	}
	return []expr.Any{
		&expr.Meta{Key: expr.MetaKeyNFPROTO, Register: register},
		&expr.Cmp{Op: expr.CmpOpEq, Register: register, Data: []byte{family}},
		&expr.Payload{DestRegister: register, Base: expr.PayloadBaseNetworkHeader, Offset: offset, Len: size},
		&expr.Cmp{Op: expr.CmpOpEq, Register: register, Data: data},
	}
}

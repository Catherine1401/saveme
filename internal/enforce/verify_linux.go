package enforce

import (
	"bytes"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
	"saveme/internal/core"
)

func (l *Linux) Verify(active bool) error {
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
	if found != active {
		return core.ErrEnforcement
	}
	if !active {
		return nil
	}
	chainList := chains(target)
	actualChains, err := connection.ListChains()
	if err != nil {
		return err
	}
	for _, expected := range chainList {
		matched := false
		for _, actual := range actualChains {
			if actual.Table.Name == target.Name && actual.Table.Family == target.Family && actual.Name == expected.Name {
				matched = actual.Type == expected.Type && actual.Hooknum != nil && actual.Priority != nil && *actual.Hooknum == *expected.Hooknum && *actual.Priority == *expected.Priority
			}
		}
		if !matched {
			return core.ErrEnforcement
		}
	}
	expected := l.rules(target, chainList[dnsIndex], chainList[filterIndex])
	if !intact(connection, chainList, expected) {
		return core.ErrEnforcement
	}
	for _, chain := range chainList {
		actual, err := connection.GetRules(target, chain)
		if err != nil {
			return err
		}
		var wanted []*nftables.Rule
		for _, rule := range expected {
			if rule.Chain.Name == chain.Name {
				wanted = append(wanted, rule)
			}
		}
		if len(actual) != len(wanted) {
			return core.ErrEnforcement
		}
		for index, rule := range actual {
			if !sameExpressions(rule.Exprs, wanted[index].Exprs, byte(target.Family)) {
				return core.ErrEnforcement
			}
		}
	}
	return nil
}

func sameExpressions(actual, expected []expr.Any, family byte) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index, expression := range actual {
		got, err := expr.Marshal(family, expression)
		if err != nil {
			return false
		}
		want, err := expr.Marshal(family, kernelExpression(expected[index]))
		if err != nil || !bytes.Equal(got, want) {
			return false
		}
	}
	return true
}

func kernelExpression(expression expr.Any) expr.Any {
	const zero = 0
	if redirect, ok := expression.(*expr.Redir); ok {
		copy := *redirect
		if copy.RegisterProtoMax == zero {
			copy.RegisterProtoMax = copy.RegisterProtoMin
		}
		if copy.RegisterProtoMin != zero {
			copy.Flags |= unix.NF_NAT_RANGE_PROTO_SPECIFIED
		}
		return &copy
	}
	return expression
}

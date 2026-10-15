package enforce

import (
	"fmt"
	"strings"
)

// Nhóm rule Firewall và comment NRPT dùng để nhận diện cấu hình của saveme.
const firewallGroup = "saveme"

// Danh sách PowerShell dạng @('a','b') cho tham số RemoteAddress.
func psList(values []string) string {
	const open, close, separator, quote = "@(", ")", ",", "'"
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = quote + value + quote
	}
	return open + strings.Join(quoted, separator) + close
}

// Script gỡ rule Firewall và NRPT của saveme rồi xóa cache DNS.
func clearScript() string {
	const format = "$ErrorActionPreference='Stop'\n" +
		"Get-NetFirewallRule -Group '%[1]s' -ErrorAction SilentlyContinue | Remove-NetFirewallRule\n" +
		"Get-DnsClientNrptRule | Where-Object Comment -eq '%[1]s' | ForEach-Object { Remove-DnsClientNrptRule -Name $_.Name -Force }\n" +
		"Clear-DnsClientCache\n"
	return fmt.Sprintf(format, firewallGroup)
}

// Script dựng lại toàn bộ rule: DNS 53 chỉ tới upstream, chặn DoT 853 và DoH 443, NRPT trỏ về resolver cục bộ.
func ensureScript(dnsRanges, doh []string) string {
	const ruleFormat = "New-NetFirewallRule -Group '%s' -DisplayName '%s-%s' -Direction Outbound -Action Block -Protocol %s -RemotePort %d -RemoteAddress %s | Out-Null\n"
	const nrptFormat = "Add-DnsClientNrptRule -Namespace '.' -NameServers '127.0.0.1','::1' -Comment '%s'\nClear-DnsClientCache\n"
	const dnsName, dotName, dohName, dnsPort, dotPort, httpsPort, anyAddress = "dns", "dot", "doh", 53, 853, 443, "Any"
	const udp, tcp, separator = "UDP", "TCP", "-"
	var script strings.Builder
	script.WriteString(clearScript())
	for _, protocol := range []string{udp, tcp} {
		suffix := strings.ToLower(protocol)
		fmt.Fprintf(&script, ruleFormat, firewallGroup, firewallGroup, dnsName+separator+suffix, protocol, dnsPort, psList(dnsRanges))
		fmt.Fprintf(&script, ruleFormat, firewallGroup, firewallGroup, dotName+separator+suffix, protocol, dotPort, anyAddress)
		fmt.Fprintf(&script, ruleFormat, firewallGroup, firewallGroup, dohName+separator+suffix, protocol, httpsPort, psList(doh))
	}
	fmt.Fprintf(&script, nrptFormat, firewallGroup)
	return script.String()
}

// Script in số rule Firewall và số rule NRPT hiện có của saveme.
func checkScript() string {
	const format = "$f=@(Get-NetFirewallRule -Group '%[1]s' -ErrorAction SilentlyContinue).Count\n" +
		"$n=@(Get-DnsClientNrptRule | Where-Object Comment -eq '%[1]s').Count\n" +
		"Write-Output \"$f $n\"\n"
	return fmt.Sprintf(format, firewallGroup)
}

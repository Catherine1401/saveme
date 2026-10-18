package enforce

import (
	"fmt"
	"strings"

	"saveme/internal/core"
)

func (f *Firewall) Verify(active bool) error {
	const expected = "SAVEME_VERIFIED"
	output, err := f.Run(auditScript(active, Complement(f.Upstreams), f.DoH))
	if err != nil {
		return err
	}
	if strings.TrimSpace(output) != expected {
		return core.ErrEnforcement
	}
	return nil
}

func auditScript(active bool, ranges, doh []string) string {
	const group = "saveme"
	const falseValue = "$false"
	const script = `$ErrorActionPreference='Stop'
$rules=@(Get-NetFirewallRule -Group '%s' -ErrorAction SilentlyContinue)
$nrpt=@(Get-DnsClientNrptRule | Where-Object Comment -eq '%s')
$active=%s
if (-not $active) {
 if ($rules.Count -ne 0 -or $nrpt.Count -ne 0) { throw 'Stale saveme rules' }
} else {
 if (@(Get-NetFirewallProfile | Where-Object Enabled -ne 'True').Count) { throw 'Firewall profile disabled' }
 if ($rules.Count -ne 6 -or $nrpt.Count -ne 1) { throw 'Missing saveme enforcement' }
 if (@($nrpt[0].Namespace).Count -ne 1 -or $nrpt[0].Namespace[0] -ne '.' -or @($nrpt[0].NameServers).Count -ne 2 -or (Compare-Object @($nrpt[0].NameServers) @('127.0.0.1','::1'))) { throw 'Invalid NRPT' }
 foreach ($protocol in @('udp','tcp')) {
  foreach ($kind in @('dns','dot','doh')) {
   $rule=@($rules | Where-Object DisplayName -eq ('saveme-'+$kind+'-'+$protocol))
   if ($rule.Count -ne 1 -or $rule[0].Enabled -ne 'True' -or $rule[0].Direction -ne 'Outbound' -or $rule[0].Action -ne 'Block') { throw 'Invalid firewall rule' }
   if ("$($rule[0].Profile)" -ne 'Any') { throw 'Invalid firewall profiles' }
   $application=$rule[0] | Get-NetFirewallApplicationFilter
   $service=$rule[0] | Get-NetFirewallServiceFilter
   if ($application.Program -ne 'Any' -or $service.Service -ne 'Any') { throw 'Restricted application filter' }
   $local=$rule[0] | Get-NetFirewallAddressFilter
   if ("$($local.LocalAddress)" -ne 'Any') { throw 'Restricted local address' }
   $port=$rule[0] | Get-NetFirewallPortFilter
   if ("$($port.LocalPort)" -ne 'Any') { throw 'Restricted local port' }
   $number=if ($protocol -eq 'udp') { 17 } else { 6 }
   if ("$($port.Protocol)" -notin @($protocol.ToUpper(),"$number")) { throw 'Invalid protocol' }
   $wantedPort=if ($kind -eq 'dns') { '53' } elseif ($kind -eq 'dot') { '853' } else { '443' }
   if (@($port.RemotePort).Count -ne 1 -or "$($port.RemotePort)" -ne $wantedPort) { throw 'Invalid port' }
   $addresses=@(($rule[0] | Get-NetFirewallAddressFilter).RemoteAddress | Sort-Object)
   $wanted=if ($kind -eq 'dns') { %s } elseif ($kind -eq 'doh') { %s } else { @('Any') }
   if (Compare-Object $addresses @($wanted | Sort-Object)) { throw 'Invalid address filters' }
  }
 }
 foreach ($path in @('HKLM:\SOFTWARE\Policies\Google\Chrome','HKLM:\SOFTWARE\Policies\Microsoft\Edge','HKLM:\SOFTWARE\Policies\Chromium')) {
  if ((Get-ItemProperty -Path $path -Name DnsOverHttpsMode).DnsOverHttpsMode -ne 'off') { throw 'Invalid DoH policy' }
 }
 $firefox=Get-ItemProperty 'HKLM:\SOFTWARE\Policies\Mozilla\Firefox\DNSOverHTTPS'
 if ($firefox.Enabled -ne 0 -or $firefox.Locked -ne 1) { throw 'Invalid Firefox policy' }
}
Write-Output 'SAVEME_VERIFIED'
`

	value := falseValue
	if active {
		const trueValue = "$true"
		value = trueValue
	}

	return fmt.Sprintf(script, group, group, value, psList(ranges), psList(doh))
}

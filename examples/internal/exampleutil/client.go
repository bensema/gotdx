package exampleutil

import "github.com/bensema/gotdx"

func NewMainClient() *gotdx.Client {
	primary, pool := splitHosts(gotdx.MainHostAddresses())
	return gotdx.New(
		gotdx.WithTCPAddress(primary),
		gotdx.WithTCPAddressPool(pool...),
	)
}

func NewMainClientForHost(host string) *gotdx.Client {
	return gotdx.New(
		gotdx.WithTCPAddress(host),
		gotdx.WithTimeoutSec(6),
	)
}

func NewExClient() *gotdx.Client {
	primary, pool := splitHosts(gotdx.ExHostAddresses())
	return gotdx.New(
		gotdx.WithExTCPAddress(primary),
		gotdx.WithExTCPAddressPool(pool...),
		gotdx.WithTimeoutSec(6),
	)
}

func NewExClientForHost(host string) *gotdx.Client {
	return gotdx.New(
		gotdx.WithExTCPAddress(host),
		gotdx.WithTimeoutSec(6),
	)
}

func NewUnifiedClient() *gotdx.Client {
	mainPrimary, mainPool := splitHosts(gotdx.MainHostAddresses())
	exPrimary, exPool := splitHosts(gotdx.ExHostAddresses())
	return gotdx.New(
		gotdx.WithTCPAddress(mainPrimary),
		gotdx.WithTCPAddressPool(mainPool...),
		gotdx.WithExTCPAddress(exPrimary),
		gotdx.WithExTCPAddressPool(exPool...),
		gotdx.WithTimeoutSec(6),
	)
}

func NewMACClient() *gotdx.Client {
	primary, pool := splitHosts(gotdx.MACHostAddresses())
	return gotdx.NewMAC(
		gotdx.WithMacTCPAddress(primary),
		gotdx.WithMacTCPAddressPool(pool...),
		gotdx.WithTimeoutSec(6),
	)
}

func NewMACExClient() *gotdx.Client {
	primary, pool := splitHosts(gotdx.MACExHostAddresses())
	return gotdx.NewMACEx(
		gotdx.WithMacExTCPAddress(primary),
		gotdx.WithMacExTCPAddressPool(pool...),
		gotdx.WithTimeoutSec(6),
	)
}

func MainHosts() []string {
	return gotdx.MainHostAddresses()
}

func ExHosts() []string {
	return gotdx.ExHostAddresses()
}

func MACHosts() []string {
	return gotdx.MACHostAddresses()
}

func MACExHosts() []string {
	return gotdx.MACExHostAddresses()
}

func splitHosts(hosts []string) (string, []string) {
	if len(hosts) == 0 {
		return "", nil
	}
	return hosts[0], append([]string(nil), hosts[1:]...)
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func PreviewHex(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

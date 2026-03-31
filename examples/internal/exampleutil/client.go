package exampleutil

import "github.com/bensema/gotdx"

var mainHosts = []string{
	"124.71.187.122:7709",
	"124.71.187.72:7709",
	"124.70.133.119:7709",
	"123.60.73.44:7709",
	"123.60.84.66:7709",
}

var exHosts = []string{
	"112.74.214.43:7727",
	"120.25.218.6:7727",
	"43.139.173.246:7727",
	"159.75.90.107:7727",
	"106.52.170.195:7727",
	"175.24.47.69:7727",
	"139.9.191.175:7727",
	"150.158.9.199:7727",
}

func NewMainClient() *gotdx.Client {
	return gotdx.New(
		gotdx.WithTCPAddress(mainHosts[0]),
		gotdx.WithTCPAddressPool(mainHosts[1:]...),
	)
}

func NewMainClientForHost(host string) *gotdx.Client {
	return gotdx.New(
		gotdx.WithTCPAddress(host),
		gotdx.WithTimeoutSec(6),
	)
}

func NewExClient() *gotdx.Client {
	return gotdx.New(
		gotdx.WithExTCPAddress(exHosts[0]),
		gotdx.WithExTCPAddressPool(exHosts[1:]...),
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
	return gotdx.New(
		gotdx.WithTCPAddress(mainHosts[0]),
		gotdx.WithTCPAddressPool(mainHosts[1:]...),
		gotdx.WithExTCPAddress(exHosts[0]),
		gotdx.WithExTCPAddressPool(exHosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
}

func MainHosts() []string {
	return append([]string(nil), mainHosts...)
}

func ExHosts() []string {
	return append([]string(nil), exHosts...)
}

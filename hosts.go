package gotdx

import (
	"errors"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
)

// HostInfo describes a built-in TDX server entry migrated from opentdx.
type HostInfo struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

// Address returns the host:port form used by the client connection layer.
func (h HostInfo) Address() string {
	return net.JoinHostPort(h.IP, strconv.Itoa(h.Port))
}

// HostProbeResult records a single TCP probe result.
type HostProbeResult struct {
	Name      string        `json:"name"`
	IP        string        `json:"ip"`
	Port      int           `json:"port"`
	Address   string        `json:"address"`
	Latency   time.Duration `json:"latency"`
	Reachable bool          `json:"reachable"`
	Error     string        `json:"error,omitempty"`
}

// ErrNoReachableHosts indicates that every probed host failed the TCP dial.
var ErrNoReachableHosts = errors.New("no reachable hosts")

var mainHostList = []HostInfo{
	{Name: "通达信深圳双线主站1", IP: "110.41.147.114", Port: 7709},
	{Name: "通达信深圳双线主站2", IP: "110.41.2.72", Port: 7709},
	{Name: "通达信深圳双线主站3", IP: "110.41.4.4", Port: 7709},
	{Name: "通达信深圳双线主站4", IP: "47.113.94.204", Port: 7709},
	{Name: "通达信深圳双线主站5", IP: "8.129.174.169", Port: 7709},
	{Name: "通达信深圳双线主站6", IP: "110.41.154.219", Port: 7709},
	{Name: "通达信上海双线主站1", IP: "124.70.176.52", Port: 7709},
	{Name: "通达信上海双线主站2", IP: "47.100.236.28", Port: 7709},
	{Name: "通达信上海双线主站3", IP: "123.60.186.45", Port: 7709},
	{Name: "通达信上海双线主站4", IP: "123.60.164.122", Port: 7709},
	{Name: "通达信上海双线主站5", IP: "47.116.105.28", Port: 7709},
	{Name: "通达信上海双线主站6", IP: "124.70.199.56", Port: 7709},
	{Name: "通达信北京双线主站1", IP: "121.36.54.217", Port: 7709},
	{Name: "通达信北京双线主站2", IP: "121.36.81.195", Port: 7709},
	{Name: "通达信北京双线主站3", IP: "123.249.15.60", Port: 7709},
	{Name: "通达信广州双线主站1", IP: "124.71.85.110", Port: 7709},
	{Name: "通达信广州双线主站2", IP: "139.9.51.18", Port: 7709},
	{Name: "通达信广州双线主站3", IP: "139.159.239.163", Port: 7709},
	{Name: "通达信上海双线主站7", IP: "106.14.201.131", Port: 7709},
	{Name: "通达信上海双线主站8", IP: "106.14.190.242", Port: 7709},
	{Name: "通达信上海双线主站9", IP: "121.36.225.169", Port: 7709},
	{Name: "通达信上海双线主站10", IP: "123.60.70.228", Port: 7709},
	{Name: "通达信上海双线主站11", IP: "123.60.73.44", Port: 7709},
	{Name: "通达信上海双线主站12", IP: "124.70.133.119", Port: 7709},
	{Name: "通达信上海双线主站13", IP: "124.71.187.72", Port: 7709},
	{Name: "通达信上海双线主站14", IP: "124.71.187.122", Port: 7709},
	{Name: "通达信武汉电信主站1", IP: "119.97.185.59", Port: 7709},
	{Name: "通达信深圳双线主站7", IP: "47.107.64.168", Port: 7709},
	{Name: "通达信北京双线主站4", IP: "124.70.75.113", Port: 7709},
	{Name: "通达信广州双线主站4", IP: "124.71.9.153", Port: 7709},
	{Name: "通达信上海双线主站15", IP: "123.60.84.66", Port: 7709},
	{Name: "通达信深圳双线主站8", IP: "47.107.228.47", Port: 7719},
	{Name: "通达信北京双线主站5", IP: "120.46.186.223", Port: 7709},
	{Name: "通达信北京双线主站6", IP: "124.70.22.210", Port: 7709},
	{Name: "通达信北京双线主站7", IP: "139.9.133.247", Port: 7709},
	{Name: "通达信广州双线主站5", IP: "116.205.163.254", Port: 7709},
	{Name: "通达信广州双线主站6", IP: "116.205.171.132", Port: 7709},
	{Name: "通达信广州双线主站7", IP: "116.205.183.150", Port: 7709},
}

var brokerHostList = []HostInfo{
	{Name: "上证云成都电信一", IP: "218.6.170.47", Port: 7709},
	{Name: "上证云北京联通一", IP: "123.125.108.14", Port: 7709},
	{Name: "上海电信主站Z1", IP: "180.153.18.170", Port: 7709},
	{Name: "上海电信主站Z80", IP: "180.153.18.172", Port: 80},
	{Name: "北京联通主站Z80", IP: "202.108.253.139", Port: 80},
	{Name: "杭州电信主站J1", IP: "60.191.117.167", Port: 7709},
	{Name: "杭州电信主站J2", IP: "115.238.56.198", Port: 7709},
	{Name: "杭州电信主站J3", IP: "218.75.126.9", Port: 7709},
	{Name: "杭州电信主站J4", IP: "115.238.90.165", Port: 7709},
	{Name: "安信", IP: "59.36.5.11", Port: 7709},
	{Name: "广发", IP: "119.29.19.242", Port: 7709},
	{Name: "广发", IP: "183.60.224.177", Port: 7709},
	{Name: "广发", IP: "183.60.224.178", Port: 7709},
	{Name: "国泰君安", IP: "117.34.114.13", Port: 7709},
	{Name: "国泰君安", IP: "117.34.114.14", Port: 7709},
	{Name: "国泰君安", IP: "117.34.114.15", Port: 7709},
	{Name: "国泰君安", IP: "117.34.114.16", Port: 7709},
	{Name: "国泰君安", IP: "117.34.114.17", Port: 7709},
	{Name: "国泰君安", IP: "117.34.114.18", Port: 7709},
	{Name: "国泰君安", IP: "117.34.114.20", Port: 7709},
	{Name: "国泰君安", IP: "117.34.114.27", Port: 7709},
	{Name: "国泰君安", IP: "117.34.114.30", Port: 7709},
	{Name: "国泰君安", IP: "117.34.114.31", Port: 7709},
	{Name: "国信", IP: "182.131.3.252", Port: 7709},
	{Name: "国信", IP: "58.63.254.247", Port: 7709},
	{Name: "海通", IP: "123.125.108.90", Port: 7709},
	{Name: "海通", IP: "175.6.5.153", Port: 7709},
	{Name: "海通", IP: "182.118.47.151", Port: 7709},
	{Name: "海通", IP: "182.131.3.245", Port: 7709},
	{Name: "海通", IP: "202.100.166.27", Port: 7709},
	{Name: "海通", IP: "58.63.254.191", Port: 7709},
	{Name: "海通", IP: "58.63.254.217", Port: 7709},
	{Name: "华林", IP: "202.100.166.21", Port: 7709},
	{Name: "华林", IP: "202.96.138.90", Port: 7709},
}

var exHostList = []HostInfo{
	{Name: "扩展市场深圳双线1", IP: "112.74.214.43", Port: 7727},
	{Name: "扩展市场深圳双线2", IP: "120.25.218.6", Port: 7727},
	{Name: "扩展市场深圳双线3", IP: "43.139.173.246", Port: 7727},
	{Name: "扩展市场深圳双线4", IP: "159.75.90.107", Port: 7727},
	{Name: "扩展市场深圳双线5", IP: "106.52.170.195", Port: 7727},
	{Name: "扩展市场广州双线3", IP: "175.24.47.69", Port: 7727},
	{Name: "扩展市场上海双线7", IP: "139.9.191.175", Port: 7727},
	{Name: "扩展市场上海双线1", IP: "150.158.9.199", Port: 7727},
	{Name: "扩展市场上海双线2", IP: "150.158.20.127", Port: 7727},
	{Name: "扩展市场上海双线3", IP: "49.235.119.116", Port: 7727},
	{Name: "扩展市场上海双线4", IP: "49.234.13.160", Port: 7727},
	{Name: "扩展市场广州双线1", IP: "116.205.143.214", Port: 7727},
	{Name: "扩展市场广州双线2", IP: "124.71.223.19", Port: 7727},
	{Name: "扩展市场广州双线3", IP: "123.60.173.210", Port: 7727},
	{Name: "扩展市场上海双线5", IP: "113.45.175.47", Port: 7727},
	{Name: "扩展市场上海双线6", IP: "118.89.69.202", Port: 7727},
}

var macHostList = []HostInfo{
	{Name: "行情主站1", IP: "121.36.248.138", Port: 7709},
	{Name: "行情主站2", IP: "123.60.47.136", Port: 7709},
	{Name: "行情主站3", IP: "121.37.207.165", Port: 7709},
}

var macExHostList = []HostInfo{
	{Name: "扩展行情1", IP: "116.205.135.205", Port: 7727},
	{Name: "扩展行情2", IP: "121.37.232.167", Port: 7727},
}

// MainHosts returns the built-in main quote servers from opentdx.
func MainHosts() []HostInfo {
	return cloneHosts(mainHostList)
}

// BrokerHosts returns the built-in broker quote servers from opentdx.
func BrokerHosts() []HostInfo {
	return cloneHosts(brokerHostList)
}

// ExHosts returns the built-in extended-market servers from opentdx.
func ExHosts() []HostInfo {
	return cloneHosts(exHostList)
}

// MACHosts returns the built-in MAC quote servers from opentdx.
func MACHosts() []HostInfo {
	return cloneHosts(macHostList)
}

// MACExHosts returns the built-in MAC extended-market servers from opentdx.
func MACExHosts() []HostInfo {
	return cloneHosts(macExHostList)
}

// MainHostAddresses returns the built-in main quote host:port list.
func MainHostAddresses() []string {
	return hostAddresses(mainHostList)
}

// BrokerHostAddresses returns the built-in broker host:port list.
func BrokerHostAddresses() []string {
	return hostAddresses(brokerHostList)
}

// ExHostAddresses returns the built-in extended-market host:port list.
func ExHostAddresses() []string {
	return hostAddresses(exHostList)
}

// MACHostAddresses returns the built-in MAC quote host:port list.
func MACHostAddresses() []string {
	return hostAddresses(macHostList)
}

// MACExHostAddresses returns the built-in MAC extended-market host:port list.
func MACExHostAddresses() []string {
	return hostAddresses(macExHostList)
}

// ProbeHosts concurrently dials the supplied built-in host entries and returns
// the results sorted by reachability and latency.
func ProbeHosts(hosts []HostInfo, timeout time.Duration) []HostProbeResult {
	results := probeAddressesWithDial(hostAddresses(hosts), timeout, net.DialTimeout)
	names := make(map[string]HostInfo, len(hosts))
	for _, host := range hosts {
		names[host.Address()] = host
	}
	for i := range results {
		if host, ok := names[results[i].Address]; ok {
			results[i].Name = host.Name
			results[i].IP = host.IP
			results[i].Port = host.Port
		}
	}
	return results
}

// ProbeAddresses concurrently dials the supplied host:port list and returns the
// results sorted by reachability and latency.
func ProbeAddresses(addresses []string, timeout time.Duration) []HostProbeResult {
	return probeAddressesWithDial(addresses, timeout, net.DialTimeout)
}

// FastestHost returns the fastest reachable built-in host entry.
func FastestHost(hosts []HostInfo, timeout time.Duration) (HostProbeResult, error) {
	return fastestProbeResult(ProbeHosts(hosts, timeout))
}

// FastestAddress returns the fastest reachable address from the supplied list.
func FastestAddress(addresses []string, timeout time.Duration) (HostProbeResult, error) {
	return fastestProbeResult(ProbeAddresses(addresses, timeout))
}

func cloneHosts(hosts []HostInfo) []HostInfo {
	return append([]HostInfo(nil), hosts...)
}

func hostAddresses(hosts []HostInfo) []string {
	addresses := make([]string, 0, len(hosts))
	for _, host := range hosts {
		addresses = append(addresses, host.Address())
	}
	return addresses
}

func defaultAddressAndPool(addresses []string, fallback string) (string, []string) {
	if len(addresses) == 0 {
		return fallback, nil
	}
	return addresses[0], append([]string(nil), addresses[1:]...)
}

func probeAddressesWithDial(addresses []string, timeout time.Duration, dial func(network, address string, timeout time.Duration) (net.Conn, error)) []HostProbeResult {
	if timeout <= 0 {
		timeout = time.Duration(_defaultTimeoutSec) * time.Second
	}

	results := make([]HostProbeResult, 0, len(addresses))
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	for _, address := range addresses {
		address := address
		wg.Add(1)
		go func() {
			defer wg.Done()

			result := hostProbeFromAddress(address)
			started := time.Now()
			conn, err := dial("tcp", address, timeout)
			if err != nil {
				result.Error = err.Error()
			} else {
				result.Reachable = true
				result.Latency = time.Since(started)
				_ = conn.Close()
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		if results[i].Reachable != results[j].Reachable {
			return results[i].Reachable
		}
		if results[i].Reachable && results[j].Reachable && results[i].Latency != results[j].Latency {
			return results[i].Latency < results[j].Latency
		}
		if results[i].Name != results[j].Name {
			return results[i].Name < results[j].Name
		}
		return results[i].Address < results[j].Address
	})

	return results
}

func fastestProbeResult(results []HostProbeResult) (HostProbeResult, error) {
	for _, result := range results {
		if result.Reachable {
			return result, nil
		}
	}
	return HostProbeResult{}, ErrNoReachableHosts
}

func hostProbeFromAddress(address string) HostProbeResult {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return HostProbeResult{
			Address: address,
			Error:   err.Error(),
		}
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return HostProbeResult{
			IP:      host,
			Address: address,
			Error:   err.Error(),
		}
	}
	return HostProbeResult{
		IP:      host,
		Port:    portNum,
		Address: address,
	}
}

package gotdx

import (
	"errors"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
)

// HostInfo 表示一条内置通达信行情服务器条目，包含站点名称、IP 地址与端口。
//
// 该结构体为纯值类型，零值对应名称、IP 为空串、端口为 0；各站点列表函数返回的是内置
// 列表的浅拷贝，调用方修改返回切片或其中条目不会影响内置数据。
type HostInfo struct {
	Name string `json:"name"` // 站点显示名称，用于界面展示与探测结果标注
	IP   string `json:"ip"`   // 服务器 IPv4 或主机名，连接层直接用于拨号
	Port int    `json:"port"` // 服务器端口号，取值范围 0 至 65535
}

// Address 返回客户端连接层使用的 `host:port` 形式地址，由 `net.JoinHostPort` 拼接，
// 自动处理 IPv6 场景所需的方括号包裹。
func (h HostInfo) Address() string {
	return net.JoinHostPort(h.IP, strconv.Itoa(h.Port))
}

// HostProbeResult 记录一次 TCP 拨号探测的结果。
//
// 当 `Reachable` 为 `true` 时 `Latency` 为本次拨号测得的往返耗时；当 `Reachable` 为 `false`
// 时 `Latency` 无意义，`Error` 保存拨号失败原因的文本描述。零值表示未探测或不可达。
type HostProbeResult struct {
	Name      string        `json:"name"`            // 站点显示名称，地址无法回填名称时为空串
	IP        string        `json:"ip"`              // 服务器 IP 地址或主机名
	Port      int           `json:"port"`            // 服务器端口号；端口解析失败时保持 0
	Address   string        `json:"address"`         // `host:port` 形式地址，原样来自探测输入
	Latency   time.Duration `json:"latency"`         // 拨号往返耗时，仅 `Reachable` 为 `true` 时有效
	Reachable bool          `json:"reachable"`       // 是否 TCP 拨号成功
	Error     string        `json:"error,omitempty"` // 拨号或地址解析失败原因，成功时为空串
}

// ErrNoReachableHosts 表示本次探测的所有主机 TCP 拨号均失败，没有任何可达节点；
// 由 `FastestHost`、`FastestAddress` 在筛选最快主机时返回。
var ErrNoReachableHosts = errors.New("no reachable hosts")

var mainHostList = []HostInfo{
	{Name: "通达信深圳双线主站1", IP: "110.41.147.114", Port: 7709},
	{Name: "通达信深圳双线主站2", IP: "110.41.2.72", Port: 7709},
	{Name: "通达信上海双线主站4", IP: "123.60.164.122", Port: 7709},
	{Name: "通达信上海双线主站6", IP: "124.70.199.56", Port: 7709},
	{Name: "通达信上海双线主站9", IP: "121.36.225.169", Port: 7709},
	{Name: "通达信上海双线主站10", IP: "123.60.70.228", Port: 7709},
	{Name: "通达信上海双线主站11", IP: "123.60.73.44", Port: 7709},
	{Name: "通达信上海双线主站12", IP: "124.70.133.119", Port: 7709},
	{Name: "通达信上海双线主站13", IP: "124.71.187.72", Port: 7709},
	{Name: "通达信上海双线主站14", IP: "124.71.187.122", Port: 7709},
	{Name: "通达信武汉电信主站1", IP: "119.97.185.59", Port: 7709},
	{Name: "通达信广州双线主站4", IP: "124.71.9.153", Port: 7709},
	{Name: "通达信上海双线主站15", IP: "123.60.84.66", Port: 7709},
	{Name: "通达信广州双线主站5", IP: "116.205.163.254", Port: 7709},
	{Name: "通达信广州双线主站6", IP: "116.205.171.132", Port: 7709},
	{Name: "通达信广州双线主站7", IP: "116.205.183.150", Port: 7709},
	{Name: "通达信上海腾讯云1", IP: "122.51.120.217", Port: 7709},
	{Name: "通达信上海腾讯云2", IP: "111.229.247.189", Port: 7709},
	{Name: "通达信上海腾讯云3", IP: "122.51.232.182", Port: 7709},
	{Name: "通达信上海腾讯云4", IP: "118.25.98.114", Port: 7709},
	{Name: "通达信上海腾讯云5", IP: "124.223.163.242", Port: 7709},
	{Name: "通达信上海腾讯云6", IP: "150.158.160.2", Port: 7709},
	{Name: "通达信上海腾讯云7", IP: "101.35.121.35", Port: 7709},
	{Name: "通达信上海腾讯云8", IP: "111.231.113.208", Port: 7709},
	{Name: "通达信北京腾讯云1", IP: "62.234.50.143", Port: 7709},
	{Name: "通达信北京腾讯云2", IP: "81.70.151.186", Port: 7709},
	{Name: "通达信北京腾讯云3", IP: "101.42.240.54", Port: 7709},
	{Name: "通达信北京腾讯云4", IP: "101.43.159.194", Port: 7709},
	{Name: "通达信北京腾讯云5", IP: "120.53.8.251", Port: 7709},
	{Name: "通达信北京腾讯云6", IP: "152.136.191.169", Port: 7709},
	{Name: "通达信北京腾讯云7", IP: "49.232.15.141", Port: 7709},
	{Name: "通达信北京腾讯云8", IP: "82.156.174.84", Port: 7709},
	{Name: "通达信北京腾讯云9", IP: "101.42.164.241", Port: 7709},
	{Name: "通达信广州腾讯云1", IP: "111.230.186.52", Port: 7709},
	{Name: "通达信广州腾讯云2", IP: "101.33.225.16", Port: 7709},
	{Name: "通达信广州腾讯云3", IP: "175.178.112.197", Port: 7709},
	{Name: "通达信广州腾讯云4", IP: "175.178.128.227", Port: 7709},
	{Name: "通达信广州腾讯云5", IP: "43.139.95.83", Port: 7709},
	{Name: "通达信广州腾讯云6", IP: "159.75.29.111", Port: 7709},
	{Name: "通达信广州腾讯云7", IP: "43.139.18.171", Port: 7709},
	{Name: "通达信广州腾讯云8", IP: "81.71.32.47", Port: 7709},
	{Name: "通达信广州腾讯云9", IP: "129.204.230.128", Port: 7709},
	{Name: "通达信杭州联通主站2", IP: "60.12.136.250", Port: 7709},
	{Name: "通达信北京双线主站8", IP: "82.156.214.79", Port: 7709},
	{Name: "通达信深圳双线主站9", IP: "43.136.95.205", Port: 7709},
	{Name: "通达信深圳双线主站10", IP: "43.136.96.138", Port: 7709},
	{Name: "通达信深圳双线主站11", IP: "101.33.244.209", Port: 7709},
	{Name: "通达信深圳双线主站12", IP: "106.53.63.234", Port: 7709},
	{Name: "通达信深圳双线主站13", IP: "119.91.226.89", Port: 7709},
	{Name: "通达信上海双线主站16", IP: "121.37.183.82", Port: 7709},
	{Name: "通达信深圳双线主站14", IP: "110.41.174.169", Port: 7709},
}

var brokerHostList = []HostInfo{
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
	{Name: "华林", IP: "220.178.55.71", Port: 7709},
	{Name: "华林", IP: "220.178.55.86", Port: 7709},
	{Name: "华林", IP: "218.106.92.182", Port: 7709},
	{Name: "华林", IP: "218.106.92.183", Port: 7709},
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
	{Name: "扩展行情3", IP: "112.74.214.43", Port: 7727},
	{Name: "扩展行情4", IP: "120.25.218.6", Port: 7727},
	{Name: "扩展行情5", IP: "43.139.173.246", Port: 7727},
	{Name: "扩展行情6", IP: "159.75.90.107", Port: 7727},
	{Name: "扩展行情7", IP: "106.52.170.195", Port: 7727},
	{Name: "扩展行情8", IP: "175.24.47.69", Port: 7727},
	{Name: "扩展行情9", IP: "139.9.191.175", Port: 7727},
	{Name: "扩展行情10", IP: "150.158.9.199", Port: 7727},
	{Name: "扩展行情11", IP: "150.158.20.127", Port: 7727},
	{Name: "扩展行情12", IP: "49.235.119.116", Port: 7727},
	{Name: "扩展行情13", IP: "49.234.13.160", Port: 7727},
	{Name: "扩展行情14", IP: "116.205.143.214", Port: 7727},
	{Name: "扩展行情15", IP: "124.71.223.19", Port: 7727},
	{Name: "扩展行情16", IP: "123.60.173.210", Port: 7727},
	{Name: "扩展行情17", IP: "113.45.175.47", Port: 7727},
	{Name: "扩展行情18", IP: "118.89.69.202", Port: 7727},
}

var icfqsHostList = []HostInfo{
	{Name: "华为均衡上海1", IP: "119.3.157.89", Port: 7615},
	{Name: "华为均衡广州1", IP: "139.9.211.159", Port: 7615},
	{Name: "华为均衡广州2", IP: "121.37.193.4", Port: 7615},
	{Name: "华为均衡广州3", IP: "124.71.56.161", Port: 7615},
	{Name: "华为均衡上海2", IP: "123.60.69.160", Port: 7615},
	{Name: "华为均衡上海3", IP: "123.60.149.213", Port: 7615},
	{Name: "腾讯上海均衡1", IP: "118.25.106.154", Port: 7615},
	{Name: "腾讯广州均衡1", IP: "129.204.254.13", Port: 7615},
	{Name: "腾讯广州均衡2", IP: "159.75.115.35", Port: 7615},
	{Name: "腾讯北京均衡", IP: "82.157.190.225", Port: 7615},
	{Name: "华为均衡5", IP: "121.36.192.253", Port: 7615},
	{Name: "华为广州均衡5", IP: "124.71.105.217", Port: 7615},
}

// MainHosts 返回内置主行情服务器列表的浅拷贝；该组站点用于主站 A 股行情协议，默认端口
// 7709，调用方修改返回切片不会影响内置列表。
func MainHosts() []HostInfo {
	return cloneHosts(mainHostList)
}

// BrokerHosts 返回内置券商行情服务器列表的浅拷贝；该组站点为券商与上证云接入地址，
// 调用方修改返回切片不会影响内置列表。
func BrokerHosts() []HostInfo {
	return cloneHosts(brokerHostList)
}

// ExHosts 返回内置扩展行情服务器列表的浅拷贝；该组站点用于期货、港股、美股等扩展行情
// 协议，默认端口 7727，调用方修改返回切片不会影响内置列表。
func ExHosts() []HostInfo {
	return cloneHosts(exHostList)
}

// MACHosts 返回内置 MAC 主行情服务器列表的浅拷贝；该组站点用于 MAC 主站协议，
// 调用方修改返回切片不会影响内置列表。
func MACHosts() []HostInfo {
	return cloneHosts(macHostList)
}

// MACExHosts 返回内置 MAC 扩展行情服务器列表的浅拷贝；该组站点用于 MAC 扩展行情协议，
// 调用方修改返回切片不会影响内置列表。
func MACExHosts() []HostInfo {
	return cloneHosts(macExHostList)
}

// ICFQSHosts 返回内置 ICFQS 资讯行情服务器列表的浅拷贝；该组站点用于资讯行情协议，
// 默认端口 7615，调用方修改返回切片不会影响内置列表。
func ICFQSHosts() []HostInfo {
	return cloneHosts(icfqsHostList)
}

// MainHostAddresses 返回内置主行情服务器的 `host:port` 地址列表副本；列表顺序与 `MainHosts`
// 一致，返回切片可由调用方自由修改。
func MainHostAddresses() []string {
	return hostAddresses(mainHostList)
}

// BrokerHostAddresses 返回内置券商服务器的 `host:port` 地址列表副本；列表顺序与
// `BrokerHosts` 一致，返回切片可由调用方自由修改。
func BrokerHostAddresses() []string {
	return hostAddresses(brokerHostList)
}

// ExHostAddresses 返回内置扩展行情服务器的 `host:port` 地址列表副本；列表顺序与 `ExHosts`
// 一致，返回切片可由调用方自由修改。
func ExHostAddresses() []string {
	return hostAddresses(exHostList)
}

// MACHostAddresses 返回内置 MAC 主行情服务器的 `host:port` 地址列表副本；列表顺序与
// `MACHosts` 一致，返回切片可由调用方自由修改。
func MACHostAddresses() []string {
	return hostAddresses(macHostList)
}

// MACExHostAddresses 返回内置 MAC 扩展行情服务器的 `host:port` 地址列表副本；列表顺序与
// `MACExHosts` 一致，返回切片可由调用方自由修改。
func MACExHostAddresses() []string {
	return hostAddresses(macExHostList)
}

// ICFQSHostAddresses 返回内置 ICFQS 资讯行情服务器的 `host:port` 地址列表副本；列表顺序与
// `ICFQSHosts` 一致，返回切片可由调用方自由修改。
func ICFQSHostAddresses() []string {
	return hostAddresses(icfqsHostList)
}

// ProbeHosts 并发对给定主机条目逐一发起 TCP 拨号探测，返回探测结果切片。
//
// `hosts` 为待探测主机条目；`timeout` 为单次拨号超时时间，传非正数时回退到内置默认超时
// （8 秒）。返回结果先按可达性排序（可达条目在前），同为可达时再按往返延迟从低到高排序，
// 延迟相同按名称、地址字典序兜底；输入为空切片时返回空切片。探测不返回错误，失败原因记录
// 在各结果条目的 `Error` 字段中。
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

// ProbeAddresses 并发对给定 `host:port` 地址列表逐一发起 TCP 拨号探测，返回探测结果切片。
//
// `addresses` 为待探测地址，格式由 `net.SplitHostPort` 解析；`timeout` 为单次拨号超时时间，
// 传非正数时回退到内置默认超时（8 秒）。排序规则同 `ProbeHosts`；输入为空切片时返回空切片。
// 探测不返回错误，失败原因记录在各结果条目的 `Error` 字段中。
func ProbeAddresses(addresses []string, timeout time.Duration) []HostProbeResult {
	return probeAddressesWithDial(addresses, timeout, net.DialTimeout)
}

// FastestHost 返回探测结果中延迟最低的可达主机条目。
//
// `hosts` 与 `timeout` 的含义同 `ProbeHosts`；探测完成后取排序结果里第一个可达条目。全部主机
// 不可达时返回零值 `HostProbeResult` 与 `ErrNoReachableHosts`。
func FastestHost(hosts []HostInfo, timeout time.Duration) (HostProbeResult, error) {
	return fastestProbeResult(ProbeHosts(hosts, timeout))
}

// FastestAddress 返回给定地址列表中延迟最低的可达地址。
//
// `addresses` 与 `timeout` 的含义同 `ProbeAddresses`。全部地址不可达时返回零值
// `HostProbeResult` 与 `ErrNoReachableHosts`。
func FastestAddress(addresses []string, timeout time.Duration) (HostProbeResult, error) {
	return fastestProbeResult(ProbeAddresses(addresses, timeout))
}

// cloneHosts 返回主机切片的浅拷贝；元素为值类型，拷贝后调用方增删或修改切片长度不会影响
// 内置列表。
//
// `hosts` 为待拷贝主机切片；输入为 `nil` 时返回非 `nil` 的空切片。
func cloneHosts(hosts []HostInfo) []HostInfo {
	return append([]HostInfo(nil), hosts...)
}

// hostAddresses 将主机条目列表转换为 `host:port` 地址切片，顺序与输入一致；输入为空时
// 返回非 `nil` 的空切片。
func hostAddresses(hosts []HostInfo) []string {
	addresses := make([]string, 0, len(hosts))
	for _, host := range hosts {
		addresses = append(addresses, host.Address())
	}
	return addresses
}

// defaultAddressAndPool 从地址列表中拆分出主地址与候选池：首个地址作为主地址，其余地址
// 作为候选池。
//
// `addresses` 为输入地址列表；`fallback` 为列表为空时使用的兜底主地址。列表为空时返回
// `fallback` 与 `nil` 候选池；非空时候选池为输入尾部地址的浅拷贝。
func defaultAddressAndPool(addresses []string, fallback string) (string, []string) {
	if len(addresses) == 0 {
		return fallback, nil
	}
	return addresses[0], append([]string(nil), addresses[1:]...)
}

// probeConcurrencyWindow 为探活并发窗口上限：同一时刻最多允许该数量的拨号在途，
// 超过窗口的候选滚动排队。避免大池全量并发拨号（例如 130+ 候选同时超时）在短时间
// 内占满本机网络栈，造成多入口同进程连续建池时的拥塞耦合；窗口内滚动调度下，全部
// 候选的最坏判负时间约为候选数除以窗口向上取整后乘以单次拨号超时。
const probeConcurrencyWindow = 32

// slowProbeTimeout 为慢路径兜底探活的单次拨号超时：快路径已用 `fastDialTimeout`
// 排除偶发秒级慢样本，慢路径只需更短超时快速判定全灭并缩短超时连接驻留时长
// （大池全灭时大量超时连接在途会瞬时拥塞本机网络栈，影响同进程其他入口建池）。
const slowProbeTimeout = 300 * time.Millisecond

// probeAddressesWithDial 受限并发拨号探测地址列表，组装排序后的探测结果。
//
// `addresses` 为待探测 `host:port` 地址；`timeout` 为拨号超时，非正数回退默认 8 秒；`dial`
// 为可替换的拨号函数，便于测试时注入假拨号实现。拨号以 `probeConcurrencyWindow` 为并发
// 窗口滚动调度，各 `goroutine` 先通过互斥锁收集结果，探测结束后统一按可达性、延迟、名称、
// 地址排序返回。
func probeAddressesWithDial(addresses []string, timeout time.Duration, dial func(network, address string, timeout time.Duration) (net.Conn, error)) []HostProbeResult {
	if timeout <= 0 {
		timeout = time.Duration(_defaultTimeoutSec) * time.Second
	}

	results := make([]HostProbeResult, 0, len(addresses))
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	sem := make(chan struct{}, probeConcurrencyWindow)
	for _, address := range addresses {
		// 循环变量在闭包内按当前取值捕获，避免所有 `goroutine` 读到同一地址。
		address := address
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 滚动窗口限制在途拨号数；拨号完成（无论成败）即释放，下一个候选续接。
			sem <- struct{}{}
			defer func() { <-sem }()

			result := hostProbeFromAddress(address)
			started := time.Now()
			conn, err := dial("tcp", address, timeout)
			if err != nil {
				result.Error = err.Error()
			} else {
				result.Reachable = true
				result.Latency = time.Since(started)
				// 拨号仅用于探测可达性，拿到连接后立即关闭，不保留探测连接。
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

// fastestProbeResult 在已按可达性与延迟排序的探测结果中取第一个可达条目；没有可达条目时
// 返回零值结果与 `ErrNoReachableHosts`。
//
// `results` 为已排序的探测结果切片；切片为空或全部不可达时返回零值结果与错误。
func fastestProbeResult(results []HostProbeResult) (HostProbeResult, error) {
	for _, result := range results {
		if result.Reachable {
			return result, nil
		}
	}
	return HostProbeResult{}, ErrNoReachableHosts
}

// hostProbeFromAddress 预先从 `host:port` 地址解析出 IP 与端口，作为探测结果的初始状态。
//
// `address` 为待解析地址；格式非法或端口不是十进制数字时，把解析错误记录到结果的 `Error`
// 字段，拨号阶段据此直接标记不可达，不会重复报错。端口解析失败时 `Port` 保持 0。
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

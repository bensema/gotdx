// Command host_probe 演示对内置主站、扩展站、MAC 主站与 MAC 扩展站四类服务器逐一做 TCP 探测，
// 打印可达性与延迟；最后通过自动选最快主机接口选出延迟最低的主站。
//
// 运行示例：`go run ./examples/host_probe`
package main

import (
	"log"
	"time"

	"github.com/bensema/gotdx"
)

func main() {
	// 分别探测主站、扩展站、MAC 主站、MAC 扩展站四类内置服务器；每组最多打印前 5 个探测结果。
	logProbe("main", gotdx.MainHosts(), 5)
	logProbe("ex", gotdx.ExHosts(), 5)
	logProbe("mac", gotdx.MACHosts(), 5)
	logProbe("mac-ex", gotdx.MACExHosts(), 5)

	// 构造客户端并按 `FastestHost` 选出延迟最低的主站；拨号与读写超时固定为 6 秒。
	// 全部候选地址不可达时 `FastestHost` 返回 `ErrNoReachableHosts`，此处降级为日志提示后退出。
	client := gotdx.New(gotdx.WithTimeoutSec(6))
	fastest, err := client.FastestHost()
	if err != nil {
		log.Printf("fastest main host unavailable: %v", err)
		return
	}
	log.Printf("fastest main host: %s %s latency=%s", fastest.Name, fastest.Address, fastest.Latency)
}

// logProbe 对一组主机并发探测可达性与延迟，按返回顺序打印前 `limit` 个结果。
//
// 参数 `label` 为日志前缀，用于区分主站、扩展站、MAC 主站与 MAC 扩展站；参数 `hosts` 为待探测的
// 主机信息列表；参数 `limit` 为单组最多打印的结果条数。探测超时固定为 1 秒；不可达主机单独打印
// 错误信息，便于观察候选地址质量。
func logProbe(label string, hosts []gotdx.HostInfo, limit int) {
	// 探测超时固定为 1 秒，超过该时长未响应的主机记为不可达。
	results := gotdx.ProbeHosts(hosts, time.Second)
	log.Printf("%s hosts=%d", label, len(hosts))

	for i, result := range results {
		if i >= limit {
			break
		}
		if result.Reachable {
			log.Printf("%s #%d %s %s latency=%s", label, i+1, result.Name, result.Address, result.Latency)
			continue
		}
		log.Printf("%s #%d %s %s error=%s", label, i+1, result.Name, result.Address, result.Error)
	}
}

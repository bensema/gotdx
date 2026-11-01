// Command ex_server_info 演示手动构造扩展行情客户端：先完成登录握手并打印服务器信息，
// 再分别查询当日分时与指定交易日历史分时，以美股 `TSLA` 为例。
//
// 运行示例：`go run ./examples/ex_server_info`
package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	hosts := exampleutil.ExHosts()
	// 首条地址作为主站单点，其余进入候选地址池；`WithTimeoutSec(6)` 将拨号与读写超时固定为 6 秒。
	// 构造时不拨号，连接在首次请求时按需建立。
	client := gotdx.NewEx(
		gotdx.WithExTCPAddress(hosts[0]),
		gotdx.WithExTCPAddressPool(hosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
	defer client.Disconnect()

	// 主动连接扩展行情服务器并完成登录握手；应答中的 `DateTime` 为登录时间，`ServerName` 为服务器名，
	// `IP` 为连接地址。
	login, err := client.ConnectEx()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("login time=%s server=%s ip=%s", login.DateTime, login.ServerName, login.IP)

	// 服务器信息查询无业务入参；`Delay` 为服务端延迟，`TimeNow` 为服务端当前时间，`ServerSign2` 为签名字段。
	info, err := client.GetExServerInfo()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("server_info name=%s version=%s delay=%d now=%s sign2=%s", info.ServerName, info.Version, info.Delay, info.TimeNow, info.ServerSign2)

	// 查询当日分时：`category` 为美股类别，`code` 为 `TSLA`；用 `min(10, len(tick.List))` 兜底截取前 10 点。
	tick, err := client.ExGetTickChart(types.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range tick.List[:min(10, len(tick.List))] {
		log.Printf("ex_tick time=%s price=%.2f avg=%.2f vol=%d", item.Time, item.Price, item.Avg, item.Vol)
	}

	// 查询历史分时：`date` 为交易日 `20260330`（`YYYYMMDD`），随后是美股类别与 `TSLA`；同样截取前 10 点。
	historyTick, err := client.ExGetHistoryTickChart(20260330, types.ExCategoryUSStock, "TSLA")
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range historyTick.List[:min(10, len(historyTick.List))] {
		log.Printf("ex_history_tick time=%s price=%.2f avg=%.2f vol=%d", item.Time, item.Price, item.Avg, item.Vol)
	}
}

// min 返回 `a`、`b` 中较小的整数，用于把截取条数限制在切片实际长度以内，避免越界。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

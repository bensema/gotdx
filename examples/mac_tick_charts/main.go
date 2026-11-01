// Command mac_tick_charts 演示通过 MAC 协议查询最近多个交易日的分时图及其逐时点价格并打印。
//
// 运行示例：`go run ./examples/mac_tick_charts`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// 市场 `types.MarketSZ`（深市），代码 `000001`；`queryDate` 传 0 表示取最新交易日作为基准日，
	// `days=3` 表示从基准日向前取 3 个交易日的分时图，服务端返回的多余交易日会按该值裁剪。
	reply, err := client.MACTickCharts(types.MarketSZ.Uint8(), "000001", 0, 3)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("mac_tick_charts code=%s name=%s time=%s close=%.2f days=%d",
		reply.Code, reply.Name, reply.DateTime, reply.Close, len(reply.Charts))

	for _, day := range reply.Charts {
		log.Printf("day=%s pre_close=%.2f ticks=%d", day.Date, day.PreClose, len(day.Ticks))
		// 单日分时点可能不足 3 个，`min(3, len(...))` 防止切片长度越界，仅打印前 3 个时点。
		for _, item := range day.Ticks[:min(3, len(day.Ticks))] {
			log.Printf("tick time=%s price=%.2f avg=%.2f vol=%d",
				item.Time, item.Price, item.Avg, item.Vol)
		}
	}
}

// min 返回整数 a 与 b 中较小的一个。
//
// 参数 a、b 为待比较的两个整数；返回值为二者较小值，用于在切片截断时把长度限制在有效范围内，避免越界。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

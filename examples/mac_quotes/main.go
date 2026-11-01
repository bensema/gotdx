// Command mac_quotes 演示通过 MAC 协议接口 `MACQuotes` 查询单只证券的实时报价及其分时图表数据并打印。
//
// 运行示例：`go run ./examples/mac_quotes`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// 市场参数 `types.MarketSZ` 表示深市，证券代码为 `000001`；`MACQuotes` 内部等价于
	// `MACQuotesWithDate(market, code, 0)`，日期传 0 表示取最新交易日。如需指定历史日期，可改用
	// `MACQuotesWithDate`，下一行已注释，仅作对照示例。
	reply, err := client.MACQuotes(types.MarketSZ.Uint8(), "000001")
	// reply, err := client.MACQuotesWithDate(types.MarketSZ.Uint8(), "000001", 20260418)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("mac_quotes code=%s name=%s time=%s close=%.2f pre_close=%.2f turnover=%.2f avg=%.2f chart_points=%d",
		reply.Code, reply.Name, reply.DateTime, reply.Close, reply.PreClose, reply.Turnover, reply.Avg, len(reply.ChartData))

	// 分时点可能不足 5 个，`min(5, len(...))` 取两者较小值，防止切片长度越界，仅打印前 5 个时点。
	for _, item := range reply.ChartData[:min(5, len(reply.ChartData))] {
		log.Printf("chart time=%s price=%.2f avg=%.2f vol=%d momentum=%.2f",
			item.Time, item.Price, item.Avg, item.Vol, item.Momentum)
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

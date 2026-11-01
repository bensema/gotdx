// Command mac_market_monitor 演示通过 MACMarketMonitor 获取市场异动监控数据，深市类别下从偏移 0 取前 20 条，
// 打印异动描述、数值与各原始字段。
//
// 运行示例：`go run ./examples/mac_market_monitor`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// 参数 `market` 为深市；`start` 为起始记录序号 0（偏移受 `uint16` 上界 65535 限制）；`count`
	// 为期望异动条数上限 20，接口内部自动分页。
	items, err := client.MACMarketMonitor(types.MarketSZ.Uint8(), 0, 20)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_market_monitor code=%s name=%s time=%s desc=%s value=%s type=%d raw=(v1=%d v2=%.2f v3=%.4f v4=%.2f)",
			item.Code, item.Name, item.Time, item.Desc, item.Value, item.UnusualType, item.V1, item.V2, item.V3, item.V4)
	}
}

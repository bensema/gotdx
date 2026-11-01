// Command stock_count 演示分别查询深市、沪市、北交所三个市场的证券总数并打印。
//
// 运行示例：`go run ./examples/stock_count`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	for _, market := range []struct {
		name   string
		market uint8
	}{
		{name: "SZ", market: types.MarketSZ.Uint8()},
		{name: "SH", market: types.MarketSH.Uint8()},
		{name: "BJ", market: types.MarketBJ.Uint8()},
	} {
		// `market` 分别取深市、沪市、北交所代码；`StockCount` 返回服务端统计的该市场证券总数。
		count, err := client.StockCount(market.market)
		if err != nil {
			log.Fatalf("%s count failed: %v", market.name, err)
		}
		log.Printf("market=%s count=%d", market.name, count)
	}
}
